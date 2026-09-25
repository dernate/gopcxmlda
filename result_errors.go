package gopcxmlda

import (
	"encoding/xml"
	"strings"
)

// OpcError is a single <Errors> element of a response: the verbose text the server
// sent for one ResultID.
type OpcError struct {
	// ID is the ResultID as sent. It is a QName that servers qualify differently -
	// unprefixed ("E_UNKNOWNITEMNAME") or with a prefix that may differ on every
	// element ("ns3:E_INVALIDITEMPATH") - so compare it with SameResultID.
	ID   string
	Text string
}

// ItemResult is an item of a response that the server marked with a ResultID - an
// error code (E_...) or a success code (S_..., e.g. S_CLAMP) - together with the
// verbose text for that ResultID, if the response contained one.
type ItemResult struct {
	ItemName         string
	ItemPath         string
	ClientItemHandle string // empty for GetProperties and Browse, whose results carry no handle
	// Property is the name of the item property the ResultID refers to, for results of
	// a single property (GetProperties, Browse); empty when it refers to the item.
	Property string
	ResultID string // as sent, see OpcError.ID
	Text     string // empty if the server sent no text (e.g. ReturnErrorText off)
}

// Failed reports whether the item's ResultID is an error code rather than a success
// code (success codes start with "S_").
func (r ItemResult) Failed() bool {
	return !strings.HasPrefix(resultIDLocalName(r.ResultID), "S_")
}

// SameResultID reports whether two ResultIDs denote the same code, comparing their
// local names: "E_FAIL", "ns1:E_FAIL" and "ns7:E_FAIL" are all the same ResultID.
func SameResultID(a, b string) bool {
	return resultIDLocalName(a) == resultIDLocalName(b)
}

func resultIDLocalName(resultID string) string {
	if _, local, ok := strings.Cut(resultID, ":"); ok {
		return local
	}
	return resultID
}

// UnmarshalXML decodes one <Errors> element. encoding/xml calls it once per element on
// the same OpcErrors, so repeated elements are collected in Entries instead of
// overwriting each other (which previously left the last ID paired with the texts of
// all elements).
func (e *OpcErrors) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var element struct {
		ID   string   `xml:"ID,attr"`
		Type string   `xml:"type,attr"`
		Text []string `xml:"Text"`
	}
	if err := d.DecodeElement(&element, &start); err != nil {
		return err
	}
	if len(e.Entries) == 0 {
		e.Id = element.ID
		e.Type = element.Type
	}
	e.Text = append(e.Text, element.Text...)
	e.Entries = append(e.Entries, OpcError{ID: element.ID, Text: strings.Join(element.Text, "\n")})
	return nil
}

// TextFor returns the text sent for resultID, matching IDs with SameResultID.
func (e OpcErrors) TextFor(resultID string) (string, bool) {
	for _, entry := range e.Entries {
		if SameResultID(entry.ID, resultID) {
			return entry.Text, true
		}
	}
	return "", false
}

// hasID reports whether any <Errors> element carried an ID, i.e. whether the response
// reported an error at all.
func (e OpcErrors) hasID() bool {
	for _, entry := range e.Entries {
		if entry.ID != "" {
			return true
		}
	}
	return e.Id != ""
}

func itemResultsOf(items []TItem, errs OpcErrors) []ItemResult {
	var results []ItemResult
	for _, item := range items {
		if item.Error == "" {
			continue
		}
		text, _ := errs.TextFor(item.Error)
		results = append(results, ItemResult{
			ItemName:         item.ItemName,
			ItemPath:         item.ItemPath,
			ClientItemHandle: item.ClientItemHandle,
			ResultID:         item.Error,
			Text:             text,
		})
	}
	return results
}

// ItemResults returns every item of the response that carries a ResultID, paired with
// its text from the response's <Errors>. It works whether or not the call returned an
// error: a server that isn't asked for error texts (ReturnErrorText) may send no
// <Errors> at all and mark failed items only by their ResultID.
func (t TRead) ItemResults() []ItemResult {
	return itemResultsOf(t.Response.ItemList.Items, t.Response.Errors)
}

// ItemResults: see TRead.ItemResults.
func (t TWrite) ItemResults() []ItemResult {
	return itemResultsOf(t.Response.ItemList.Items, t.Response.Errors)
}

// ItemResults: see TRead.ItemResults.
func (t TSubscribe) ItemResults() []ItemResult {
	items := make([]TItem, len(t.Response.ItemList.Items))
	for i, item := range t.Response.ItemList.Items {
		items[i] = item.ItemValue
	}
	return itemResultsOf(items, t.Response.Errors)
}

// ItemResults: see TRead.ItemResults.
func (t TSubscriptionPolledRefresh) ItemResults() []ItemResult {
	return itemResultsOf(t.Response.ItemList.Items, t.Response.Errors)
}

// ItemResults returns every property list and every single property of the response
// that carries a ResultID, paired with its text; see TRead.ItemResults.
func (t TGetProperties) ItemResults() []ItemResult {
	var results []ItemResult
	for _, list := range t.Response.PropertyList {
		if list.ResultId != "" {
			text, _ := t.Response.Errors.TextFor(list.ResultId)
			results = append(results, ItemResult{
				ItemName: list.ItemName,
				ItemPath: list.ItemPath,
				ResultID: list.ResultId,
				Text:     text,
			})
		}
		results = append(results, propertyResultsOf(list.ItemName, list.ItemPath, list.Properties, t.Response.Errors)...)
	}
	return results
}

// ItemResults returns every property of the browsed elements that carries a
// ResultID, paired with its text; see TRead.ItemResults.
func (t TBrowse) ItemResults() []ItemResult {
	var results []ItemResult
	for _, element := range t.Response.Elements {
		results = append(results, propertyResultsOf(element.ItemName, element.ItemPath, element.Properties, t.Response.Errors)...)
	}
	return results
}

func propertyResultsOf(itemName, itemPath string, properties []TProperties, errs OpcErrors) []ItemResult {
	var results []ItemResult
	for _, property := range properties {
		if property.ResultId == "" {
			continue
		}
		text, _ := errs.TextFor(property.ResultId)
		results = append(results, ItemResult{
			ItemName: itemName,
			ItemPath: itemPath,
			Property: property.Name,
			ResultID: property.ResultId,
			Text:     text,
		})
	}
	return results
}

func (t TRead) itemResults() []ItemResult                      { return t.ItemResults() }
func (t TWrite) itemResults() []ItemResult                     { return t.ItemResults() }
func (t TSubscribe) itemResults() []ItemResult                 { return t.ItemResults() }
func (t TSubscriptionPolledRefresh) itemResults() []ItemResult { return t.ItemResults() }
func (t TGetProperties) itemResults() []ItemResult             { return t.ItemResults() }
func (t TGetStatus) itemResults() []ItemResult                 { return nil }
func (t TBrowse) itemResults() []ItemResult                    { return t.ItemResults() }
func (t TSubscriptionCancel) itemResults() []ItemResult        { return nil }
