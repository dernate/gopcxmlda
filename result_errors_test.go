package gopcxmlda

import (
	"context"
	"encoding/xml"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func readWithFixture(t *testing.T, fixture string, itemCount int) (TRead, error) {
	t.Helper()
	s := fixtureServer(t, http.StatusOK, fixture)
	items := make([]TItem, itemCount)
	for i := range items {
		items[i] = TItem{ItemName: "x"}
	}
	crh, cih := "", []string(nil)
	return s.Read(context.Background(), items, &crh, &cih, "", nil)
}

// TestErrorsEntriesStayPairedWithTheirText is the regression test for several
// <Errors> elements collapsing into one: previously the last ID was paired with the
// texts of all elements.
func TestErrorsEntriesStayPairedWithTheirText(t *testing.T) {
	read, err := readWithFixture(t, "gsoap/read_errors.xml", 4)
	var opcErr *OpcResponseError
	if !errors.As(err, &opcErr) {
		t.Fatalf("expected an *OpcResponseError, got: %v", err)
	}

	wantEntries := []OpcError{
		{ID: "E_UNKNOWNITEMNAME", Text: "The item name is no longer available in the server address space"},
		{ID: "S_CLAMP", Text: "The value was accepted but clamped"},
		{ID: "E_WRITEONLY", Text: "The item is not readable"},
	}
	if !reflect.DeepEqual(read.Response.Errors.Entries, wantEntries) {
		t.Errorf("expected entries %+v, got %+v", wantEntries, read.Response.Errors.Entries)
	}
	if !reflect.DeepEqual(opcErr.Errors, wantEntries) {
		t.Errorf("expected the error to carry the entries, got %+v", opcErr.Errors)
	}
	// Backwards compatible fields: the first ID, and all texts in order.
	if opcErr.Id != "E_UNKNOWNITEMNAME" || len(opcErr.Text) != 3 || opcErr.Text[2] != "The item is not readable" {
		t.Errorf("unexpected Id/Text: %q %q", opcErr.Id, opcErr.Text)
	}
	if text, ok := read.Response.Errors.TextFor("ns9:E_WRITEONLY"); !ok || text != "The item is not readable" {
		t.Errorf("TextFor: got %q, %v", text, ok)
	}
	if _, ok := read.Response.Errors.TextFor("E_FAIL"); ok {
		t.Error("TextFor: expected no text for a ResultID that doesn't occur")
	}
}

// TestOpcResponseErrorNamesTheAffectedItems checks that the error says which items
// carry which ResultID and why, including success codes.
func TestOpcResponseErrorNamesTheAffectedItems(t *testing.T) {
	_, err := readWithFixture(t, "gsoap/read_errors.xml", 4)
	var opcErr *OpcResponseError
	if !errors.As(err, &opcErr) {
		t.Fatalf("expected an *OpcResponseError, got: %v", err)
	}
	want := []ItemResult{
		{ItemName: "Does/Not/Exist", ClientItemHandle: "h1", ResultID: "E_UNKNOWNITEMNAME",
			Text: "The item name is no longer available in the server address space"},
		{ItemName: "Plant/Setpoint", ClientItemHandle: "h2", ResultID: "S_CLAMP", Text: "The value was accepted but clamped"},
		{ItemName: "Secret", ItemPath: "Plant/", ClientItemHandle: "h3", ResultID: "E_WRITEONLY", Text: "The item is not readable"},
	}
	if !reflect.DeepEqual(opcErr.Items, want) {
		t.Fatalf("expected items %+v, got %+v", want, opcErr.Items)
	}
	if opcErr.Items[0].Failed() != true || opcErr.Items[1].Failed() != false || opcErr.Items[2].Failed() != true {
		t.Errorf("unexpected Failed() results for %+v", opcErr.Items)
	}

	msg := err.Error()
	for _, part := range []string{
		"Id: E_UNKNOWNITEMNAME, Text: [", // the form of earlier versions stays the prefix
		"errors: E_UNKNOWNITEMNAME (The item name is no longer available",
		"S_CLAMP (The value was accepted but clamped)",
		"items: Does/Not/Exist: E_UNKNOWNITEMNAME (The item name",
		"Plant/Secret: E_WRITEONLY (The item is not readable)",
	} {
		if !strings.Contains(msg, part) {
			t.Errorf("expected %q in error message: %s", part, msg)
		}
	}
}

// TestItemTextsMatchDespiteDifferentPrefixes covers the Axis dialect, which qualifies
// the same ResultID with a different prefix on every element.
func TestItemTextsMatchDespiteDifferentPrefixes(t *testing.T) {
	_, err := readWithFixture(t, "axis/read.xml", 3)
	var opcErr *OpcResponseError
	if !errors.As(err, &opcErr) {
		t.Fatalf("expected an *OpcResponseError, got: %v", err)
	}
	if len(opcErr.Items) != 2 {
		t.Fatalf("expected 2 failed items, got %+v", opcErr.Items)
	}
	for _, item := range opcErr.Items {
		if item.Text != "Invalid node" || !SameResultID(item.ResultID, "E_INVALIDITEMPATH") {
			t.Errorf("expected text %q for %s, got %+v", "Invalid node", item.ItemName, item)
		}
	}
}

func TestItemResultsOfOtherOperations(t *testing.T) {
	t.Run("Subscribe", func(t *testing.T) {
		s := fixtureServer(t, http.StatusOK, "gsoap/subscribe.xml")
		crh, cih := "", []string(nil)
		sub, err := s.Subscribe(context.Background(), []TItem{{ItemName: "a"}, {ItemName: "b"}}, &crh, &cih, "", true, 1000, nil)
		var opcErr *OpcResponseError
		if !errors.As(err, &opcErr) || len(opcErr.Items) != 1 || opcErr.Items[0].ClientItemHandle != "s1" {
			t.Fatalf("expected the failed subscribe item in the error, got %v", err)
		}
		if !reflect.DeepEqual(sub.ItemResults(), opcErr.Items) {
			t.Errorf("expected ItemResults() to match the error's items, got %+v", sub.ItemResults())
		}
	})
	t.Run("GetProperties", func(t *testing.T) {
		s := fixtureServer(t, http.StatusOK, "axis/getproperties.xml")
		crh := ""
		_, err := s.GetProperties(context.Background(), []TItem{{ItemName: "Plant/Missing"}}, TPropertyOptions{}, &crh, "")
		var opcErr *OpcResponseError
		if !errors.As(err, &opcErr) || len(opcErr.Items) != 1 {
			t.Fatalf("expected one failed property list in the error, got %v", err)
		}
		if item := opcErr.Items[0]; item.ItemName != "Plant/Missing" || item.Text != "Unknown node" {
			t.Errorf("unexpected item: %+v", item)
		}
	})
}

// TestItemResultsWithoutErrorsElement covers a server that marks failed items only by
// their ResultID and sends no <Errors> (e.g. ReturnErrorText off): the call returns no
// error - as before - but ItemResults still lists the failed items.
func TestItemResultsWithoutErrorsElement(t *testing.T) {
	doc := soapEnvelope(`<ReadResponse><RItemList>` +
		`<Items ItemName="ok"><Value xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xsd:int">1</Value></Items>` +
		`<Items ItemName="missing" ClientItemHandle="h1" ResultID="ns1:E_INVALIDITEMPATH"/>` +
		`</RItemList></ReadResponse>`)
	var read TRead
	if err := xml.Unmarshal([]byte(doc), &read); err != nil {
		t.Fatal(err)
	}
	want := []ItemResult{{ItemName: "missing", ClientItemHandle: "h1", ResultID: "ns1:E_INVALIDITEMPATH"}}
	if got := read.ItemResults(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %+v, got %+v", want, got)
	}
	if read.Response.Errors.hasID() {
		t.Fatal("expected no response-level errors")
	}
}

func TestResultIDHelpers(t *testing.T) {
	if !SameResultID("E_FAIL", "ns3:E_FAIL") || !SameResultID("ns1:E_FAIL", "ns2:E_FAIL") || SameResultID("E_FAIL", "E_BUSY") {
		t.Error("unexpected SameResultID results")
	}
	for id, failed := range map[string]bool{"E_FAIL": true, "ns1:E_FAIL": true, "S_CLAMP": false, "ns2:S_UNSUPPORTEDRATE": false} {
		if got := (ItemResult{ResultID: id}).Failed(); got != failed {
			t.Errorf("Failed() for %s: expected %v, got %v", id, failed, got)
		}
	}
}

// TestPropertyLevelResultsAreAttributed covers a ResultID on a single property (here:
// optional properties the item doesn't have) - the error must name item and property.
func TestPropertyLevelResultsAreAttributed(t *testing.T) {
	s := fixtureServer(t, http.StatusOK, "axis/getproperties_pid.xml")
	crh := ""
	props, err := s.GetProperties(context.Background(), []TItem{{ItemName: "Plant/Voltage"}},
		TPropertyOptions{ReturnAllProperties: true}, &crh, "")
	var opcErr *OpcResponseError
	if !errors.As(err, &opcErr) {
		t.Fatalf("expected an *OpcResponseError, got: %v", err)
	}
	want := []ItemResult{
		{ItemName: "Plant/Voltage", Property: "lowEU", ResultID: "ns2:E_INVALIDPID", Text: "Invalid property for item"},
		{ItemName: "Plant/Voltage", Property: "highEU", ResultID: "ns3:E_INVALIDPID", Text: "Invalid property for item"},
	}
	if !reflect.DeepEqual(opcErr.Items, want) {
		t.Fatalf("expected %+v, got %+v", want, opcErr.Items)
	}
	if !strings.Contains(err.Error(), "Plant/Voltage property lowEU: ns2:E_INVALIDPID (Invalid property for item)") {
		t.Errorf("expected item and property in the message, got: %v", err)
	}
	// The properties that could be returned are still there.
	list := props.Response.PropertyList[0]
	if len(list.Properties) != 5 || list.Properties[1].Value.Value != float32(10.25) || list.Properties[3].ResultId == "" {
		t.Errorf("unexpected properties: %+v", list.Properties)
	}
}

func TestBrowsePropertyResults(t *testing.T) {
	doc := soapEnvelope(`<BrowseResponse>` +
		`<Elements Name="Voltage" ItemName="Plant/Voltage" IsItem="true">` +
		`<Properties Name="value"><Value xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xsd:float">1</Value></Properties>` +
		`<Properties Name="lowEU" ResultID="E_INVALIDPID"/>` +
		`</Elements>` +
		`<Errors ID="E_INVALIDPID"><Text>Invalid property for item</Text></Errors>` +
		`</BrowseResponse>`)
	var browse TBrowse
	if err := xml.Unmarshal([]byte(doc), &browse); err != nil {
		t.Fatal(err)
	}
	want := []ItemResult{{ItemName: "Plant/Voltage", Property: "lowEU", ResultID: "E_INVALIDPID", Text: "Invalid property for item"}}
	if got := browse.ItemResults(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %+v, got %+v", want, got)
	}
}
