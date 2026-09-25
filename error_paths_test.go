package gopcxmlda

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestUnsupportedMediaTypeIsReported covers what issue #4 looked like from the
// caller's side: a strict server rejecting the request with 415 and an HTML body. The
// status must be visible in the error, and no half-parsed result may be returned.
func TestUnsupportedMediaTypeIsReported(t *testing.T) {
	s, ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusUnsupportedMediaType)
		_, _ = w.Write([]byte("<html><body><h1>415 Unsupported Media Type</h1></body></html>"))
	})
	defer ts.Close()

	crh := ""
	status, err := s.GetStatus(context.Background(), &crh, "")
	if err == nil || !strings.Contains(err.Error(), "415") {
		t.Fatalf("expected an error mentioning status 415, got: %v", err)
	}
	if status.Response.Result.ServerState != "" {
		t.Fatalf("expected a zero result, got %+v", status)
	}
}

func TestMalformedResponseBodyIsAnError(t *testing.T) {
	s, ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<SOAP-ENV:Envelope><SOAP-ENV:Body><GetStatusResponse>`))
	})
	defer ts.Close()

	crh := ""
	if _, err := s.GetStatus(context.Background(), &crh, ""); err == nil {
		t.Fatal("expected an error for a truncated response body")
	}
}

// TestUnreachableServerReturnsTransportErrorOnly pins down that a transport failure is
// reported as-is, without a spurious XML "EOF" error from decoding an empty body.
func TestUnreachableServerReturnsTransportErrorOnly(t *testing.T) {
	s, ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {})
	ts.Close() // nothing listens on the URL any more

	crh := ""
	_, err := s.GetStatus(context.Background(), &crh, "")
	if err == nil {
		t.Fatal("expected an error for an unreachable server")
	}
	if strings.Contains(err.Error(), "EOF") {
		t.Fatalf("expected only the transport error, got: %v", err)
	}
}

func TestCanceledContextAbortsRequest(t *testing.T) {
	s, ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be reached with an already canceled context")
	})
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	crh := ""
	if _, err := s.GetStatus(ctx, &crh, ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func TestSlowServerHitsTimeout(t *testing.T) {
	release := make(chan struct{})
	s, ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) { <-release })
	defer ts.Close()
	defer close(release)
	s.Timeout = 50 * time.Millisecond

	crh := ""
	start := time.Now()
	if _, err := s.GetStatus(context.Background(), &crh, ""); err == nil {
		t.Fatal("expected a timeout error")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("expected the configured 50ms timeout to apply, took %v", elapsed)
	}
}

// TestInvalidArgumentsNeverReachTheServer checks the argument validation of every
// public method: each case must fail with an error before any request is sent.
func TestInvalidArgumentsNeverReachTheServer(t *testing.T) {
	s, ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server contacted for %s despite invalid arguments", r.Header.Get("SOAPAction"))
	})
	defer ts.Close()
	ctx := context.Background()
	items := []TItem{{ItemName: "A", Value: TValue{Value: 1}}, {ItemName: "B", Value: TValue{Value: 2}}}
	crh := func() *string { s := ""; return &s }
	oneHandle := func() *[]string { h := []string{"only-one"}; return &h }
	noHandles := func() *[]string { var h []string; return &h }

	cases := map[string]func() error{
		"GetStatus nil request handle": func() error { _, err := s.GetStatus(ctx, nil, ""); return err },
		"Read nil request handle":      func() error { _, err := s.Read(ctx, items, nil, noHandles(), "", nil); return err },
		"Read nil item handles":        func() error { _, err := s.Read(ctx, items, crh(), nil, "", nil); return err },
		"Read mismatched item handles": func() error { _, err := s.Read(ctx, items, crh(), oneHandle(), "", nil); return err },
		"Read invalid MaxAge": func() error {
			_, err := s.Read(ctx, items, crh(), noHandles(), "", map[string]interface{}{MaxAgeOption: -1})
			return err
		},
		"Browse nil request handle":     func() error { _, err := s.Browse(ctx, "", nil, "", TBrowseOptions{}); return err },
		"Write nil request handle":      func() error { _, err := s.Write(ctx, items, nil, noHandles(), "", nil); return err },
		"Write mismatched item handles": func() error { _, err := s.Write(ctx, items, crh(), oneHandle(), "", nil); return err },
		"Write nil value": func() error {
			_, err := s.Write(ctx, []TItem{{ItemName: "A"}}, crh(), noHandles(), "", nil)
			return err
		},
		"Write unsupported value type": func() error {
			_, err := s.Write(ctx, []TItem{{ItemName: "A", Value: TValue{Value: struct{}{}}}}, crh(), noHandles(), "", nil)
			return err
		},
		"Subscribe nil request handle": func() error {
			_, err := s.Subscribe(ctx, items, nil, noHandles(), "", false, 0, nil)
			return err
		},
		"Subscribe mismatched item handles": func() error {
			_, err := s.Subscribe(ctx, items, crh(), oneHandle(), "", false, 0, nil)
			return err
		},
		"SubscriptionPolledRefresh nil request handle": func() error {
			_, err := s.SubscriptionPolledRefresh(ctx, "h", 0, "", nil, nil, TServerTime{UseClientTime: true})
			return err
		},
		"SubscriptionCancel nil request handle": func() error { _, err := s.SubscriptionCancel(ctx, "h", "", nil); return err },
		"GetProperties nil request handle": func() error {
			_, err := s.GetProperties(ctx, items, TPropertyOptions{}, nil, "")
			return err
		},
	}
	for name, call := range cases {
		t.Run(name, func(t *testing.T) {
			if err := call(); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestNilServerAndUnknownActionAreRejected(t *testing.T) {
	var nilServer *Server
	if _, err := send(context.Background(), nilServer, "", "GetStatus"); err == nil {
		t.Error("expected an error for a nil *Server")
	}
	u, _ := url.Parse("http://example.invalid")
	if _, err := send(context.Background(), &Server{Url: u}, "", "NoSuchOperation"); err == nil {
		t.Error("expected an error for an unknown SOAPAction")
	}
}

// TestGeneratedHandlesFillOnlyWhatIsMissing checks that request and item handles are
// generated when empty, left alone when supplied, and written back to the caller.
func TestGeneratedHandlesFillOnlyWhatIsMissing(t *testing.T) {
	rec := &headerRecorder{}
	s, ts := newTestServer(t, rec.handle)
	defer ts.Close()
	items := []TItem{{ItemName: "A"}, {ItemName: "B"}}

	crh, cih := "", []string(nil)
	_, _ = s.Read(context.Background(), items, &crh, &cih, "", nil)
	if len(crh) != 16 || len(cih) != 2 || cih[0] == cih[1] {
		t.Fatalf("expected a generated 16-char request handle and 2 distinct item handles, got %q %q", crh, cih)
	}

	crh, cih = "mine", []string{"a", "b"}
	_, _ = s.Read(context.Background(), items, &crh, &cih, "", nil)
	if crh != "mine" || cih[0] != "a" || cih[1] != "b" {
		t.Fatalf("expected supplied handles to be kept, got %q %q", crh, cih)
	}
}

func TestGenerateClientHandles(t *testing.T) {
	crh, handles, err := GenerateClientHandles(3)
	if err != nil {
		t.Fatal(err)
	}
	if len(crh) != 16 || len(handles) != 3 {
		t.Fatalf("expected a 16-char handle and 3 item handles, got %q %q", crh, handles)
	}
	seen := map[string]bool{}
	for _, h := range handles {
		if seen[h] {
			t.Fatalf("duplicate item handle %q in %q", h, handles)
		}
		seen[h] = true
	}
	other, _, _ := GenerateClientHandles(0)
	if other == crh {
		t.Fatal("expected two calls to produce different request handles")
	}
}

func TestCalcHoldTime(t *testing.T) {
	serverTime := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	got, err := calcHoldTime(1500, TServerTime{ServerTime: serverTime})
	if err != nil {
		t.Fatal(err)
	}
	if want := serverTime.Add(1500 * time.Millisecond).Format(time.RFC3339); got != want {
		t.Fatalf("expected HoldTime %s based on the server time, got %s", want, got)
	}

	before := time.Now().Add(-time.Second)
	got, err = calcHoldTime(60000, TServerTime{UseClientTime: true})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := time.Parse(time.RFC3339, got)
	if err != nil {
		t.Fatalf("HoldTime %q is not RFC3339: %v", got, err)
	}
	if parsed.Before(before.Add(60*time.Second)) || parsed.After(time.Now().Add(61*time.Second)) {
		t.Fatalf("expected HoldTime about 60s after the client time, got %s", got)
	}
}

func TestErrorMessages(t *testing.T) {
	opcErr := (&OpcResponseError{Id: "E_FAIL", Text: []string{"boom"}, Type: "t"}).Error()
	if !strings.Contains(opcErr, "E_FAIL") || !strings.Contains(opcErr, "boom") {
		t.Errorf("unexpected OpcResponseError message: %s", opcErr)
	}
	invalid := (&InvalidServerSubHandlesError{Handles: []string{"h1", "h2"}}).Error()
	if !strings.Contains(invalid, "h1") || !strings.Contains(invalid, "h2") {
		t.Errorf("unexpected InvalidServerSubHandlesError message: %s", invalid)
	}
}

// TestEmptyNamespaceDefaultsToNs0 pins down the documented default namespace prefix.
func TestEmptyNamespaceDefaultsToNs0(t *testing.T) {
	var body string
	s, ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		_, _ = w.Write([]byte(soapEnvelope("")))
	})
	defer ts.Close()

	crh := ""
	_, _ = s.GetStatus(context.Background(), &crh, "")
	if !strings.Contains(body, "<ns0:GetStatus ") {
		t.Fatalf("expected the ns0 prefix by default, got: %s", body)
	}
}
