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

	_items := []string{"Loc/Wec/Plant3/Log/T82a1/Raw/Val-1"}
	items := make([]TItem, len(_items))
	for i, item := range _items {
		items[i] = TItem{
			ItemName: item,
		}
	}

	options := map[string]interface{}{
		"ReturnItemTime": true,
		"ReturnItemPath": true,
		"ReturnItemName": true,
	}
	var ClientRequestHandle string
	var ClientItemHandles []string
	for _, item := range _items {
		ClientItemHandles = append(ClientItemHandles, item)
	}
	SubscriptionPingRate := uint(20000)
	response, err := s.Subscribe(items, &ClientRequestHandle, &ClientItemHandles, "", true, SubscriptionPingRate, false, 20000, options)
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
