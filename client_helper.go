package gopcxmlda

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func buildHeader(builder *strings.Builder, namespace string) string {
	builder.WriteString(EnvelopeOpen1)
	builder.WriteString(namespace)
	builder.WriteString(EnvelopeHeaderToBody)
	builder.WriteString(namespace)
	builder.WriteString(EnvelopeBodyOpenNs2)

	return builder.String()
}

// send sends a payload to the server and returns the byte response and an error if any.
func send(ctx context.Context, s *Server, payload string) ([]byte, error) {
	if s.Timeout == 0 {
		s.Timeout = 10
	}
	postbody := bytes.NewBuffer([]byte(payload))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.Url.String(), postbody)
	if err != nil {
		return []byte(""), err
	}
	req.Header.Set("Content-Type", HeadersSoap["content-type"])
	httpClient := &http.Client{
		Timeout: s.Timeout,
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return []byte(""), err
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			logError(err, "send")
		}
	}(resp.Body)

	var errReturn error
	if resp.StatusCode != http.StatusOK {
		errReturn = errors.Join(errReturn, fmt.Errorf("unexpected response status: %s", resp.Status))
	}
	respbody, err := io.ReadAll(resp.Body)
	if err != nil {
		errReturn = errors.Join(errReturn, err)
		return []byte(""), errReturn
	}
	return respbody, errReturn
}

func buildGetStatusPayload(s *Server, namespace string, ClientRequestHandle *string) string {
	var payload strings.Builder
	//header
	payload.WriteString(XmlVersion)
	buildHeader(&payload, namespace)
	//body
	payload.WriteString(fmt.Sprintf(
		"<%s:GetStatus LocaleID=\"%s\" ClientRequestHandle=\"%s\"></%s:GetStatus>",
		namespace, s.LocaleID, *ClientRequestHandle, namespace,
	))

	payload.WriteString(Footer)

	return payload.String()
}

func buildReadPayload(s *Server, ClientRequestHandle *string, ClientItemHandles *[]string, namespace string,
	items []TItem, options map[string]interface{}) string {
	var payload strings.Builder
	//header
	payload.WriteString(XmlVersion)
	buildHeader(&payload, namespace)
	//body
	payload.WriteString(fmt.Sprintf(
		"<%s:Read LocaleID=\"%s\" ClientRequestHandle=\"%s\">", namespace, s.LocaleID, *ClientRequestHandle,
	))
	buildOptionItems(&payload, options, namespace)
	payload.WriteString(fmt.Sprintf("<%s:ItemList>", namespace))
	payload.WriteString(buildReadItems(items, ClientItemHandles, namespace))
	payload.WriteString(fmt.Sprintf("</%s:ItemList></%s:Read>", namespace, namespace))
	payload.WriteString(Footer)

	return payload.String()
}

func buildOptionItems(optionPayload *strings.Builder, options map[string]interface{}, namespace string) string {

	if len(options) > 0 {
		optionPayload.WriteString(fmt.Sprintf("<%s:Options", namespace))
		for key, value := range options {
			if reflect.TypeOf(value).Kind() == reflect.Bool {
				optionPayload.WriteString(
					fmt.Sprintf(" %s=\"%s\"", key, strings.ToLower(fmt.Sprintf("%v", value))),
				)
			} else {
				optionPayload.WriteString(fmt.Sprintf(" %s=\"%v\"", key, value))
			}
		}
		optionPayload.WriteString("/>")
	}

	return optionPayload.String()
}

func buildReadItems(items []TItem, ClientItemHandles *[]string, namespace string) string {
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
	itemPath string, namespace string, options TBrowseOptions) string {
	var payload strings.Builder

	// Header
	payload.WriteString(XmlVersion)
	buildHeader(&payload, namespace)

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
	payload.WriteString(Footer)

	return payload.String()
}

func buildWritePayload(s *Server, namespace string, items []TItem, ClientRequestHandle *string, ClientItemHandles *[]string, options map[string]interface{}) string {
	// make sure all items have a (correct) opc-xml-da type
	items = setOpcXmlDaTypes(items)

	var payload strings.Builder
	//header
	payload.WriteString(XmlVersion)
	buildHeader(&payload, namespace)
	//body
	payload.WriteString(fmt.Sprintf("<%s:Write ReturnValuesOnReply=\"true\">", namespace))
	options["ClientRequestHandle"] = *ClientRequestHandle
	options["LocaleID"] = s.LocaleID
	buildOptionItems(&payload, options, namespace)
	payload.WriteString(fmt.Sprintf("<%s:ItemList>", namespace))
	payload.WriteString(buildWriteItems(items, namespace, ClientItemHandles))
	payload.WriteString(fmt.Sprintf("</%s:ItemList></%s:Write>", namespace, namespace))
	payload.WriteString(Footer)

	return payload.String()
}

