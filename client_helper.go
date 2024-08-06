package gopcxmlda

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// http was preset, because OPC-XML-DA does not support https
func getClientUrl(s *Server) string {
	return fmt.Sprintf("http://%s:%s", s.Addr, s.Port)
}

func buildHeader(namespace string) string {
	var header strings.Builder

	header.WriteString(ENVELOPE_OPEN_1)
	header.WriteString(namespace)
	header.WriteString(ENVELOPE_OPEN_2)
	header.WriteString(ENVELOPE_HEADER)
	header.WriteString(ENVELOPE_BODY_OPEN_NS_1)
	header.WriteString(namespace)
	header.WriteString(ENVELOPE_BODY_OPEN_NS_2)

	return header.String()
}

func buildFooter() string {
	return ENVELOPE_BODY_CLOSE + ENVELOPE_CLOSE
}

// send sends a payload to the server and returns the byte response and an error if any.
func send(s *Server, payload string, timeout time.Duration) ([]byte, error) {
	if timeout == 0 {
		timeout = 10
	}
	url := getClientUrl(s)
	postbody := bytes.NewBuffer([]byte(payload))
	logDebug(fmt.Sprintf("send payload to %s with timeout %d", url, timeout), "send", postbody.String())
	httpClient := http.Client{
		Timeout: timeout * time.Second,
	}
	resp, err := httpClient.Post(url, HEADERS_SOAP["content-type"], postbody)
	if err != nil {
		logError(err.Error(), "send")
		return []byte(""), err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logError(err.Error(), "send")
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		logError(fmt.Sprintf("response status %s", resp.Status), "send")
		return []byte(""), fmt.Errorf("unexpected response status: %s", resp.Status)
	}
	logDebug(fmt.Sprintf("response status %s", resp.Status), "send", resp.Status)
	respbody, err := io.ReadAll(resp.Body)
	if err != nil {
		logError(err.Error(), "send")
		return []byte(""), err
	}
	logDebug("response body", "send", string(respbody))
	return respbody, nil
}

func buildGetStatusPayload(s *Server, namespace string, ClientRequestHandle *string) string {
	var payload strings.Builder
	//header
	payload.WriteString(XML_VERSION)
	payload.WriteString(buildHeader(namespace))
	//body
	payload.WriteString(fmt.Sprintf(
		"<%s:GetStatus LocaleID=\"%s\" ClientRequestHandle=\"%s\"></%s:GetStatus>",
		namespace, s.LocaleID, *ClientRequestHandle, namespace,
	))

	payload.WriteString(buildFooter())

	return payload.String()
}

func buildReadPayload(s *Server, ClientRequestHandle *string, ClientItemHandles *[]string, namespace string,
	items []T_Item, options map[string]interface{}) string {
	var payload strings.Builder
	//header
	payload.WriteString(XML_VERSION)
	payload.WriteString(buildHeader(namespace))
	//body
	payload.WriteString(fmt.Sprintf(
		"<%s:Read LocaleID=\"%s\" ClientRequestHandle=\"%s\">", namespace, s.LocaleID, *ClientRequestHandle,
	))
	payload.WriteString(buildOptionItems(options, namespace))
	payload.WriteString(fmt.Sprintf("<%s:ItemList>", namespace))
	payload.WriteString(buildReadItems(items, ClientItemHandles, namespace))
	payload.WriteString(fmt.Sprintf("</%s:ItemList></%s:Read>", namespace, namespace))
	payload.WriteString(buildFooter())

	return payload.String()
}

func buildOptionItems(options map[string]interface{}, namespace string) string {
	var optionPayload strings.Builder

	if len(options) > 0 {
		optionPayload.WriteString(fmt.Sprintf("<%s:Options", namespace))
		for key, value := range options {
			if reflect.TypeOf(value).Kind() == reflect.Bool {
				optionPayload.WriteString(
					fmt.Sprintf(" %s=\"%v\"", key, strings.ToLower(fmt.Sprintf("%v", value))),
				)
			} else {
				optionPayload.WriteString(fmt.Sprintf(" %s=\"%v\"", key, value))
			}
		}
		optionPayload.WriteString("/>")
	}

	return optionPayload.String()
}

func buildReadItems(items []T_Item, ClientItemHandles *[]string, namespace string) string {
	var readItems strings.Builder

	for i, item := range items {
		readItems.WriteString(fmt.Sprintf("<%s:Items ", namespace))
		if item.ItemName != "" {
			readItems.WriteString(fmt.Sprintf("ItemName=\"%s\" ", item.ItemName))
		}
		if item.ItemPath != "" {
			readItems.WriteString(fmt.Sprintf("ItemPath=\"%s\" ", item.ItemPath))
		}
		readItems.WriteString(fmt.Sprintf("ClientItemHandle=\"%s\"></%s:Items>", (*ClientItemHandles)[i], namespace))
	}

	return readItems.String()
}

