package gopcxmlda

import (
	"fmt"
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
	OPCIP := os.Getenv("IP")
	OPCPort := os.Getenv("PORT")
	s := Server{OPCIP, OPCPort, "en-US", 10}
	var ClientRequestHandle string
	_, err = s.GetStatus(&ClientRequestHandle, "")
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log("GetStatus successful")
	}
}

func TestRead(t *testing.T) {
	err := godotenv.Load()
	if err != nil {
		t.Fatal("Error loading .env file")
	}
	OPCIP := os.Getenv("IP")
	OPCPort := os.Getenv("PORT")
	s := Server{OPCIP, OPCPort, "en-US", 10}
	items := []T_Item{
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
	_, err = s.Read(items, &ClientRequestHandle, &ClientItemHandles, "", options)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log("Read successful")
	}
}

func TestBrowse(t *testing.T) {
	err := godotenv.Load()
	if err != nil {
		t.Fatal("Error loading .env file")
	}
	OPCIP := os.Getenv("IP")
	OPCPort := os.Getenv("PORT")
	s := Server{OPCIP, OPCPort, "en-US", 10}
	var ClientRequestHandle string
	r, err := s.Browse("Loc/Wec/Plant1", &ClientRequestHandle, "", T_BrowseOptions{
		ReturnAllProperties:  true,
		ReturnPropertyValues: true,
	})
	if err != nil {
		t.Fatal(err)
	} else {
		fmt.Println(r)
		t.Log("Browse successful")
	}
}

func TestWrite(t *testing.T) {
	err := godotenv.Load()
	if err != nil {
		t.Fatal("Error loading .env file")
	}
	OPCIP := os.Getenv("IP")
	OPCPort := os.Getenv("PORT")
	s := Server{OPCIP, OPCPort, "en-US", 10}
	items := []T_Item{
		{
			ItemName: "Loc/Wec/Plant1/Ctrl/SessionRequest",
			Value: T_Value{
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
	_, err = s.Write(items, &ClientRequestHandle, &ClientItemHandles, "", options)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log("Write successful")
	}
}

func TestSubscribe(t *testing.T) {
	err := godotenv.Load()
	if err != nil {
		t.Fatal("Error loading .env file")
	}
	OPCIP := os.Getenv("IP")
	OPCPort := os.Getenv("PORT")
	s := Server{OPCIP, OPCPort, "en-US", 10}
	items := []T_Item{
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
		"ReturnItemPath": true,
		"ReturnItemName": true,
	}
	var ClientRequestHandle string
	var ClientItemHandles []string
	SubscriptionPingRate := uint(3000)
	response, err := s.Subscribe(items, &ClientRequestHandle, &ClientItemHandles, "", true, SubscriptionPingRate, false, options)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log("Subscribe successful")
	}

	// Subscription Polled Refresh
	var ClientRequestHandle1 string
	optionsPolledRefresh := map[string]interface{}{
		"ReturnErrorText": true,
		"ReturnItemPath":  true,
		"ReturnItemTime":  true,
		"ReturnItemName":  true,
	}
	ServerTime := T_ServerTime{response.Body.Response.Result.ReplyTime, false}
	refreshResponse, err := s.SubscriptionPolledRefresh(
		response.Body.Response.ServerSubHandle, SubscriptionPingRate,
		"", &ClientRequestHandle1, optionsPolledRefresh, ServerTime,
	)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log("Polled Refresh successful")
	}
	ServerTime = T_ServerTime{
		refreshResponse.Body.SubscriptionPolledRefreshResponse.SubscriptionPolledRefreshResult.ReplyTime,
		false,
	}
	refreshResponse, err = s.SubscriptionPolledRefresh(response.Body.Response.ServerSubHandle, SubscriptionPingRate, "", &ClientRequestHandle1, optionsPolledRefresh, ServerTime)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log("Polled Refresh successful")
	}
	ServerTime = T_ServerTime{
		refreshResponse.Body.SubscriptionPolledRefreshResponse.SubscriptionPolledRefreshResult.ReplyTime,
		false,
	}
	refreshResponse, err = s.SubscriptionPolledRefresh(response.Body.Response.ServerSubHandle, SubscriptionPingRate, "", &ClientRequestHandle1, optionsPolledRefresh, ServerTime)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log("Polled Refresh successful")
	}
	ServerTime = T_ServerTime{
		refreshResponse.Body.SubscriptionPolledRefreshResponse.SubscriptionPolledRefreshResult.ReplyTime,
		false,
	}
	refreshResponse, err = s.SubscriptionPolledRefresh(response.Body.Response.ServerSubHandle, SubscriptionPingRate, "", &ClientRequestHandle1, optionsPolledRefresh, ServerTime)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log("Polled Refresh successful")
	}
	ServerTime = T_ServerTime{
		refreshResponse.Body.SubscriptionPolledRefreshResponse.SubscriptionPolledRefreshResult.ReplyTime,
		false,
	}
	time.Sleep(time.Duration(SubscriptionPingRate) * time.Millisecond)

	// Unsubscribe
	var ClientRequestHandle2 string
	canceled, err := s.SubscriptionCancel(response.Body.Response.ServerSubHandle, "", &ClientRequestHandle2)
	if err != nil {
		t.Fatal(err)
	} else if !canceled {
		t.Fatal("Subscription not canceled")
	} else {
		t.Log("Subscription started, refreshed and canceled successfully")
	}
}
