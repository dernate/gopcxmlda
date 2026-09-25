//go:build live

// Live integration tests against a real OPC XML-DA server. They only compile with the
// "live" build tag, so a plain `go test ./...` never touches the network and runs the
// complete offline suite instead. Run the offline suite first and the live tests only
// once it passes:
//
//	go test ./... && go test -tags live -run '^TestLive' -v .
//
// The server URL comes from OPC_URL (process environment or .env). Every test except
// TestLiveWrite is read-only; TestLiveWrite additionally requires GOPCXMLDA_LIVE_WRITE=1.
package gopcxmlda

import (
	"context"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

// liveWriteOptIn is the environment variable that must be set to "1" for TestLiveWrite
// to run. TestLiveWrite writes to a real control item on the live server, so it must
// never run as a side effect of running the live suite.
const liveWriteOptIn = "GOPCXMLDA_LIVE_WRITE"

// liveServer returns a Server for OPC_URL, loading .env if present. It skips the test
// (instead of failing with an opaque "unsupported protocol scheme" error) when no URL
// is configured.
func liveServer(t *testing.T, timeout time.Duration) *Server {
	t.Helper()
	// A missing .env is fine as long as OPC_URL is set in the environment; Load never
	// overrides variables that are already set.
	_ = godotenv.Load()
	raw := os.Getenv("OPC_URL")
	if raw == "" {
		t.Skip("OPC_URL is not set (neither in the environment nor in .env)")
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("OPC_URL is not a valid URL: %v", err)
	}
	return &Server{Url: u, LocaleID: "en-US", Timeout: timeout}
}

func TestLiveGetStatus(t *testing.T) {
	s := liveServer(t, 10*time.Second)
	var ClientRequestHandle string
	status, err := s.GetStatus(context.Background(), &ClientRequestHandle, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Status: %+v", status)
}

func TestLiveRead(t *testing.T) {
	s := liveServer(t, 10*time.Second)
	items := []TItem{
		{ItemName: "Loc/Wec/Plant1/P"},
		{ItemName: "Loc/Wec/Plant1/Log/Wecstd/Rep/Val-1"},
		{ItemName: "Loc/Wec/Plant1/Status/St"},
	}
	options := map[string]interface{}{
		"ReturnItemTime": true,
		"ReturnItemPath": true,
		"ReturnItemName": true,
	}
	var ClientRequestHandle string
	var ClientItemHandles []string
	r, err := s.Read(context.Background(), items, &ClientRequestHandle, &ClientItemHandles, "", options)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Read: %+v", r)
}

func TestLiveBrowse(t *testing.T) {
	s := liveServer(t, 10*time.Second)
	var ClientRequestHandle string
	r, err := s.Browse(context.Background(), "Loc/Wec/Plant1", &ClientRequestHandle, "", TBrowseOptions{
		ReturnAllProperties:  true,
		ReturnPropertyValues: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Browse: %+v", r)
}

func TestLiveGetProperties(t *testing.T) {
	s := liveServer(t, 10*time.Second)
	var ClientRequestHandle string
	items := []TItem{
		{ItemName: "Loc/Wec/Plant1/Log/Wecstd/Rep/Val-1"},
		{ItemName: "Loc/LocNo"},
	}
	p, err := s.GetProperties(context.Background(), items, TPropertyOptions{
		ReturnAllProperties:  true,
		ReturnPropertyValues: true,
		ReturnErrorText:      true,
	}, &ClientRequestHandle, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("GetProperties: %+v", p)
}

// TestLiveSubscribe subscribes, polls a few refreshes and cancels. Subscribing only
// creates server-side subscription state; no process value is written.
func TestLiveSubscribe(t *testing.T) {
	s := liveServer(t, 30*time.Second)
	ctx := context.Background()
	items := []TItem{
		{ItemName: "Loc/Wec/Plant1/Vane", EnableBuffering: true, RequestedSamplingRate: 3000},
		{ItemName: "Loc/Wec/Plant1/P", EnableBuffering: true, RequestedSamplingRate: 1000},
		{ItemName: "Loc/Wec/Plant1/Vwind", EnableBuffering: false, RequestedSamplingRate: 5000},
	}
	options := map[string]interface{}{
		"ReturnItemTime": true,
		"ReturnItemPath": true,
		"ReturnItemName": true,
	}
	var ClientRequestHandle string
	var ClientItemHandles []string
	for _, item := range items {
		ClientItemHandles = append(ClientItemHandles, item.ItemName)
	}
	const SubscriptionPingRate = uint(2000)

	response, err := s.Subscribe(ctx, items, &ClientRequestHandle, &ClientItemHandles, "", true, SubscriptionPingRate, options)
	handle := response.Response.ServerSubHandle
	// Cancel the subscription whenever the server created one - also when a later step
	// fails - so a failed run doesn't leave a dangling subscription on the server.
	canceled := false
	if handle != "" {
		t.Cleanup(func() {
			if canceled {
				return
			}
			var crh string
			if _, err := s.SubscriptionCancel(ctx, handle, "", &crh); err != nil {
				t.Logf("cleanup: canceling subscription %s failed: %v", handle, err)
			}
		})
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Subscription started. SubscriptionResponse: %+v", response)

	optionsPolledRefresh := map[string]interface{}{
		"ReturnErrorText": true,
		"ReturnItemTime":  true,
	}
	serverTime := TServerTime{ServerTime: response.Response.Result.ReplyTime}
	for i := 0; i < 6; i++ {
		var crh string
		refresh, err := s.SubscriptionPolledRefresh(ctx, handle, SubscriptionPingRate, "", &crh,
			optionsPolledRefresh, serverTime)
		if err != nil {
			t.Fatalf("polled refresh %d: %v", i+1, err)
		}
		t.Logf("Polled refresh %d successful. RefreshResponse: %+v", i+1, refresh)
		serverTime = TServerTime{ServerTime: refresh.Response.Result.ReplyTime}
	}
	time.Sleep(time.Duration(SubscriptionPingRate) * time.Millisecond)

	var ClientRequestHandle2 string
	ok, err := s.SubscriptionCancel(ctx, handle, "", &ClientRequestHandle2)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("Subscription not canceled")
	}
	canceled = true
	t.Log("Subscription started, refreshed and canceled successfully")
}

// TestLiveWrite writes [0,0,0] to a real control item. It is skipped unless
// GOPCXMLDA_LIVE_WRITE=1 is set in the process environment.
func TestLiveWrite(t *testing.T) {
	// Checked before liveServer() loads .env on purpose: only the real process
	// environment can enable the write, so it can't be switched on permanently via .env.
	if os.Getenv(liveWriteOptIn) != "1" {
		t.Skipf("writes to a live control item; set %s=1 to run it", liveWriteOptIn)
	}
	s := liveServer(t, 10*time.Second)
	items := []TItem{
		{ItemName: "Loc/Wec/Plant1/Ctrl/SessionRequest", Value: TValue{Value: []int{0, 0, 0}}},
	}
	options := map[string]interface{}{
		"ReturnErrorText": true,
		"ReturnItemName":  true,
		"ReturnItemPath":  true,
	}
	var ClientRequestHandle string
	var ClientItemHandles []string
	w, err := s.Write(context.Background(), items, &ClientRequestHandle, &ClientItemHandles, "", options)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Write: %+v", w)
}