func buildWriteItems(items []TItem, namespace string, ClientItemHandles *[]string) string {
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

func buildWriteItemsValue(value TValue, namespace string) string {
	var writeItemsValue strings.Builder

	if valueIsArrayOrSlice(value.Value) {
		vo := reflect.ValueOf(value.Value)
		for i := 0; i < vo.Len(); i++ {
			v := vo.Index(i).Interface()
			valueType, err := getOpcXmlDaType(v)
			if err != nil {
				logError(err, "buildWriteItems_Value")
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

func buildSubscribePayload(namespace string, items []TItem, ClientRequestHandle *string, ClientItemHandles *[]string,
	returnValuesOnReply bool, subscriptionPingRate uint, options map[string]interface{}) string {
	var payload strings.Builder
	//header
	payload.WriteString(XmlVersion)
	buildHeader(&payload, namespace)
	//body

	payload.WriteString(fmt.Sprintf("<%s:Subscribe ReturnValuesOnReply=\"%s\" SubscriptionPingRate=\"%d\" ClientRequestHandle=\"%s\">",
		namespace, strings.ToLower(fmt.Sprintf("%v", returnValuesOnReply)), subscriptionPingRate, *ClientRequestHandle))
	buildOptionItems(&payload, options, namespace)
	payload.WriteString(fmt.Sprintf("<%s:ItemList xsi:type=\"SubscribeRequestItemList\" ItemPath=\"\">", namespace))
	payload.WriteString(buildSubscribeItems(items, ClientItemHandles, namespace))
	payload.WriteString(fmt.Sprintf("</%s:ItemList></%s:Subscribe>", namespace, namespace))
	payload.WriteString(Footer)

	return payload.String()
}

func buildSubscribeItems(items []TItem, ClientItemHandles *[]string, namespace string) string {
	var subscribeItems strings.Builder

	for i, item := range items {
		subscribeItems.WriteString(fmt.Sprintf("<%s:Items xsi:type=\"%s:SubscribeRequestItem\" Deadband=\"%.0f\" ", namespace, namespace, item.DeadBand))
		subscribeItems.WriteString(fmt.Sprintf(
			"RequestedSamplingRate=\"%d\" EnableBuffering=\"%s\"",
			item.RequestedSamplingRate, strings.ToLower(fmt.Sprintf("%t", item.EnableBuffering)),
		))
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
	payload.WriteString(XmlVersion)
	buildHeader(&payload, namespace)
	//body
	payload.WriteString(fmt.Sprintf("<%s:SubscriptionCancel ServerSubHandle=\"%s\" ClientRequestHandle=\"%s\"></%s:SubscriptionCancel>",
		namespace, serverSubHandle, *ClientRequestHandle, namespace))
	payload.WriteString(Footer)

	return payload.String()
}

func buildSubscriptionPolledRefreshPayload(serverSubHandle string, namespace string, ClientRequestHandle *string,
	SubscriptionPingRate uint, options map[string]interface{}, ServerTime TServerTime) (string, error) {
	var payload strings.Builder
	//header
	payload.WriteString(XmlVersion)
	buildHeader(&payload, namespace)
	//body
	holdTime, err := calcHoldTime(SubscriptionPingRate, ServerTime)
	if err != nil {
		return "", err
	}
	payload.WriteString(fmt.Sprintf("<%s:SubscriptionPolledRefresh HoldTime=\"%s\" ReturnAllItems=\"false\" WaitTime=\"500\">", namespace, holdTime))
	options["ClientRequestHandle"] = *ClientRequestHandle
	buildOptionItems(&payload, options, namespace)
	payload.WriteString(fmt.Sprintf("<%s:ServerSubHandles>%s</%s:ServerSubHandles>", namespace, serverSubHandle, namespace))
	payload.WriteString(fmt.Sprintf("</%s:SubscriptionPolledRefresh>", namespace))
	payload.WriteString(Footer)

	return payload.String(), nil
}

func calcHoldTime(subscriptionPingRate uint, ServerTime TServerTime) (string, error) {
	if ServerTime.UseClientTime {
		now := time.Now()
		next := now.Add(time.Duration(subscriptionPingRate) * time.Millisecond)
		return next.Format(time.RFC3339), nil
	} else {
		next := ServerTime.ServerTime.Add(time.Duration(subscriptionPingRate) * time.Millisecond)
		return next.Format(time.RFC3339), nil
	}

}

func buildGetPropertiesPayload(s *Server, ClientRequestHandle *string, namespace string, items []TItem, PropertyOptions TPropertyOptions) string {
	var payload strings.Builder
	//header
	payload.WriteString(XmlVersion)
	buildHeader(&payload, namespace)
	//body
	payload.WriteString(fmt.Sprintf(
		"<%s:GetProperties LocaleID=\"%s\" ClientRequestHandle=\"%s\" ",
		namespace, s.LocaleID, *ClientRequestHandle,
	))
	payload.WriteString(fmt.Sprintf(
		"ReturnAllProperties=\"%s\" ReturnPropertyValues=\"%s\" ReturnErrorText=\"%s\">",
		strconv.FormatBool(PropertyOptions.ReturnAllProperties), strconv.FormatBool(PropertyOptions.ReturnPropertyValues),
		strconv.FormatBool(PropertyOptions.ReturnErrorText),
	))
	for _, item := range items {
		payload.WriteString(fmt.Sprintf("<%s:ItemIDs ItemName=\"%s\" ItemPath=\"%s\"/>", namespace, item.ItemName, item.ItemPath))
	}
	for _, propertyName := range PropertyOptions.PropertyNames {
		payload.WriteString(fmt.Sprintf("<%s:PropertyNames>%s</%s:PropertyNames>", namespace, propertyName, namespace))
	}
	payload.WriteString(fmt.Sprintf("</%s:GetProperties>", namespace))
	payload.WriteString(Footer)

	return payload.String()
}