func buildBrowsePayload(s *Server, ClientRequestHandle *string,
	itemPath string, namespace string, options T_BrowseOptions) string {
	var payload strings.Builder

	// Header
	payload.WriteString(XML_VERSION)
	payload.WriteString(buildHeader(namespace))

	// Body start
	payload.WriteString(fmt.Sprintf("<%s:Browse LocaleID=\"%s\" ", namespace, s.LocaleID))

	// Adding parameters
	payload.WriteString(fmt.Sprintf("ItemPath=\"%s\" ", itemPath))

	payload.WriteString(fmt.Sprintf("ClientRequestHandle=\"%s\" ", *ClientRequestHandle))
	payload.WriteString(fmt.Sprintf("ItemName=\"%s\" ", options.ItemName))
	payload.WriteString(fmt.Sprintf("ContinuationPoint=\"%s\" ", options.ContinuationPoint))
	payload.WriteString(fmt.Sprintf("MaxElementsReturned=\"%d\" ", options.MaxElementsReturned))
	payload.WriteString(fmt.Sprintf("BrowseFilter=\"%s\" ", options.BrowseFilter))
	payload.WriteString(fmt.Sprintf("ElementNameFilter=\"%s\" ", options.ElementNameFilter))
	payload.WriteString(fmt.Sprintf("VendorFilter=\"%s\" ", options.VendorFilter))
	payload.WriteString(fmt.Sprintf("ReturnAllProperties=\"%s\" ", strconv.FormatBool(options.ReturnAllProperties)))
	payload.WriteString(fmt.Sprintf("ReturnPropertyValues=\"%s\" ", strconv.FormatBool(options.ReturnPropertyValues)))
	payload.WriteString(fmt.Sprintf("ReturnErrorText=\"%s\"", strconv.FormatBool(options.ReturnErrorText)))

	// Body end
	payload.WriteString(fmt.Sprintf("></%s:Browse>", namespace))

	// Footer
	payload.WriteString(buildFooter())

	return payload.String()
}

func buildWritePayload(s *Server, namespace string, items []T_Item, ClientRequestHandle *string, ClientItemHandles *[]string, options map[string]interface{}) string {
	// make sure all items have a (correct) opc-xml-da type
	items = setOpcXmlDaTypes(items)

	var payload strings.Builder
	//header
	payload.WriteString(XML_VERSION)
	payload.WriteString(buildHeader(namespace))
	//body
	payload.WriteString(fmt.Sprintf("<%s:Write ReturnValuesOnReply=\"true\">", namespace))
	options["ClientRequestHandle"] = *ClientRequestHandle
	options["LocaleID"] = s.LocaleID
	payload.WriteString(buildOptionItems(options, namespace))
	payload.WriteString(fmt.Sprintf("<%s:ItemList>", namespace))
	payload.WriteString(buildWriteItems(items, namespace, ClientItemHandles))
	payload.WriteString(fmt.Sprintf("</%s:ItemList></%s:Write>", namespace, namespace))
	payload.WriteString(buildFooter())

	return payload.String()
}

func buildWriteItems(items []T_Item, namespace string, ClientItemHandles *[]string) string {
	var writeItems strings.Builder

	for i, item := range items {
		writeItems.WriteString(fmt.Sprintf("<%s:Items ", namespace))
		if item.ItemName != "" {
			writeItems.WriteString(fmt.Sprintf("ItemName=\"%s\" ", item.ItemName))
		}
		writeItems.WriteString(fmt.Sprintf("ClientItemHandle=\"%s\">", (*ClientItemHandles)[i]))
		writeItems.WriteString(fmt.Sprintf("<%s:Value xsi:type=\"%s:%s\">", namespace, namespace, item.Value.Type))
		writeItems.WriteString(fmt.Sprintf("%s</%s:Value>", buildWriteItemsValue(item.Value, namespace), namespace))
		writeItems.WriteString(fmt.Sprintf("</%s:Items>", namespace))
	}

	return writeItems.String()
}

func buildWriteItemsValue(value T_Value, namespace string) string {
	var writeItemsValue strings.Builder

	if valueIsArrayOrSlice(value.Value) {
		vo := reflect.ValueOf(value.Value)
		for i := 0; i < vo.Len(); i++ {
			v := vo.Index(i).Interface()
			valueType, err := getOpcXmlDaType(v)
			if err != nil {
				logError(err.Error(), "buildWriteItems_Value")
			}
			writeItemsValue.WriteString(
				fmt.Sprintf("<%s:%s>%v</%s:%s>", namespace, valueType, v, namespace, valueType),
			)
		}
	} else {
		writeItemsValue.WriteString(fmt.Sprintf("%v", value.Value))
	}

	return writeItemsValue.String()
}

