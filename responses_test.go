package gopcxmlda

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// These tests replay the fixtures in testdata/responses (see the README there) through
// the real request/response path of every public method, so response parsing is
// covered for both server dialects without a live server.

// fixtureServer returns a Server whose endpoint answers every request with the given
// fixture file and HTTP status.
func fixtureServer(t *testing.T, status int, fixture string) *Server {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", "responses", fixture))
	if err != nil {
		t.Fatal(err)
	}
	s, ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", DefaultContentType)
		w.WriteHeader(status)
		_, _ = w.Write(body)
	})
	t.Cleanup(ts.Close)
	return s
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

// assertResultID compares a ResultID/error QName by its local part. Servers qualify it
// differently - unprefixed (gSOAP) or with a per-element prefix such as "ns2:" (Axis) -
// and the local part is what identifies the error.
func assertResultID(t *testing.T, what, got, wantLocal string) {
	t.Helper()
	if got != wantLocal && !strings.HasSuffix(got, ":"+wantLocal) {
		t.Errorf("%s: expected ResultID %s, got %q", what, wantLocal, got)
	}
}

func assertValue(t *testing.T, what string, got TValue, wantType string, want interface{}) {
	t.Helper()
	if got.Type != wantType {
		t.Errorf("%s: expected Type %q, got %q", what, wantType, got.Type)
	}
	if !reflect.DeepEqual(got.Value, want) {
		t.Errorf("%s: expected Value %#v, got %#v", what, want, got.Value)
	}
}

func TestResponseGetStatus(t *testing.T) {
	cases := []struct {
		fixture, locale, version, info, vendor, startTime string
		replyTime                                         string
	}{
		{"gsoap/getstatus.xml", "en-us", "V1.00", "The server is running normally.",
			"Example OPC XML-DA Server", "2026-09-04T13:05:53.164+00:00", "2026-09-25T17:51:00.376+00:00"},
		{"axis/getstatus.xml", "en", "1.3.17", "Server is open for communication",
			"Example OPC XML-DA Gateway", "2026-08-04T06:35:57.157Z", "2026-09-25T15:38:10.084Z"},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			s := fixtureServer(t, http.StatusOK, tc.fixture)
			crh := ""
			status, err := s.GetStatus(context.Background(), &crh, "")
			if err != nil {
				t.Fatal(err)
			}
			result := status.Response.Result
			if result.ServerState != "running" || result.RevisedLocaleID != tc.locale || result.ClientRequestHandle != "crh1" {
				t.Errorf("unexpected result: %+v", result)
			}
			if !result.ReplyTime.Equal(mustTime(t, tc.replyTime)) {
				t.Errorf("expected ReplyTime %s, got %s", tc.replyTime, result.ReplyTime)
			}
			st := status.Response.Status
			if st.ProductVersion != tc.version || st.StatusInfo != tc.info || st.VendorInfo != tc.vendor ||
				st.StartTime != tc.startTime || st.SupportedInterfaceVersions != "XML_DA_Version_1_0" {
				t.Errorf("unexpected status: %+v", st)
			}
		})
	}
}

