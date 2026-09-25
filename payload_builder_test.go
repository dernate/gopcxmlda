package gopcxmlda

import (
	"encoding/xml"
	"math"
	"net/url"
	"strings"
	"testing"
	"time"
)

func testServer() *Server {
	u, _ := url.Parse("http://example.invalid/opcxmlda")
	return &Server{Url: u, LocaleID: "en-US"}
}

func TestBuildGetStatusPayloadEscapesAttributes(t *testing.T) {
	crh := `handle"with<special&chars`
	payload, err := buildGetStatusPayload(testServer(), "ns1", &crh)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(payload, crh) {
		t.Fatalf("expected ClientRequestHandle to be XML-escaped, got: %s", payload)
	}
	if !strings.Contains(payload, `ClientRequestHandle="handle&#34;with&lt;special&amp;chars"`) {
		t.Fatalf("expected escaped ClientRequestHandle attribute, got: %s", payload)
	}
	if !strings.Contains(payload, "<ns1:GetStatus") || !strings.Contains(payload, "</ns1:GetStatus>") {
		t.Fatalf("expected ns1:GetStatus element, got: %s", payload)
	}
}

func TestBuildReadPayloadEscapesItemNameAndOmitsEmptyPath(t *testing.T) {
	crh := "crh1"
	handles := []string{"h0"}
	items := []TItem{{ItemName: `My/Item&<Name>`}}
	payload, err := buildReadPayload(testServer(), &crh, &handles, "ns1", items, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload, `ItemName="My/Item&amp;&lt;Name&gt;"`) {
		t.Fatalf("expected escaped ItemName, got: %s", payload)
	}
	if strings.Contains(payload, "ItemPath=") {
		t.Fatalf("expected ItemPath attribute to be omitted when empty, got: %s", payload)
	}
	// ClientRequestHandle/LocaleID must end up in Options, not on <ns1:Read> itself:
	// per the specification's WSDL, <Read> carries no attributes of its own at all.
	if !strings.Contains(payload, `ClientRequestHandle="crh1"`) || !strings.Contains(payload, `LocaleID="en-US"`) {
		t.Fatalf("expected ClientRequestHandle/LocaleID in Options, got: %s", payload)
	}
	readElement := payload[strings.Index(payload, "<ns1:Read"):strings.Index(payload, "<ns1:Options")]
	if strings.Contains(readElement, "ClientRequestHandle") || strings.Contains(readElement, "LocaleID") {
		t.Fatalf("expected no attributes directly on <ns1:Read>, got: %s", readElement)
	}
}

func TestBuildReadPayloadRendersOptions(t *testing.T) {
	crh := "crh1"
	handles := []string{"h0"}
	items := []TItem{{ItemName: "Item1"}}
	options := map[string]interface{}{"ReturnItemTime": true}
	payload, err := buildReadPayload(testServer(), &crh, &handles, "ns1", items, options)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload, `ReturnItemTime="true"`) {
		t.Fatalf("expected rendered Options element, got: %s", payload)
	}
	if !strings.Contains(payload, `ClientRequestHandle="crh1"`) || !strings.Contains(payload, `LocaleID="en-US"`) {
		t.Fatalf("expected ClientRequestHandle/LocaleID merged into Options, got: %s", payload)
	}
}

func TestBuildWritePayloadArrayValue(t *testing.T) {
	crh := "crh1"
	handles := []string{"h0"}
	items := []TItem{{ItemName: "Item1", Value: TValue{Value: []int{1, 2, 3}}}}
	payload, err := buildWritePayload(testServer(), "ns1", items, &crh, &handles, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload, `xsi:type="ns1:ArrayOfInt"`) {
		t.Fatalf("expected ArrayOfInt type, got: %s", payload)
	}
	if !strings.Contains(payload, "<ns1:int>1</ns1:int><ns1:int>2</ns1:int><ns1:int>3</ns1:int>") {
		t.Fatalf("expected three nested int elements, got: %s", payload)
	}
	// ClientRequestHandle/LocaleID must end up in Options, not on <ns1:Write> itself.
	if !strings.Contains(payload, `ClientRequestHandle="crh1"`) || !strings.Contains(payload, `LocaleID="en-US"`) {
		t.Fatalf("expected ClientRequestHandle/LocaleID in Options, got: %s", payload)
	}
}

