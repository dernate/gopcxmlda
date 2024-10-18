package gopcxmlda

import (
	"context"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func TestGetStatus(t *testing.T) {
	err := godotenv.Load()
	if err != nil {
		t.Fatal("Error loading .env file")
	}
	OpcUrl := os.Getenv("OPC_URL")
	_url, err := url.Parse(OpcUrl)
	if err != nil {
		t.Fatal(err)
	}
	s := Server{_url, "en-US", 10}
	var ClientRequestHandle string
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
	}()
	Status, err := s.GetStatus(ctx, &ClientRequestHandle, "")
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("Status: %+v", Status)
	}
}

func TestRead(t *testing.T) {
	err := godotenv.Load()
	if err != nil {
		t.Fatal("Error loading .env file")
	}
	OpcUrl := os.Getenv("OPC_URL")
	_url, err := url.Parse(OpcUrl)
	s := Server{_url, "en-US", 10}
	items := []TItem{
		{
			ItemName: "Loc/Wec/Plant1/P",
		},
		{
			ItemName: "Loc/Wec/Plant1/Log/Wecstd/Rep/Val-1",
		},
		{
			ItemName: "Loc/Wec/Plant1/Status/St",
		},
	}
	options := map[string]interface{}{
		"ReturnItemTime": true,
		"returnItemPath": true,
		"returnItemName": true,
	}
	var ClientRequestHandle string
	var ClientItemHandles []string
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
	}()
	R, err := s.Read(ctx, items, &ClientRequestHandle, &ClientItemHandles, "", options)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("Read: %+v", R)
	}
}

func TestBrowse(t *testing.T) {
	err := godotenv.Load()
	if err != nil {
		t.Fatal("Error loading .env file")
	}
	OpcUrl := os.Getenv("OPC_URL")
	_url, err := url.Parse(OpcUrl)
	s := Server{_url, "en-US", 10}
	var ClientRequestHandle string
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
	}()
	r, err := s.Browse(ctx, "Loc/Wec/Plant1", &ClientRequestHandle, "", TBrowseOptions{
		ReturnAllProperties:  true,
		ReturnPropertyValues: true,
	})
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("Browse: %+v", r)
	}
}

func TestWrite(t *testing.T) {
	err := godotenv.Load()
	if err != nil {
		t.Fatal("Error loading .env file")
	}
	OpcUrl := os.Getenv("OPC_URL")
	_url, err := url.Parse(OpcUrl)
	s := Server{_url, "en-US", 10}
	items := []TItem{
		{
			ItemName: "Loc/Wec/Plant1/Ctrl/SessionRequest",
			Value: TValue{
				Value: []int{0, 0, 0},
			},
		},
	}
	var ClientRequestHandle string
	var ClientItemHandles []string
	options := map[string]interface{}{
		"ReturnErrorText": true,
		"ReturnItemName":  true,
		"ReturnItemPath":  true,
	}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
	}()
	w, err := s.Write(ctx, items, &ClientRequestHandle, &ClientItemHandles, "", options)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("Write: %+v", w)
	}
}