// TestResponseReadGSOAP covers scalar and array values, quality, timestamps and a
// per-item error. The response-level <Errors> element (the error text for the item's
// ResultID) is surfaced as an *OpcResponseError - and the parsed items must still be
// returned alongside it.
func TestResponseReadGSOAP(t *testing.T) {
	s := fixtureServer(t, http.StatusOK, "gsoap/read.xml")
	items := []TItem{{ItemName: "Plant/Power"}, {ItemName: "Plant/Report"}, {ItemName: "Plant/Status"}, {ItemName: "Does/Not/Exist"}}
	crh, cih := "", []string(nil)
	read, err := s.Read(context.Background(), items, &crh, &cih, "", nil)

	var opcErr *OpcResponseError
	if !errors.As(err, &opcErr) {
		t.Fatalf("expected an *OpcResponseError for the item-level error, got: %v", err)
	}
	assertResultID(t, "response error", opcErr.Id, "E_UNKNOWNITEMNAME")
	if len(opcErr.Text) != 1 || !strings.Contains(opcErr.Text[0], "no longer available") {
		t.Errorf("unexpected error text: %q", opcErr.Text)
	}

	got := read.Response.ItemList.Items
	if len(got) != 4 {
		t.Fatalf("expected 4 items to be returned alongside the error, got %d", len(got))
	}
	assertValue(t, "int item", got[0].Value, "int", 74)
	if got[0].Value.Namespace != "xsd" {
		t.Errorf("expected Namespace xsd, got %q", got[0].Value.Namespace)
	}
	if got[0].Quality != (TQuality{VendorField: "0", LimitField: "none", QualityField: "good"}) {
		t.Errorf("unexpected quality: %+v", got[0].Quality)
	}
	if !got[0].Timestamp.Equal(mustTime(t, "2026-09-25T15:38:06.025+00:00")) {
		t.Errorf("unexpected timestamp: %s", got[0].Timestamp)
	}
	if got[0].ClientItemHandle != "h0" || got[0].ItemName != "Plant/Power" || got[0].Error != "" {
		t.Errorf("unexpected item attributes: %+v", got[0])
	}
	assertValue(t, "double array", got[1].Value, "ArrayOfDouble",
		[]interface{}{10.0, 3.8999999999999999, -275.0, 65.534999999999997})
	assertValue(t, "unsignedShort array", got[2].Value, "ArrayOfUnsignedShort",
		[]interface{}{uint16(0), uint16(2), uint16(65535)})
	assertResultID(t, "unknown item", got[3].Error, "E_UNKNOWNITEMNAME")
	if got[3].Value.Value != nil {
		t.Errorf("expected no value for a failed item, got %#v", got[3].Value.Value)
	}
}

// TestResponseReadAxis covers the Axis dialect: self-closing items, no xsi:type on
// anything but <Value>, and ResultIDs qualified with a different, locally declared
// prefix on every item.
func TestResponseReadAxis(t *testing.T) {
	s := fixtureServer(t, http.StatusOK, "axis/read.xml")
	items := []TItem{{ItemName: "Plant/Power"}, {ItemName: "Plant/Missing"}, {ItemName: "Plant/AlsoMissing"}}
	crh, cih := "", []string(nil)
	read, err := s.Read(context.Background(), items, &crh, &cih, "", nil)

	var opcErr *OpcResponseError
	if !errors.As(err, &opcErr) {
		t.Fatalf("expected an *OpcResponseError, got: %v", err)
	}
	assertResultID(t, "response error", opcErr.Id, "E_INVALIDITEMPATH")

	got := read.Response.ItemList.Items
	if len(got) != 3 {
		t.Fatalf("expected 3 items, got %d", len(got))
	}
	assertValue(t, "double item", got[0].Value, "double", 1234.5)
	if got[0].Quality.QualityField != "good" {
		t.Errorf("unexpected quality: %+v", got[0].Quality)
	}
	assertResultID(t, "item 1", got[1].Error, "E_INVALIDITEMPATH")
	assertResultID(t, "item 2", got[2].Error, "E_INVALIDITEMPATH")
}

func TestResponseBrowseGSOAP(t *testing.T) {
	s := fixtureServer(t, http.StatusOK, "gsoap/browse.xml")
	crh := ""
	browse, err := s.Browse(context.Background(), "Plant", &crh, "", TBrowseOptions{ReturnAllProperties: true})
	if err != nil {
		t.Fatal(err)
	}
	r := browse.Response
	if r.MoreElements != "true" || r.ContinuationPoint != "191577240" {
		t.Errorf("unexpected paging attributes: MoreElements=%q ContinuationPoint=%q", r.MoreElements, r.ContinuationPoint)
	}
	want := []TBrowseElement{
		{HasChildren: false, IsItem: true, Name: "SerialNo", ItemName: "Plant/SerialNo"},
		{HasChildren: true, IsItem: false, Name: "Log", ItemName: "Plant/Log"},
	}
	if !reflect.DeepEqual(r.Elements, want) {
		t.Errorf("expected elements %+v, got %+v", want, r.Elements)
	}
}