func TestBuildWritePayloadDoesNotMutateCallerOptions(t *testing.T) {
	crh := "crh1"
	handles := []string{"h0"}
	items := []TItem{{ItemName: "Item1", Value: TValue{Value: 1}}}
	options := map[string]interface{}{"ReturnErrorText": true}
	if _, err := buildWritePayload(testServer(), "ns1", items, &crh, &handles, options); err != nil {
		t.Fatal(err)
	}
	if _, ok := options["ClientRequestHandle"]; ok {
		t.Fatalf("expected caller options map to remain unmodified, got: %v", options)
	}
	if len(options) != 1 {
		t.Fatalf("expected caller options map to keep its original size, got: %v", options)
	}
}

// TestBuildWritePayloadNilValueReturnsErrorNotPanic pins down that an item with an
// unset (nil) Value.Value produces a clear error instead of panicking inside
// reflect/getOpcXmlDaType.
func TestBuildWritePayloadNilValueReturnsErrorNotPanic(t *testing.T) {
	crh := "crh1"
	handles := []string{"h0"}
	items := []TItem{{ItemName: "Item1"}} // Value.Value left nil
	if _, err := buildWritePayload(testServer(), "ns1", items, &crh, &handles, map[string]interface{}{}); err == nil {
		t.Fatal("expected an error for an item with an unset Value.Value")
	}
}

// TestBuildReadPayloadMismatchedHandlesReturnsErrorNotPanic pins down that a
// ClientItemHandles slice shorter than items produces a clear error instead of an
// index-out-of-range panic.
func TestBuildReadPayloadMismatchedHandlesReturnsErrorNotPanic(t *testing.T) {
	crh := "crh1"
	handles := []string{"h0"}                                  // only 1 handle
	items := []TItem{{ItemName: "Item1"}, {ItemName: "Item2"}} // 2 items
	if _, err := buildReadPayload(testServer(), &crh, &handles, "ns1", items, nil); err == nil {
		t.Fatal("expected an error for mismatched ClientItemHandles/items lengths")
	}
}

// TestBuildSubscribePayloadPreservesDeadBandFraction pins down that a fractional
// DeadBand is rendered with its fractional digits intact, rather than being
// truncated to a whole number (the previous "%.0f" formatting turned 2.5 into "2").
func TestBuildSubscribePayloadPreservesDeadBandFraction(t *testing.T) {
	crh := "crh1"
	handles := []string{"h0"}
	items := []TItem{{ItemName: "Item1", DeadBand: 2.5}}
	payload, err := buildSubscribePayload("ns1", items, &crh, &handles, false, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload, `Deadband="2.5"`) {
		t.Fatalf("expected Deadband=\"2.5\" to survive formatting, got: %s", payload)
	}
}

// TestBuildSubscribePayloadRendersClientRequestHandleInOptions pins down that
// ClientRequestHandle ends up in Options, not on <ns1:Subscribe> itself: per the
// specification's WSDL, <Subscribe>'s own attributes are only ReturnValuesOnReply and
// SubscriptionPingRate.
func TestBuildSubscribePayloadRendersClientRequestHandleInOptions(t *testing.T) {
	crh := "crh1"
	handles := []string{"h0"}
	items := []TItem{{ItemName: "Item1"}}
	payload, err := buildSubscribePayload("ns1", items, &crh, &handles, false, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload, `<ns1:Options ClientRequestHandle="crh1">`) {
		t.Fatalf("expected ClientRequestHandle in Options, got: %s", payload)
	}
	subscribeElement := payload[strings.Index(payload, "<ns1:Subscribe"):strings.Index(payload, "<ns1:Options")]
	if strings.Contains(subscribeElement, "ClientRequestHandle") {
		t.Fatalf("expected no ClientRequestHandle attribute directly on <ns1:Subscribe>, got: %s", subscribeElement)
	}
}

