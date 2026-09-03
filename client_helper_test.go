package gopcxmlda

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) (*Server, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(handler)
	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	return &Server{Url: u}, ts
}

func soapEnvelope(body string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>` +
		`<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/">` +
		`<SOAP-ENV:Body>` + body + `</SOAP-ENV:Body>` +
		`</SOAP-ENV:Envelope>`
}

func TestSendDefaultsTimeoutAndReusesClient(t *testing.T) {
	s, ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/soap+xml")
		_, _ = w.Write([]byte(soapEnvelope(`<GetStatusResponse></GetStatusResponse>`)))
	})
	defer ts.Close()

	if s.Timeout != 0 {
		t.Fatalf("expected zero Timeout before first request, got %v", s.Timeout)
	}

	crh := ""
	if _, err := s.GetStatus(context.Background(), &crh, ""); err != nil {
		t.Fatal(err)
	}
	if s.Timeout != DefaultTimeout {
		t.Fatalf("expected Timeout to default to %v (not 10ns), got %v", DefaultTimeout, s.Timeout)
	}
	if s.Client == nil {
		t.Fatal("expected Client to be initialized after the first request")
	}
	firstClient := s.Client

	crh2 := ""
	if _, err := s.GetStatus(context.Background(), &crh2, ""); err != nil {
		t.Fatal(err)
	}
	if s.Client != firstClient {
		t.Fatal("expected the same *http.Client to be reused across requests")
	}
}

func TestSendRejectsNilURL(t *testing.T) {
	s := &Server{}
	if _, err := send(context.Background(), s, "", "GetStatus"); err == nil {
		t.Fatal("expected an error for a Server with a nil Url")
	}
}

func TestSubscriptionCancelReturnsFalseOnFault(t *testing.T) {
	s, ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/soap+xml")
		_, _ = w.Write([]byte(soapEnvelope(
			`<SOAP-ENV:Fault><faultcode>Client</faultcode><faultstring>bad request</faultstring></SOAP-ENV:Fault>`,
		)))
	})
	defer ts.Close()

	crh := ""
	ok, err := s.SubscriptionCancel(context.Background(), "sub123", "", &crh)
	if err == nil {
		t.Fatal("expected an error when the server returns a SOAP fault")
	}
	if ok {
		t.Fatal("expected SubscriptionCancel to report failure when the server returns a SOAP fault")
	}
}

// TestConcurrentGetStatusIsRaceFree pins down that a single *Server can be used from
// multiple goroutines concurrently without racing on the lazy Client/Timeout
// initialization in send(). Run with -race to verify.
func TestConcurrentGetStatusIsRaceFree(t *testing.T) {
	s, ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/soap+xml")
		_, _ = w.Write([]byte(soapEnvelope(`<GetStatusResponse></GetStatusResponse>`)))
	})
	defer ts.Close()

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			crh := ""
			if _, err := s.GetStatus(context.Background(), &crh, ""); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}

// TestSoapFaultIsErrorsAsDetectable pins down that a SOAP fault surfaces as a
// *SoapFaultError that callers can extract with errors.As, instead of only a
// formatted string.
func TestSoapFaultIsErrorsAsDetectable(t *testing.T) {
	s, ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/soap+xml")
		_, _ = w.Write([]byte(soapEnvelope(
			`<SOAP-ENV:Fault><faultcode>Client</faultcode><faultstring>bad request</faultstring></SOAP-ENV:Fault>`,
		)))
	})
	defer ts.Close()

	crh := ""
	_, err := s.GetStatus(context.Background(), &crh, "")
	var faultErr *SoapFaultError
	if !errors.As(err, &faultErr) {
		t.Fatalf("expected errors.As to find a *SoapFaultError in: %v", err)
	}
	if faultErr.FaultCode != "Client" || faultErr.FaultString != "bad request" {
		t.Fatalf("unexpected fault contents: %+v", faultErr)
	}
}

// TestNilClientRequestHandleReturnsErrorNotPanic pins down that passing a nil
// ClientRequestHandle pointer returns a clear error instead of panicking on the
// dereference.
func TestNilClientRequestHandleReturnsErrorNotPanic(t *testing.T) {
	s, ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be contacted when ClientRequestHandle is nil")
	})
	defer ts.Close()

	if _, err := s.GetStatus(context.Background(), nil, ""); err == nil {
		t.Fatal("expected an error for a nil ClientRequestHandle, got none")
	}
}

func TestSubscriptionCancelReturnsTrueOnSuccess(t *testing.T) {
	s, ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/soap+xml")
		_, _ = w.Write([]byte(soapEnvelope(`<SubscriptionCancelResponse ClientRequestHandle="abc"></SubscriptionCancelResponse>`)))
	})
	defer ts.Close()

	crh := ""
	ok, err := s.SubscriptionCancel(context.Background(), "sub123", "", &crh)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected SubscriptionCancel to report success")
	}
}
