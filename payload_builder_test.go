package gopcxmlda

import (
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