// TestValueUnmarshalXMLIsAttributeOrderIndependent pins down that xsi:type is found
// by name rather than assumed to be the first attribute, so a perfectly valid
// document with another attribute preceding it still decodes correctly.
func TestValueUnmarshalXMLIsAttributeOrderIndependent(t *testing.T) {
	doc := `<Root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:ns1="urn:x">` +
		`<Value id="1" xsi:type="ns1:int">42</Value></Root>`
	var w struct {
		V TValue `xml:"Value"`
	}
	if err := xml.Unmarshal([]byte(doc), &w); err != nil {
		t.Fatalf("expected decoding to succeed despite a preceding attribute, got: %v", err)
	}
	if w.V.Value != 42 || w.V.Type != "int" {
		t.Fatalf("expected Value=42 Type=int, got Value=%v Type=%s", w.V.Value, w.V.Type)
	}
}

func TestBuildSubscriptionCancelPayloadUsesNamespacePrefix(t *testing.T) {
	crh := "crh1"
	payload, err := buildSubscriptionCancelPayload("sub123", "ns1", &crh)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload, "<ns1:SubscriptionCancel ") || !strings.Contains(payload, "</ns1:SubscriptionCancel>") {
		t.Fatalf("expected ns1:SubscriptionCancel element, got: %s", payload)
	}
}

// TestBuildWritePayloadTypeQNames pins down that scalar values are typed with the
// XML Schema prefix (xsd:int) and only arrays with the OPC XML-DA namespace
// (ns1:ArrayOfInt): the OPC XML-DA schema defines no scalar types of its own.
func TestBuildWritePayloadTypeQNames(t *testing.T) {
	cases := []struct {
		value interface{}
		want  string
	}{
		{42, `xsi:type="xsd:int"`},
		{"text", `xsi:type="xsd:string"`},
		{true, `xsi:type="xsd:boolean"`},
		{2.5, `xsi:type="xsd:double"`},
		{time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC), `xsi:type="xsd:dateTime"`},
		{[]byte{1}, `xsi:type="xsd:base64Binary"`},
		{[]int{1}, `xsi:type="ns1:ArrayOfInt"`},
		{[]string{"a"}, `xsi:type="ns1:ArrayOfString"`},
	}
	for _, tc := range cases {
		crh, handles := "crh1", []string{"h0"}
		payload, err := buildWritePayload(testServer(), "ns1", []TItem{{ItemName: "Item1", Value: TValue{Value: tc.value}}},
			&crh, &handles, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(payload, tc.want) {
			t.Errorf("%T: expected %s, got: %s", tc.value, tc.want, payload)
		}
	}
}

// TestBuildWritePayloadLexicalForms pins down the XML Schema lexical form of values
// whose Go formatting differs from it.
func TestBuildWritePayloadLexicalForms(t *testing.T) {
	cases := []struct {
		value interface{}
		want  string
	}{
		{time.Date(2026, 9, 25, 15, 38, 6, 500000000, time.UTC), `>2026-09-25T15:38:06.5Z<`},
		{math.Inf(1), `>INF<`},
		{math.Inf(-1), `>-INF<`},
		{float32(math.Inf(1)), `>INF<`},
		{math.NaN(), `>NaN<`},
		{[]byte{1, 2, 3}, `>AQID<`},
		{[]float64{math.Inf(1), 1.5}, `<ns1:double>INF</ns1:double><ns1:double>1.5</ns1:double>`},
		{[]time.Time{time.Date(2026, 9, 25, 15, 38, 6, 0, time.UTC)}, `<ns1:dateTime>2026-09-25T15:38:06Z</ns1:dateTime>`},
	}
	for _, tc := range cases {
		crh, handles := "crh1", []string{"h0"}
		payload, err := buildWritePayload(testServer(), "ns1", []TItem{{ItemName: "Item1", Value: TValue{Value: tc.value}}},
			&crh, &handles, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(payload, tc.want) {
			t.Errorf("%#v: expected %s in: %s", tc.value, tc.want, payload)
		}
	}
}