func TestSubscribe(t *testing.T) {
	err := godotenv.Load()
	if err != nil {
		t.Fatal("Error loading .env file")
	}
	OpcUrl := os.Getenv("OPC_URL")
	_url, err := url.Parse(OpcUrl)
	s := Server{_url, "en-US", 30}

	items := []TItem{
		{
			ItemName:              "Loc/Wec/Plant1/Vane",
			EnableBuffering:       true,
			RequestedSamplingRate: 3000,
		},
		{
			ItemName:              "Loc/Wec/Plant1/P",
			EnableBuffering:       true,
			RequestedSamplingRate: 1000,
		},
		{
			ItemName:              "Loc/Wec/Plant1/Vwind",
			EnableBuffering:       false,
			RequestedSamplingRate: 5000,
		},
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
	SubscriptionPingRate := uint(2000)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
	}()
	response, err := s.Subscribe(ctx, items, &ClientRequestHandle, &ClientItemHandles, "", true, SubscriptionPingRate, options)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("Subscription started. SubscriptionResponse: %+v", response)
	}

	// Subscription Polled Refresh
	var ClientRequestHandle1 string
	optionsPolledRefresh := map[string]interface{}{
		"ReturnErrorText": true,
		"ReturnItemTime":  true,
	}
	ServerTime := TServerTime{response.Response.Result.ReplyTime, false}
	refreshResponse, err := s.SubscriptionPolledRefresh(
		ctx, response.Response.ServerSubHandle, SubscriptionPingRate,
		"", &ClientRequestHandle1, optionsPolledRefresh, ServerTime,
	)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("Polled Refresh successful. RefreshResponse: %+v", refreshResponse)
	}
	ServerTime = TServerTime{
		refreshResponse.Response.Result.ReplyTime,
		false,
	}
	refreshResponse, err = s.SubscriptionPolledRefresh(
		ctx, response.Response.ServerSubHandle, SubscriptionPingRate,
		"", &ClientRequestHandle1, optionsPolledRefresh, ServerTime,
	)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("Polled Refresh successful. RefreshResponse: %+v", refreshResponse)
	}
	ServerTime = TServerTime{
		refreshResponse.Response.Result.ReplyTime,
		false,
	}
	refreshResponse, err = s.SubscriptionPolledRefresh(
		ctx, response.Response.ServerSubHandle, SubscriptionPingRate,
		"", &ClientRequestHandle1, optionsPolledRefresh, ServerTime,
	)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("Polled Refresh successful. RefreshResponse: %+v", refreshResponse)
	}
	ServerTime = TServerTime{
		refreshResponse.Response.Result.ReplyTime,
		false,
	}
	refreshResponse, err = s.SubscriptionPolledRefresh(
		ctx, response.Response.ServerSubHandle, SubscriptionPingRate,
		"", &ClientRequestHandle1, optionsPolledRefresh, ServerTime,
	)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("Polled Refresh successful. RefreshResponse: %+v", refreshResponse)
	}
	ServerTime = TServerTime{
		refreshResponse.Response.Result.ReplyTime,
		false,
	}
	refreshResponse, err = s.SubscriptionPolledRefresh(
		ctx, response.Response.ServerSubHandle, SubscriptionPingRate,
		"", &ClientRequestHandle1, optionsPolledRefresh, ServerTime,
	)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("Polled Refresh successful. RefreshResponse: %+v", refreshResponse)
	}
	ServerTime = TServerTime{
		refreshResponse.Response.Result.ReplyTime,
		false,
	}
	refreshResponse, err = s.SubscriptionPolledRefresh(
		ctx, response.Response.ServerSubHandle, SubscriptionPingRate,
		"", &ClientRequestHandle1, optionsPolledRefresh, ServerTime,
	)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("Polled Refresh successful. RefreshResponse: %+v", refreshResponse)
	}
	ServerTime = TServerTime{
		refreshResponse.Response.Result.ReplyTime,
		false,
	}
	time.Sleep(time.Duration(SubscriptionPingRate) * time.Millisecond)

	// Unsubscribe
	var ClientRequestHandle2 string
	canceled, err := s.SubscriptionCancel(ctx, response.Response.ServerSubHandle, "", &ClientRequestHandle2)
	if err != nil {
		t.Fatal(err)
	} else if !canceled {
		t.Fatal("Subscription not canceled")
	} else {
		t.Log("Subscription started, refreshed and canceled successfully")
	}
}

func TestGetProperties(t *testing.T) {
	err := godotenv.Load()
	if err != nil {
		t.Fatal("Error loading .env file")
	}
	OpcUrl := os.Getenv("OPC_URL")
	_url, err := url.Parse(OpcUrl)
	s := Server{_url, "en-US", 10}
	var ClientRequestHandle string
	items := []TItem{
		{
			ItemName: "Loc/Wec/Plant1/Log/Wecstd/Rep/Val-1",
		},
		{
			ItemName: "Loc/LocNo",
		},
	}
	propertyOptions := TPropertyOptions{
		ReturnAllProperties:  true,
		ReturnPropertyValues: true,
		ReturnErrorText:      true,
	}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
	}()
	p, err := s.GetProperties(ctx, items, propertyOptions, &ClientRequestHandle, "")
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("GetProperties: %+v", p)
	}
}