// TestResponseBrowseAxisFault covers a SOAP fault delivered with HTTP 500: both the
// fault and the unexpected status must be detectable in the returned error.
func TestResponseBrowseAxisFault(t *testing.T) {
	s := fixtureServer(t, http.StatusInternalServerError, "axis/browse_fault.xml")
	crh := ""
	_, err := s.Browse(context.Background(), "Plant", &crh, "", TBrowseOptions{})

	var fault *SoapFaultError
	if !errors.As(err, &fault) {
		t.Fatalf("expected a *SoapFaultError, got: %v", err)
	}
	if fault.FaultCode != "soapenv:Server.userException" || fault.FaultString != "java.lang.IllegalArgumentException" {
		t.Errorf("unexpected fault: %+v", fault)
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected the HTTP status to be part of the error, got: %v", err)
	}
}

func TestResponseGetPropertiesGSOAP(t *testing.T) {
	s := fixtureServer(t, http.StatusOK, "gsoap/getproperties.xml")
	crh := ""
	props, err := s.GetProperties(context.Background(), []TItem{{ItemName: "Plant/Report"}},
		TPropertyOptions{ReturnAllProperties: true, ReturnPropertyValues: true}, &crh, "")
	if err != nil {
		t.Fatal(err)
	}
	lists := props.Response.PropertyList
	if len(lists) != 1 || lists[0].ItemName != "Plant/Report" {
		t.Fatalf("unexpected property lists: %+v", lists)
	}
	byName := map[string]TProperties{}
	for _, p := range lists[0].Properties {
		byName[p.Name] = p
	}
	if len(byName) != 6 {
		t.Fatalf("expected 6 properties, got %d: %+v", len(byName), lists[0].Properties)
	}
	assertValue(t, "accessRights", byName["ns1:accessRights"].Value, "string", "readable")
	assertValue(t, "dataType", byName["ns1:dataType"].Value, "QName", "ArrayOfDouble")
	assertValue(t, "quality", byName["ns1:quality"].Value, "OPCQuality",
		TQuality{VendorField: "0", LimitField: "none", QualityField: "good"})
	assertValue(t, "scanRate", byName["ns1:scanRate"].Value, "float", float32(0))
	assertValue(t, "timestamp", byName["ns1:timestamp"].Value, "dateTime", mustTime(t, "2026-09-25T15:30:00.000+00:00"))
	assertValue(t, "value", byName["ns1:value"].Value, "ArrayOfDouble", []interface{}{10.0, 2.5})
	if byName["ns1:scanRate"].Description == "" || byName["ns1:value"].ItemPath != "Plant/" {
		t.Errorf("unexpected property attributes: %+v", byName["ns1:value"])
	}
}

func TestResponseGetPropertiesAxisUnknownItem(t *testing.T) {
	s := fixtureServer(t, http.StatusOK, "axis/getproperties.xml")
	crh := ""
	props, err := s.GetProperties(context.Background(), []TItem{{ItemName: "Plant/Missing"}}, TPropertyOptions{}, &crh, "")
	var opcErr *OpcResponseError
	if !errors.As(err, &opcErr) {
		t.Fatalf("expected an *OpcResponseError, got: %v", err)
	}
	lists := props.Response.PropertyList
	if len(lists) != 1 || len(lists[0].Properties) != 0 {
		t.Fatalf("expected one empty property list, got %+v", lists)
	}
	assertResultID(t, "property list", lists[0].ResultId, "E_UNKNOWNITEMPATH")
}

// TestResponseSubscribeKeepsHandleDespiteItemError pins down that a subscription
// which the server did create is returned - handle included - even when one item
// failed and the call therefore returns an error. The caller needs the handle to cancel
// the subscription again.
func TestResponseSubscribeKeepsHandleDespiteItemError(t *testing.T) {
	s := fixtureServer(t, http.StatusOK, "gsoap/subscribe.xml")
	items := []TItem{{ItemName: "Plant/Power"}, {ItemName: "Does/Not/Exist"}}
	crh, cih := "", []string(nil)
	sub, err := s.Subscribe(context.Background(), items, &crh, &cih, "", true, 2000, nil)

	var opcErr *OpcResponseError
	if !errors.As(err, &opcErr) {
		t.Fatalf("expected an *OpcResponseError for the failed item, got: %v", err)
	}
	r := sub.Response
	if r.ServerSubHandle != "191915368" {
		t.Fatalf("expected the ServerSubHandle to be returned alongside the error, got %q", r.ServerSubHandle)
	}
	if len(r.ItemList.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(r.ItemList.Items))
	}
	first := r.ItemList.Items[0]
	if first.RevisedSamplingRate != 2500 || first.ItemValue.ClientItemHandle != "s0" {
		t.Errorf("unexpected first item: %+v", first)
	}
	assertValue(t, "first item", first.ItemValue.Value, "int", 74)
	assertResultID(t, "second item", r.ItemList.Items[1].ItemValue.Error, "E_UNKNOWNITEMNAME")
}

