package gopcxmlda

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// checkHandleCount returns an error if the caller-supplied ClientItemHandles doesn't
// have exactly one handle per item. Without this check, a mismatched slice (e.g.
// reused from a previous, differently-sized batch) causes an index-out-of-range
// panic deep inside payload construction instead of a clear error.
func checkHandleCount(itemCount int, clientItemHandles []string) error {
	if len(clientItemHandles) != itemCount {
		return fmt.Errorf("gopcxmlda: got %d ClientItemHandles for %d items, lengths must match",
			len(clientItemHandles), itemCount)
	}
	return nil
}

// marshalPayload marshals body via encoding/xml and wraps the result in the SOAP
// envelope for namespace. Using encoding/xml for the body (instead of hand-built
// strings) guarantees that all element/attribute content is properly XML-escaped.
func marshalPayload(namespace string, body interface{}) (string, error) {
	bodyXML, err := xml.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("gopcxmlda: failed to marshal request body: %w", err)
	}

	var payload strings.Builder
	payload.WriteString(XmlVersion)
	buildHeader(&payload, namespace)
	payload.Write(bodyXML)
	payload.WriteString(Footer)

	return payload.String(), nil
}

// xmlOptions renders a <namespace:Options .../> element with an arbitrary, caller-
// supplied set of attributes (the OPC-XML-DA "Options" element differs per call).
type xmlOptions struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",any,attr"`
}

// newXmlOptions returns nil (and therefore renders nothing) when options is empty,
// matching the previous behavior of only emitting the Options element when needed.
func newXmlOptions(namespace, elementName string, options map[string]interface{}) *xmlOptions {
	if len(options) == 0 {
		return nil
	}
	return &xmlOptions{
		XMLName: xml.Name{Local: namespace + ":" + elementName},
		Attrs:   optionAttrs(options),
	}
}

func optionAttrs(options map[string]interface{}) []xml.Attr {
	attrs := make([]xml.Attr, 0, len(options))
	for key, value := range options {
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: key}, Value: fmt.Sprintf("%v", value)})
	}
	return attrs
}

// mergeOptions returns a new map containing options plus extra, without mutating
// either input map.
func mergeOptions(options map[string]interface{}, extra map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{}, len(options)+len(extra))
	for k, v := range options {
		merged[k] = v
	}
	for k, v := range extra {
		merged[k] = v
	}
	return merged
}

type xmlGetStatusRequest struct {
	XMLName             xml.Name
	LocaleID            string `xml:"LocaleID,attr"`
	ClientRequestHandle string `xml:"ClientRequestHandle,attr"`
}

func buildGetStatusPayload(s *Server, namespace string, ClientRequestHandle *string) (string, error) {
	body := xmlGetStatusRequest{
		XMLName:             xml.Name{Local: namespace + ":GetStatus"},
		LocaleID:            s.LocaleID,
		ClientRequestHandle: *ClientRequestHandle,
	}
	return marshalPayload(namespace, body)
}

// xmlReadItem renders a ReadRequestItem. MaxAge is a *int rather than an int because 0
// is a meaningful value for this attribute (it requests a device read), so it must be
// distinguishable from "not set" - which `omitempty` on a plain int could not do, as it
// would drop exactly that value.
type xmlReadItem struct {
	XMLName          xml.Name
	ItemName         string `xml:"ItemName,attr,omitempty"`
	ItemPath         string `xml:"ItemPath,attr,omitempty"`
	ClientItemHandle string `xml:"ClientItemHandle,attr"`
	MaxAge           *int   `xml:"MaxAge,attr,omitempty"`
}

// xmlItemList renders a ReadRequestItemList. Its MaxAge is the list-level default that
// applies to every item without a MaxAge of its own; see xmlReadItem on why it is a
// pointer.
type xmlItemList struct {
	XMLName xml.Name
	MaxAge  *int `xml:"MaxAge,attr,omitempty"`
	Items   []xmlReadItem
}

// xmlReadRequest renders a Read request. Per the specification's WSDL, the <Read>
// element itself carries no attributes at all - only an Options and an ItemList child
// element - so LocaleID and ClientRequestHandle have to be merged into Options
// (RequestOptions has both) rather than set directly on <Read>.
type xmlReadRequest struct {
	XMLName  xml.Name
	Options  *xmlOptions
	ItemList xmlItemList
}

func buildReadPayload(s *Server, ClientRequestHandle *string, ClientItemHandles *[]string, namespace string,
	items []TItem, options map[string]interface{}) (string, error) {
	if err := checkHandleCount(len(items), *ClientItemHandles); err != nil {
		return "", err
	}
	// MaxAge belongs on the item list, not in <Options>, so it is taken out of the
	// options map before the rest is rendered as RequestOptions attributes.
	listMaxAge, options, err := splitMaxAgeOption(options)
	if err != nil {
		return "", err
	}
	readItems := make([]xmlReadItem, len(items))
	for i, item := range items {
		if err := validateMaxAge(item.MaxAge, fmt.Sprintf("item %q", item.ItemName)); err != nil {
			return "", err
		}
		readItems[i] = xmlReadItem{
			XMLName:          xml.Name{Local: namespace + ":Items"},
			ItemName:         item.ItemName,
			ItemPath:         item.ItemPath,
			ClientItemHandle: (*ClientItemHandles)[i],
			MaxAge:           copyMaxAge(item.MaxAge),
		}
	}

	// ClientRequestHandle/LocaleID are options of the Read request itself; merge them
	// into a copy so the caller-supplied options map is never mutated as a side effect.
	mergedOpts := mergeOptions(options, map[string]interface{}{
		"ClientRequestHandle": *ClientRequestHandle,
		"LocaleID":            s.LocaleID,
	})

	body := xmlReadRequest{
		XMLName: xml.Name{Local: namespace + ":Read"},
		Options: newXmlOptions(namespace, "Options", mergedOpts),
		ItemList: xmlItemList{
			XMLName: xml.Name{Local: namespace + ":ItemList"},
			MaxAge:  listMaxAge,
			Items:   readItems,
		},
	}
	return marshalPayload(namespace, body)
}

type xmlBrowseRequest struct {
	XMLName             xml.Name
	LocaleID            string `xml:"LocaleID,attr"`
	ItemPath            string `xml:"ItemPath,attr"`
	ClientRequestHandle string `xml:"ClientRequestHandle,attr"`
	ItemName            string `xml:"ItemName,attr"`
	ContinuationPoint   string `xml:"ContinuationPoint,attr"`
	MaxElementsReturned int    `xml:"MaxElementsReturned,attr"`
	// BrowseFilter is an enumeration (all, branch, item) with default "all"; an empty
	// value is not valid and makes strict servers reject the whole request, so it is
	// omitted when unset and the server applies the default.
	BrowseFilter         string `xml:"BrowseFilter,attr,omitempty"`
	ElementNameFilter    string `xml:"ElementNameFilter,attr"`
	VendorFilter         string `xml:"VendorFilter,attr"`
	ReturnAllProperties  bool   `xml:"ReturnAllProperties,attr"`
	ReturnPropertyValues bool   `xml:"ReturnPropertyValues,attr"`
	ReturnErrorText      bool   `xml:"ReturnErrorText,attr"`
}

func buildBrowsePayload(s *Server, ClientRequestHandle *string, itemPath string, namespace string, options TBrowseOptions) (string, error) {
	body := xmlBrowseRequest{
		XMLName:              xml.Name{Local: namespace + ":Browse"},
		LocaleID:             s.LocaleID,
		ItemPath:             itemPath,
		ClientRequestHandle:  *ClientRequestHandle,
		ItemName:             options.ItemName,
		ContinuationPoint:    options.ContinuationPoint,
		MaxElementsReturned:  options.MaxElementsReturned,
		BrowseFilter:         options.BrowseFilter,
		ElementNameFilter:    options.ElementNameFilter,
		VendorFilter:         options.VendorFilter,
		ReturnAllProperties:  options.ReturnAllProperties,
		ReturnPropertyValues: options.ReturnPropertyValues,
		ReturnErrorText:      options.ReturnErrorText,
	}
	return marshalPayload(namespace, body)
}

// xmlWriteValue renders the <namespace:Value xsi:type="..."> element of a Write item,
// either as scalar text content or, for slice/array values, as one typed sub-element
// per entry (e.g. <namespace:int>1</namespace:int>). Implementing xml.Marshaler here
// (instead of building the element by hand) ensures the value content is escaped.
type xmlWriteValue struct {
	XMLName   xml.Name
	Namespace string
	Type      string
	Value     interface{}
}

func (v xmlWriteValue) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = v.XMLName
	start.Attr = []xml.Attr{{Name: xml.Name{Local: "xsi:type"}, Value: writeTypeQName(v.Namespace, v.Type)}}
	if err := e.EncodeToken(start); err != nil {
		return err
	}

	_, isBytes := v.Value.([]byte)
	if valueIsArrayOrSlice(v.Value) && !(isBytes && v.Type == "base64Binary") {
		rv := reflect.ValueOf(v.Value)
		for i := 0; i < rv.Len(); i++ {
			elem := rv.Index(i).Interface()
			elemType, err := getOpcXmlDaType(elem)
			if err != nil {
				return err
			}
			// Array elements are named after their type in the OPC XML-DA namespace,
			// e.g. <ns0:int> inside an ns0:ArrayOfInt.
			elemStart := xml.StartElement{Name: xml.Name{Local: fmt.Sprintf("%s:%s", v.Namespace, elemType)}}
			if err := e.EncodeElement(xml.CharData(formatXsdScalar(elem)), elemStart); err != nil {
				return err
			}
		}
	} else if err := e.EncodeToken(xml.CharData(formatXsdScalar(v.Value))); err != nil {
		return err
	}

	return e.EncodeToken(start.End())
}

// writeTypeQName returns the xsi:type of a written value. Only the ArrayOf* types are
// defined in the OPC XML-DA namespace; scalar values are XML Schema built-in types and
// have to be qualified with the xsd prefix declared on the envelope - "ns0:int" would
// name a type that doesn't exist in the OPC XML-DA schema.
func writeTypeQName(namespace, typ string) string {
	if strings.HasPrefix(typ, "ArrayOf") {
		return namespace + ":" + typ
	}
	return "xsd:" + typ
}

// formatXsdScalar renders a single value in the lexical form of its XML Schema type.
// fmt's %v is right for most types but not for these: time.Time would come out as
// "2006-01-02 15:04:05 +0000 UTC" instead of an xsd:dateTime, infinities as "+Inf"
// instead of "INF", and a []byte (base64Binary) as a list of numbers.
func formatXsdScalar(value interface{}) string {
	switch v := value.(type) {
	case time.Time:
		return v.Format(time.RFC3339Nano)
	case float32:
		return formatXsdFloat(float64(v), 32)
	case float64:
		return formatXsdFloat(v, 64)
	case []byte:
		return base64.StdEncoding.EncodeToString(v)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func formatXsdFloat(f float64, bitSize int) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "INF"
	case math.IsInf(f, -1):
		return "-INF"
	default:
		return strconv.FormatFloat(f, 'g', -1, bitSize)
	}
}

type xmlWriteItem struct {
	XMLName          xml.Name
	ItemName         string `xml:"ItemName,attr,omitempty"`
	ClientItemHandle string `xml:"ClientItemHandle,attr"`
	Value            xmlWriteValue
}

type xmlWriteItemList struct {
	XMLName xml.Name
	Items   []xmlWriteItem
}

type xmlWriteRequest struct {
	XMLName             xml.Name
	ReturnValuesOnReply bool `xml:"ReturnValuesOnReply,attr"`
	Options             *xmlOptions
	ItemList            xmlWriteItemList
}

func buildWritePayload(s *Server, namespace string, items []TItem, ClientRequestHandle *string,
	ClientItemHandles *[]string, options map[string]interface{}) (string, error) {
	if err := checkHandleCount(len(items), *ClientItemHandles); err != nil {
		return "", err
	}

	// make sure all items have a (correct) opc-xml-da type
	items, err := setOpcXmlDaTypes(items)
	if err != nil {
		return "", err
	}

	writeItems := make([]xmlWriteItem, len(items))
	for i, item := range items {
		writeItems[i] = xmlWriteItem{
			XMLName:          xml.Name{Local: namespace + ":Items"},
			ItemName:         item.ItemName,
			ClientItemHandle: (*ClientItemHandles)[i],
			Value: xmlWriteValue{
				XMLName:   xml.Name{Local: namespace + ":Value"},
				Namespace: namespace,
				Type:      item.Value.Type,
				Value:     item.Value.Value,
			},
		}
	}

	// ClientRequestHandle/LocaleID are options of the Write request itself; merge them
	// into a copy so the caller-supplied options map is never mutated as a side effect.
	mergedOpts := mergeOptions(options, map[string]interface{}{
		"ClientRequestHandle": *ClientRequestHandle,
		"LocaleID":            s.LocaleID,
	})

	body := xmlWriteRequest{
		XMLName:             xml.Name{Local: namespace + ":Write"},
		ReturnValuesOnReply: true,
		Options:             newXmlOptions(namespace, "Options", mergedOpts),
		ItemList: xmlWriteItemList{
			XMLName: xml.Name{Local: namespace + ":ItemList"},
			Items:   writeItems,
		},
	}
	return marshalPayload(namespace, body)
}

type xmlSubscribeItem struct {
	XMLName               xml.Name
	Type                  string `xml:"xsi:type,attr"`
	DeadBand              string `xml:"Deadband,attr"`
	RequestedSamplingRate uint   `xml:"RequestedSamplingRate,attr"`
	EnableBuffering       bool   `xml:"EnableBuffering,attr"`
	ItemName              string `xml:"ItemName,attr,omitempty"`
	ItemPath              string `xml:"ItemPath,attr,omitempty"`
	ClientItemHandle      string `xml:"ClientItemHandle,attr"`
}

type xmlSubscribeItemList struct {
	XMLName  xml.Name
	Type     string `xml:"xsi:type,attr"`
	ItemPath string `xml:"ItemPath,attr"`
	Items    []xmlSubscribeItem
}

// xmlSubscribeRequest renders a Subscribe request. Per the specification's WSDL, the
// <Subscribe> element's own attributes are only ReturnValuesOnReply and
// SubscriptionPingRate - ClientRequestHandle is not among them and has to be merged
// into Options (RequestOptions has it) instead of set directly on <Subscribe>.
type xmlSubscribeRequest struct {
	XMLName              xml.Name
	ReturnValuesOnReply  bool `xml:"ReturnValuesOnReply,attr"`
	SubscriptionPingRate uint `xml:"SubscriptionPingRate,attr"`
	Options              *xmlOptions
	ItemList             xmlSubscribeItemList
}

func buildSubscribePayload(namespace string, items []TItem, ClientRequestHandle *string, ClientItemHandles *[]string,
	returnValuesOnReply bool, subscriptionPingRate uint, options map[string]interface{}) (string, error) {
	if err := checkHandleCount(len(items), *ClientItemHandles); err != nil {
		return "", err
	}
	subscribeItems := make([]xmlSubscribeItem, len(items))
	for i, item := range items {
		subscribeItems[i] = xmlSubscribeItem{
			XMLName: xml.Name{Local: namespace + ":Items"},
			Type:    namespace + ":SubscribeRequestItem",
			// FormatFloat with precision -1 renders the minimal number of digits that
			// round-trips exactly (e.g. 2.5 -> "2.5"), unlike the previous "%.0f" which
			// truncated every DeadBand to a whole number (2.5 -> "2").
			DeadBand:              strconv.FormatFloat(item.DeadBand, 'f', -1, 64),
			RequestedSamplingRate: item.RequestedSamplingRate,
			EnableBuffering:       item.EnableBuffering,
			ItemName:              item.ItemName,
			ItemPath:              item.ItemPath,
			ClientItemHandle:      (*ClientItemHandles)[i],
		}
	}

	// ClientRequestHandle is an option of the Subscribe request itself; merge it into a
	// copy so the caller-supplied options map is never mutated as a side effect.
	mergedOpts := mergeOptions(options, map[string]interface{}{
		"ClientRequestHandle": *ClientRequestHandle,
	})

	body := xmlSubscribeRequest{
		XMLName:              xml.Name{Local: namespace + ":Subscribe"},
		ReturnValuesOnReply:  returnValuesOnReply,
		SubscriptionPingRate: subscriptionPingRate,
		Options:              newXmlOptions(namespace, "Options", mergedOpts),
		ItemList: xmlSubscribeItemList{
			XMLName:  xml.Name{Local: namespace + ":ItemList"},
			Type:     "SubscribeRequestItemList",
			ItemPath: "",
			Items:    subscribeItems,
		},
	}
	return marshalPayload(namespace, body)
}

type xmlSubscriptionCancelRequest struct {
	XMLName             xml.Name
	ServerSubHandle     string `xml:"ServerSubHandle,attr"`
	ClientRequestHandle string `xml:"ClientRequestHandle,attr"`
}

func buildSubscriptionCancelPayload(serverSubHandle string, namespace string, ClientRequestHandle *string) (string, error) {
	body := xmlSubscriptionCancelRequest{
		XMLName:             xml.Name{Local: namespace + ":SubscriptionCancel"},
		ServerSubHandle:     serverSubHandle,
		ClientRequestHandle: *ClientRequestHandle,
	}
	return marshalPayload(namespace, body)
}

type xmlServerSubHandles struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

type xmlSubscriptionPolledRefreshRequest struct {
	XMLName          xml.Name
	HoldTime         string `xml:"HoldTime,attr"`
	ReturnAllItems   bool   `xml:"ReturnAllItems,attr"`
	WaitTime         int    `xml:"WaitTime,attr"`
	Options          *xmlOptions
	ServerSubHandles xmlServerSubHandles
}

func buildSubscriptionPolledRefreshPayload(serverSubHandle string, namespace string, ClientRequestHandle *string,
	SubscriptionPingRate uint, options map[string]interface{}, ServerTime TServerTime) (string, error) {
	holdTime, err := calcHoldTime(SubscriptionPingRate, ServerTime)
	if err != nil {
		return "", err
	}

	mergedOpts := mergeOptions(options, map[string]interface{}{
		"ClientRequestHandle": *ClientRequestHandle,
	})

	body := xmlSubscriptionPolledRefreshRequest{
		XMLName:        xml.Name{Local: namespace + ":SubscriptionPolledRefresh"},
		HoldTime:       holdTime,
		ReturnAllItems: false,
		WaitTime:       500,
		Options:        newXmlOptions(namespace, "Options", mergedOpts),
		ServerSubHandles: xmlServerSubHandles{
			XMLName: xml.Name{Local: namespace + ":ServerSubHandles"},
			Value:   serverSubHandle,
		},
	}
	return marshalPayload(namespace, body)
}

func calcHoldTime(subscriptionPingRate uint, ServerTime TServerTime) (string, error) {
	if ServerTime.UseClientTime {
		now := time.Now()
		next := now.Add(time.Duration(subscriptionPingRate) * time.Millisecond)
		return next.Format(time.RFC3339), nil
	}
	next := ServerTime.ServerTime.Add(time.Duration(subscriptionPingRate) * time.Millisecond)
	return next.Format(time.RFC3339), nil
}

type xmlItemID struct {
	XMLName  xml.Name
	ItemName string `xml:"ItemName,attr"`
	ItemPath string `xml:"ItemPath,attr"`
}

type xmlPropertyName struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

type xmlGetPropertiesRequest struct {
	XMLName              xml.Name
	LocaleID             string `xml:"LocaleID,attr"`
	ClientRequestHandle  string `xml:"ClientRequestHandle,attr"`
	ReturnAllProperties  bool   `xml:"ReturnAllProperties,attr"`
	ReturnPropertyValues bool   `xml:"ReturnPropertyValues,attr"`
	ReturnErrorText      bool   `xml:"ReturnErrorText,attr"`
	ItemIDs              []xmlItemID
	PropertyNames        []xmlPropertyName
}

func buildGetPropertiesPayload(s *Server, ClientRequestHandle *string, namespace string, items []TItem,
	PropertyOptions TPropertyOptions) (string, error) {
	itemIDs := make([]xmlItemID, len(items))
	for i, item := range items {
		itemIDs[i] = xmlItemID{
			XMLName:  xml.Name{Local: namespace + ":ItemIDs"},
			ItemName: item.ItemName,
			ItemPath: item.ItemPath,
		}
	}

	propertyNames := make([]xmlPropertyName, len(PropertyOptions.PropertyNames))
	for i, name := range PropertyOptions.PropertyNames {
		propertyNames[i] = xmlPropertyName{
			XMLName: xml.Name{Local: namespace + ":PropertyNames"},
			Value:   name,
		}
	}

	body := xmlGetPropertiesRequest{
		XMLName:              xml.Name{Local: namespace + ":GetProperties"},
		LocaleID:             s.LocaleID,
		ClientRequestHandle:  *ClientRequestHandle,
		ReturnAllProperties:  PropertyOptions.ReturnAllProperties,
		ReturnPropertyValues: PropertyOptions.ReturnPropertyValues,
		ReturnErrorText:      PropertyOptions.ReturnErrorText,
		ItemIDs:              itemIDs,
		PropertyNames:        propertyNames,
	}
	return marshalPayload(namespace, body)
}
