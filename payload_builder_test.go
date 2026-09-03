package gopcxmlda

import (
	"encoding/xml"
	"net/url"
	"strings"
	"testing"
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
	if strings.Contains(payload, "ns1:Options") {
		t.Fatalf("expected no Options element when no options given, got: %s", payload)
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
	if !strings.Contains(payload, `<ns1:Options ReturnItemTime="true"`) {
		t.Fatalf("expected rendered Options element, got: %s", payload)
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