func TestResponseSubscribeAxisWithoutHandle(t *testing.T) {
	s := fixtureServer(t, http.StatusOK, "axis/subscribe_nohandle.xml")
	items := []TItem{{ItemName: "Plant/Missing"}}
	crh, cih := "", []string(nil)
	sub, err := s.Subscribe(context.Background(), items, &crh, &cih, "", true, 2000, nil)
	if err != nil {
		t.Fatal(err)
	}
	r := sub.Response
	if r.ServerSubHandle != "" {
		t.Errorf("expected no ServerSubHandle, got %q", r.ServerSubHandle)
	}
	if r.ItemList.RevisedSamplingRate != 1000 || len(r.ItemList.Items) != 1 {
		t.Fatalf("unexpected item list: %+v", r.ItemList)
	}
	assertResultID(t, "item", r.ItemList.Items[0].ItemValue.Error, "E_INVALIDITEMPATH")
}

func TestResponseSubscriptionPolledRefresh(t *testing.T) {
	refresh := func(t *testing.T, fixture string) (TSubscriptionPolledRefresh, error) {
		s := fixtureServer(t, http.StatusOK, fixture)
		crh := ""
		return s.SubscriptionPolledRefresh(context.Background(), "191915368", 1000, "", &crh, nil,
			TServerTime{UseClientTime: true})
	}

	t.Run("with changes", func(t *testing.T) {
		r, err := refresh(t, "gsoap/polledrefresh.xml")
		if err != nil {
			t.Fatal(err)
		}
		list := r.Response.ItemList
		if list.SubscriptionHandle != "191915368" || len(list.Items) != 1 {
			t.Fatalf("unexpected item list: %+v", list)
		}
		assertValue(t, "item", list.Items[0].Value, "int", 75)
		if r.Response.DataBufferOverflow {
			t.Error("expected DataBufferOverflow=false")
		}
	})

	t.Run("without changes", func(t *testing.T) {
		r, err := refresh(t, "gsoap/polledrefresh_empty.xml")
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Response.ItemList.Items) != 0 || len(r.Response.InvalidServerSubHandles) != 0 {
			t.Fatalf("expected an empty refresh, got %+v", r.Response)
		}
	})

	t.Run("invalid handle", func(t *testing.T) {
		_, err := refresh(t, "gsoap/polledrefresh_invalid.xml")
		var invalid *InvalidServerSubHandlesError
		if !errors.As(err, &invalid) {
			t.Fatalf("expected an *InvalidServerSubHandlesError, got: %v", err)
		}
		if !reflect.DeepEqual(invalid.Handles, []string{"invalid-handle"}) {
			t.Errorf("unexpected handles: %q", invalid.Handles)
		}
	})
}

func TestResponseSubscriptionCancel(t *testing.T) {
	s := fixtureServer(t, http.StatusOK, "gsoap/cancel.xml")
	crh := ""
	ok, err := s.SubscriptionCancel(context.Background(), "191915368", "", &crh)
	if err != nil || !ok {
		t.Fatalf("expected a successful cancel, got ok=%v err=%v", ok, err)
	}
}

func TestResponseWrite(t *testing.T) {
	s := fixtureServer(t, http.StatusOK, "gsoap/write.xml")
	items := []TItem{{ItemName: "Plant/Setpoint", Value: TValue{Value: []int{0, 0, 0}}}}
	crh, cih := "", []string(nil)
	w, err := s.Write(context.Background(), items, &crh, &cih, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	got := w.Response.ItemList.Items
	if len(got) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got))
	}
	assertValue(t, "written item", got[0].Value, "ArrayOfInt", []interface{}{0, 0, 0})
}