func buildSubscribePayload(namespace string, items []T_Item, ClientRequestHandle *string, ClientItemHandles *[]string,
	returnValuesOnReply bool, subscriptionPingRate uint, enableBuffering bool, options map[string]interface{}) string {
	var payload strings.Builder
	//header
	payload.WriteString(XML_VERSION)
	payload.WriteString(buildHeader(namespace))
	//body

	payload.WriteString(fmt.Sprintf("<%s:Subscribe ReturnValuesOnReply=\"%s\" SubscriptionPingRate=\"%v\" ClientRequestHandle=\"%s\">",
		namespace, strings.ToLower(fmt.Sprintf("%v", returnValuesOnReply)), subscriptionPingRate, *ClientRequestHandle))
	payload.WriteString(buildOptionItems(options, namespace))
	payload.WriteString(fmt.Sprintf("<%s:ItemList xsi:type=\"SubscribeRequestItemList\" ItemPath=\"\" ", namespace))
	payload.WriteString("Deadband=\"0.0\" RequestedSamplingRate=\"0\" ")
	payload.WriteString(fmt.Sprintf("EnableBuffering=\"%s\">", strings.ToLower(fmt.Sprintf("%v", enableBuffering))))
	payload.WriteString(buildSubscribeItems(items, ClientItemHandles, namespace, enableBuffering))
	payload.WriteString(fmt.Sprintf("</%s:ItemList></%s:Subscribe>", namespace, namespace))
	payload.WriteString(buildFooter())

	return payload.String()
}

func buildSubscribeItems(items []T_Item, ClientItemHandles *[]string, namespace string, enableBuffering bool) string {
	var subscribeItems strings.Builder

	for i, item := range items {
		subscribeItems.WriteString(fmt.Sprintf("<%s:Items xsi:type=\"%s:SubscribeRequestItem\" Deadband=\"0.0\" ", namespace, namespace))
		subscribeItems.WriteString(fmt.Sprintf("RequestedSamplingRate=\"0\" EnableBuffering=\"%s\"", strings.ToLower(fmt.Sprintf("%v", enableBuffering))))
		if item.ItemName != "" {
			subscribeItems.WriteString(fmt.Sprintf(" ItemName=\"%s\"", item.ItemName))
		}
		if item.ItemPath != "" {
			subscribeItems.WriteString(fmt.Sprintf(" ItemPath=\"%s\"", item.ItemPath))
		}
		subscribeItems.WriteString(fmt.Sprintf(" ClientItemHandle=\"%s\"></%s:Items>", (*ClientItemHandles)[i], namespace))
	}

	return subscribeItems.String()
}

func buildSubscriptionCancelPayload(serverSubHandle string, namespace string, ClientRequestHandle *string) string {
	var payload strings.Builder
	//header
	payload.WriteString(XML_VERSION)
	payload.WriteString(buildHeader(namespace))
	//body
	payload.WriteString(fmt.Sprintf("<%s:SubscriptionCancel ServerSubHandle=\"%s\" ClientRequestHandle=\"%s\"></%s:SubscriptionCancel>",
		namespace, serverSubHandle, *ClientRequestHandle, namespace))
	payload.WriteString(buildFooter())

	return payload.String()
}

func buildSubscriptionPolledRefreshPayload(serverSubHandle string, namespace string, ClientRequestHandle *string,
	SubscriptionPingRate uint, options map[string]interface{}, ServerTime T_ServerTime) (string, error) {
	var payload strings.Builder
	//header
	payload.WriteString(XML_VERSION)
	payload.WriteString(buildHeader(namespace))
	//body
	holdTime, err := calcHoldTime(SubscriptionPingRate, ServerTime)
	if err != nil {
		return "", err
	}
	payload.WriteString(fmt.Sprintf("<%s:SubscriptionPolledRefresh HoldTime=\"%s\" ReturnAllItems=\"false\" WaitTime=\"500\">", namespace, holdTime))
	options["ClientRequestHandle"] = *ClientRequestHandle
	payload.WriteString(buildOptionItems(options, namespace))
	payload.WriteString(fmt.Sprintf("<%s:ServerSubHandles>%s</%s:ServerSubHandles>", namespace, serverSubHandle, namespace))
	payload.WriteString(fmt.Sprintf("</%s:SubscriptionPolledRefresh>", namespace))
	payload.WriteString(buildFooter())

	return payload.String(), nil
}

func calcHoldTime(subscriptionPingRate uint, ServerTime T_ServerTime) (string, error) {
	if ServerTime.UseClientTime {
		now := time.Now()
		next := now.Add(time.Duration(subscriptionPingRate) * time.Millisecond)
		return next.Format(time.RFC3339), nil
	} else {
		next := ServerTime.ServerTime.Add(time.Duration(subscriptionPingRate) * time.Millisecond)
		return next.Format(time.RFC3339), nil
	}

}
