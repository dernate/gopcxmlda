package gopcxmlda

import (
	"context"
	"net/http"
	"sync"
	"testing"
)

// headerRecorder is a test server handler that records the Content-Type and
// SOAPAction headers of the last request it received and answers with an empty SOAP
// envelope. The response content doesn't matter here: these tests only look at what
// the client sent.
type headerRecorder struct {
	mu          sync.Mutex
	contentType string
	soapAction  string
}

func (h *headerRecorder) handle(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	h.contentType = r.Header.Get("Content-Type")
	h.soapAction = r.Header.Get("SOAPAction")
	h.mu.Unlock()
	w.Header().Set("Content-Type", DefaultContentType)
	_, _ = w.Write([]byte(soapEnvelope("")))
}

func (h *headerRecorder) last() (contentType, soapAction string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.contentType, h.soapAction
}

// callEveryMethod invokes each public Server method once and reports the operation
// name alongside, so a test can check the headers each request was sent with.
// Errors are ignored: the recorder answers with an empty envelope, which some methods
// legitimately reject, and only the request side is under test.
func callEveryMethod(s *Server, after func(operation string)) {
	ctx := context.Background()
	items := []TItem{{ItemName: "Item1", Value: TValue{Value: 1}}}
	newHandles := func() (*string, *[]string) {
		crh := ""
		var cih []string
		return &crh, &cih
	}

	crh, _ := newHandles()
	_, _ = s.GetStatus(ctx, crh, "")
	after("GetStatus")

	crh, cih := newHandles()
	_, _ = s.Read(ctx, items, crh, cih, "", nil)
	after("Read")

	crh, _ = newHandles()
	_, _ = s.Browse(ctx, "", crh, "", TBrowseOptions{})
	after("Browse")

	crh, cih = newHandles()
	_, _ = s.Write(ctx, items, crh, cih, "", nil)
	after("Write")

	crh, cih = newHandles()
	_, _ = s.Subscribe(ctx, items, crh, cih, "", false, 1000, nil)
	after("Subscribe")

	crh, _ = newHandles()
	_, _ = s.SubscriptionPolledRefresh(ctx, "sub1", 1000, "", crh, nil, TServerTime{UseClientTime: true})
	after("SubscriptionPolledRefresh")

	crh, _ = newHandles()
	_, _ = s.SubscriptionCancel(ctx, "sub1", "", crh)
	after("SubscriptionCancel")

	crh, _ = newHandles()
	_, _ = s.GetProperties(ctx, items, TPropertyOptions{}, crh, "")
	after("GetProperties")
}

// TestEveryRequestSendsSOAP11Headers pins down the HTTP binding of OPC XML-DA (SOAP
// 1.1) for every operation: Content-Type text/xml, not the SOAP 1.2 media type
// application/soap+xml that earlier versions sent and that strict servers reject with
// 415 Unsupported Media Type, and the operation's soapAction from the specification's
// WSDL as a quoted SOAPAction header.
func TestEveryRequestSendsSOAP11Headers(t *testing.T) {
	rec := &headerRecorder{}
	s, ts := newTestServer(t, rec.handle)
	defer ts.Close()

	called := 0
	callEveryMethod(s, func(operation string) {
		called++
		contentType, soapAction := rec.last()
		if contentType != "text/xml; charset=utf-8" {
			t.Errorf("%s: expected Content-Type %q, got %q", operation, "text/xml; charset=utf-8", contentType)
		}
		want := `"http://opcfoundation.org/webservices/XMLDA/1.0/` + operation + `"`
		if soapAction != want {
			t.Errorf("%s: expected SOAPAction %s, got %s", operation, want, soapAction)
		}
	})
	if called != len(soapActions) {
		t.Fatalf("expected every one of the %d operations in soapActions to be exercised, got %d",
			len(soapActions), called)
	}
}

// TestContentTypeOverride pins down that Server.ContentType replaces the default on
// every request, as the escape hatch for servers that insist on another media type.
func TestContentTypeOverride(t *testing.T) {
	rec := &headerRecorder{}
	s, ts := newTestServer(t, rec.handle)
	defer ts.Close()
	s.ContentType = "application/soap+xml"

	callEveryMethod(s, func(operation string) {
		if contentType, _ := rec.last(); contentType != "application/soap+xml" {
			t.Errorf("%s: expected overridden Content-Type %q, got %q", operation, "application/soap+xml", contentType)
		}
	})
}
