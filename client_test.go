package gopcxmlda

import (
	"net/url"
	"os"
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
	Status, err := s.GetStatus(&ClientRequestHandle, "")
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
	R, err := s.Read(items, &ClientRequestHandle, &ClientItemHandles, "", options)
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
	r, err := s.Browse("Loc/Wec/Plant1", &ClientRequestHandle, "", TBrowseOptions{
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
	w, err := s.Write(items, &ClientRequestHandle, &ClientItemHandles, "", options)
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
	response, err := s.Subscribe(items, &ClientRequestHandle, &ClientItemHandles, "", true, SubscriptionPingRate, options)
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
		response.Response.ServerSubHandle, SubscriptionPingRate,
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
		response.Response.ServerSubHandle, SubscriptionPingRate,
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
		response.Response.ServerSubHandle, SubscriptionPingRate,
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
		response.Response.ServerSubHandle, SubscriptionPingRate,
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
		response.Response.ServerSubHandle, SubscriptionPingRate,
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
		response.Response.ServerSubHandle, SubscriptionPingRate,
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
	canceled, err := s.SubscriptionCancel(response.Response.ServerSubHandle, "", &ClientRequestHandle2)
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
	p, err := s.GetProperties(items, propertyOptions, &ClientRequestHandle, "")
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("GetProperties: %+v", p)
	}
}
