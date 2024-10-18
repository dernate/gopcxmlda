package gopcxmlda

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
)

// GenerateClientHandles generates a random ClientRequestHandle and a specified number of ClientItemHandles.
//
// Parameters:
// - count (int): The number of ClientItemHandles to generate.
//
// Returns:
// - (string): A randomly generated ClientRequestHandle consisting of 16 random characters.
// - ([]string): A slice of ClientItemHandles, each uniquely suffixed with an index from 0 to count-1.
// - (error): An error if any issues occur during the generation of random bytes.
//
// Example:
//
//	 clientRequestHandle, clientItemHandles, err := GenerateClientHandles(5)
//	 if err != nil {
//		 log.Fatal(err)
//	 } else {
//		 // do something with the clientRequestHandle and clientItemHandles
//	 }
func GenerateClientHandles(count int) (string, []string, error) {
	// 16 random characters
	length := 16
	bBytes := make([]byte, (length*3+3)/4)
	_, err := rand.Read(bBytes)
	if err != nil {
		return "", nil, err
	}
	randomStr := base64.URLEncoding.EncodeToString(bBytes)
	clientRequestHandle := randomStr[:length]

	var clientItemHandles []string
	for i := 0; i < count; i++ {
		clientItemHandles = append(clientItemHandles, fmt.Sprintf("%sItem_%d", clientRequestHandle, i))
	}
	return clientRequestHandle, clientItemHandles, nil
}

// GetStatus gives back the status of an opc-xml-da server.
// It takes the ClientRequestHandle and namespace as parameters.
// If the namespace is empty, it defaults to "ns0".
// It returns the status of the client request as a TGetStatus struct and an error, if any.
//
// Parameters:
// - ctx (context.Context): The context of the request.
// - ClientRequestHandle (*string): The client request handle to use for the request.
// - namespace (string): The namespace to use for the request.
//
// Returns:
// - (TGetStatus): The status of the OPC-XML-DA Server as a TGetStatus struct.
// - (error): An error if any issues occur during the request.
//
// Example:
//
//		  _url, _ := url.Parse("http://opc-addr-or-IP.local:8080")
//			 s := Server{_url, "en-US", 10}
//	         var ClientRequestHandle string
//				response, ClientRequestHandle, err := s.GetStatus(context.Background, &ClientRequestHandle, "")
//				if err != nil {
//					log.Fatal(err)
//				} else {
//					// do something with the response-object TGetStatus
//				}
func (s *Server) GetStatus(ctx context.Context, ClientRequestHandle *string, namespace string) (TGetStatus, error) {
	if namespace == "" {
		namespace = "ns0"
	}
	if *ClientRequestHandle == "" {
		clientRequestHandle, _, err := GenerateClientHandles(0)
		if err != nil {
			logError(err, "GetStatus")
			return TGetStatus{}, err
		}
		*ClientRequestHandle = clientRequestHandle
	}
	payload := buildGetStatusPayload(s, namespace, ClientRequestHandle)

	var errReturn error
	response, err := send(ctx, s, payload)
	if err != nil {
		errReturn = errors.Join(errReturn, err)
	}

	var Status TGetStatus
	if err = xml.Unmarshal(response, &Status); err != nil {
		errReturn = errors.Join(errReturn, err)
		if errReturn != nil {
			logError(errReturn, "GetStatus")
		}
		return TGetStatus{}, errReturn
	}

	if Status.Fault.FaultCode != "" {
		errReturn = errors.Join(errReturn,
			errors.New(fmt.Sprintf(
				"Faultcode: %s, Faultstring: %s, Detail: %s",
				Status.Fault.FaultCode, Status.Fault.FaultString, Status.Fault.Detail,
			)),
		)
	}
	if Status.Response.Errors.Id != "" {
		errReturn = errors.Join(errReturn,
			errors.New(fmt.Sprintf(
				"Id: %s, Text: %s, Type: %s",
				Status.Response.Errors.Id, Status.Response.Errors.Text, Status.Response.Errors.Type,
			)),
		)
	}

	if errReturn != nil {
		logError(errReturn, "GetStatus")
	}
	return Status, errReturn
}

// Read reads items from the specified namespace using the given options.
// It takes the items, namespace, and options as parameters.
// It returns the read result and an error if any.
//
// Parameters:
// - ctx (context.Context): The context of the request.
// - items ([]TItem): The items to read from the server.
// - ClientRequestHandle (*string): The client request handle to use for the request.
// - ClientItemHandles (*[]string): The client item handles to use for the request.
// - namespace (string): The namespace to use for the request.
// - options (map[string]string): The options to use for the request.
//
// Returns:
// - (TRead): The read result as a TRead struct.
// - (error): An error if any issues occur during the request.
//
// Example:
//
//		  _url, _ := url.Parse("http://opc-addr-or-IP.local:8080")
//			 s := Server{_url, "en-US", 10}
//				items := []TItem{
//					{
//						ItemName: "My/Item",
//					},
//				}
//				options := map[string]string{
//					"ReturnItemTime": "true",
//					"returnItemPath": "true",
//				}
//	         var ClientRequestHandle string
//				response, err := s.Read(context.Background, items, &ClientRequestHandle, options)
//				if err != nil {
//					log.Fatal(err)
//				} else {
//					// do something with the response-object TRead
//				}
func (s *Server) Read(ctx context.Context, items []TItem, ClientRequestHandle *string, ClientItemHandles *[]string,
	namespace string, options map[string]interface{}) (TRead, error) {
	if namespace == "" {
		namespace = "ns0"
	}
	if *ClientRequestHandle == "" || len(*ClientItemHandles) == 0 {
		clientRequestHandle, clientItemHandles, err := GenerateClientHandles(len(items))
		if err != nil {
			logError(err, "Read")
			return TRead{}, err
		}
		if *ClientRequestHandle == "" {
			*ClientRequestHandle = clientRequestHandle
		}
		if len(*ClientItemHandles) == 0 {
			*ClientItemHandles = clientItemHandles
		}
	}
	payload := buildReadPayload(s, ClientRequestHandle, ClientItemHandles, namespace, items, options)

	var errReturn error
	response, err := send(ctx, s, payload)
	if err != nil {
		errReturn = errors.Join(errReturn, err)
	}

	var R TRead
	if err = xml.Unmarshal(response, &R); err != nil {
		errReturn = errors.Join(errReturn, err)
		if errReturn != nil {
			logError(errReturn, "Read")
		}
		return TRead{}, errReturn
	}

	if R.Fault.FaultCode != "" {
		errReturn = errors.Join(errReturn,
			errors.New(fmt.Sprintf(
				"Faultcode: %s, Faultstring: %s, Detail: %s",
				R.Fault.FaultCode, R.Fault.FaultString, R.Fault.Detail,
			)),
		)
	}
	if R.Response.Errors.Id != "" {
		errReturn = errors.Join(errReturn,
			errors.New(fmt.Sprintf(
				"Id: %s, Text: %s, Type: %s",
				R.Response.Errors.Id, R.Response.Errors.Text, R.Response.Errors.Type,
			)),
		)
	}

	if errReturn != nil {
		logError(errReturn, "Read")
	}

	return R, errReturn
}

// Browse sends a browse request to the server and returns the browse response.
// It takes the itemPath, namespace, and options as parameters.
// The itemPath specifies the path of the item to browse.
// It returns the browse response and an error if any.
//
// Parameters:
// - ctx (context.Context): The context of the request.
// - itemPath (string): The path of the item to browse.
// - ClientRequestHandle (*string): The client request handle to use for the request.
// - namespace (string): The namespace to use for the request.
// - options (TBrowseOptions): The options to use for the request.
//
// Returns:
// - (TBrowse): The browse response as a TBrowse struct.
// - (error): An error if any issues occur during the request.
//
// Example:
//
//		  _url, _ := url.Parse("http://opc-addr-or-IP.local:8080")
//			 s := Server{_url, "en-US", 10}
//	         var ClientRequestHandle string
//				response, err := s.Browse(context.Background, "My/Item", &ClientRequestHandle, TBrowseOptions{})
//				if err != nil {
//					log.Fatal(err)
//				} else {
//					// do something with the response-object TBrowse
//				}
func (s *Server) Browse(ctx context.Context, itemPath string, ClientRequestHandle *string,
	namespace string, options TBrowseOptions) (TBrowse, error) {
	if namespace == "" {
		namespace = "ns0"
	}
	if *ClientRequestHandle == "" {
		clientRequestHandle, _, err := GenerateClientHandles(0)
		if err != nil {
			logError(err, "Browse")
			return TBrowse{}, err
		}
		*ClientRequestHandle = clientRequestHandle
	}
	payload := buildBrowsePayload(s, ClientRequestHandle, itemPath, namespace, options)

	var errReturn error
	response, err := send(ctx, s, payload)
	if err != nil {
		errReturn = errors.Join(errReturn, err)
	}

	var B TBrowse
	if err = xml.Unmarshal(response, &B); err != nil {
		errReturn = errors.Join(errReturn, err)
		if errReturn != nil {
			logError(errReturn, "Browse")
		}
		return TBrowse{}, errReturn
	}

	if B.Fault.FaultCode != "" {
		errReturn = errors.Join(errReturn,
			errors.New(fmt.Sprintf(
				"Faultcode: %s, Faultstring: %s, Detail: %s",
				B.Fault.FaultCode, B.Fault.FaultString, B.Fault.Detail,
			)),
		)
	}
	if B.Response.Errors.Id != "" {
		errReturn = errors.Join(errReturn,
			errors.New(fmt.Sprintf(
				"Id: %s, Text: %s, Type: %s",
				B.Response.Errors.Id, B.Response.Errors.Text, B.Response.Errors.Type,
			)),
		)
	}

	if errReturn != nil {
		logError(errReturn, "Browse")
	}

	return B, errReturn
}

// Write items to the specified namespace using the given options.
// It takes the items, namespace, and options as parameters.
// It returns the write result and an error if any.
//
// Parameters:
// - ctx (context.Context): The context of the request.
// - items ([]TItem): The items to write to the server.
// - ClientRequestHandle (*string): The client request handle to use for the request.
// - ClientItemHandles (*[]string): The client item handles to use for the request.
// - namespace (string): The namespace to use for the request.
// - options (map[string]string): The options to use for the request.
//
// Returns:
// - (T_Write): The write result as a T_Write struct.
// - (error): An error if any issues occur during the request.
//
// Example:
//
//		  _url, _ := url.Parse("http://opc-addr-or-IP.local:8080")
//			 s := Server{_url, "en-US", 10}
//			 items := []TItem{
//				{
//					ItemName: "My/Item",
//			 		Value: T_Value{
//			 			Value: []int{0, 0, 0},
//			 		},
//			 	},
//				{
//					ItemName: "My/Item2",
//			 		Value: T_Value{
//			 			Value: 1.234,
//			 		},
//			 	},
//			 }
//	      var ClientRequestHandle string
//			 response, err := s.Write(context.Background, items, &ClientRequestHandle, map[string]string{})
//			 if err != nil {
//				t.Fatal(err)
//			 } else {
//			 	t.Log(response)
//			 }
func (s *Server) Write(ctx context.Context, items []TItem, ClientRequestHandle *string, ClientItemHandles *[]string,
	namespace string, options map[string]interface{}) (TWrite, error) {
	if namespace == "" {
		namespace = "ns0"
	}
	if *ClientRequestHandle == "" || len(*ClientItemHandles) == 0 {
		clientRequestHandle, clientItemHandles, err := GenerateClientHandles(len(items))
		if err != nil {
			logError(err, "Write")
			return TWrite{}, err
		}
		if *ClientRequestHandle == "" {
			*ClientRequestHandle = clientRequestHandle
		}
		if len(*ClientItemHandles) == 0 {
			*ClientItemHandles = clientItemHandles
		}
	}
	payload := buildWritePayload(s, namespace, items, ClientRequestHandle, ClientItemHandles, options)

	var errReturn error
	response, err := send(ctx, s, payload)
	if err != nil {
		errReturn = errors.Join(errReturn, err)
	}

	var W TWrite
	if err = xml.Unmarshal(response, &W); err != nil {
		errReturn = errors.Join(errReturn, err)
		if errReturn != nil {
			logError(errReturn, "Write")
		}
		return TWrite{}, errReturn
	}

	if W.Fault.FaultCode != "" {
		errReturn = errors.Join(errReturn,
			errors.New(fmt.Sprintf(
				"Faultcode: %s, Faultstring: %s, Detail: %s",
				W.Fault.FaultCode, W.Fault.FaultString, W.Fault.Detail,
			)),
		)
	}
	if W.Response.Errors.Id != "" {
		errReturn = errors.Join(errReturn,
			errors.New(fmt.Sprintf(
				"Id: %s, Text: %s, Type: %s",
				W.Response.Errors.Id, W.Response.Errors.Text, W.Response.Errors.Type,
			)),
		)
	}

	if errReturn != nil {
		logError(errReturn, "Write")
	}

	return W, errReturn
}

// Subscribe subscribes a client to a set of items, enabling the client to receive updates about the items' states.
//
// Parameters:
// - ctx (context.Context): The context of the request.
// - items: A slice of TItem representing the items to be subscribed to.
// - ClientRequestHandle: A pointer to a string representing the client request handle. If not provided, it will be generated.
// - ClientItemHandles: A pointer to a slice of strings representing the client item handles. If not provided, they will be generated.
// - namespace: A string representing the namespace. Defaults to "ns0" if not provided.
// - returnValuesOnReply: A boolean indicating whether to return values on reply.
// - subscriptionPingRate: An unsigned integer representing the subscription ping rate.
// - enableBuffering: A boolean indicating whether buffering is enabled.
// - options: A map of additional options for the subscription.
//
// Returns:
// - T_Subscribe: The subscription object.
// - error: An error object if an error occurs.
//
// Example:
//
//		  _url, _ := url.Parse("http://opc-addr-or-IP.local:8080")
//			 s := Server{_url, "en-US", 10}
//			 items := []TItem{
//				 {
//					 ItemName: "My/Item",
//				 },
//			 }
//	  	 var ClientRequestHandle string
//			 response, err := s.Subscribe(context.Background, items, &ClientRequestHandle, "", "", false, 0, false, map[string]interface{})
//			 if err != nil {
//				 log.Fatal(err)
//			 } else {
//				 // do something with the response-object T_Subscribe
//			 }
func (s *Server) Subscribe(ctx context.Context, items []TItem, ClientRequestHandle *string, ClientItemHandles *[]string,
	namespace string, returnValuesOnReply bool, subscriptionPingRate uint,
	options map[string]interface{}) (TSubscribe, error) {
	if namespace == "" {
		namespace = "ns0"
	}
	if *ClientRequestHandle == "" || len(*ClientItemHandles) == 0 {
		clientRequestHandle, clientItemHandles, err := GenerateClientHandles(len(items))
		if err != nil {
			logError(err, "Subscribe")
			return TSubscribe{}, err
		}
		if *ClientRequestHandle == "" {
			*ClientRequestHandle = clientRequestHandle
		}
		if len(*ClientItemHandles) == 0 {
			*ClientItemHandles = clientItemHandles
		}
	}
	payload := buildSubscribePayload(namespace, items, ClientRequestHandle, ClientItemHandles,
		returnValuesOnReply, subscriptionPingRate, options)

	var errReturn error
	response, err := send(ctx, s, payload)
	if err != nil {
		errReturn = errors.Join(errReturn, err)
	}

	var Sub TSubscribe
	if err = xml.Unmarshal(response, &Sub); err != nil {
		errReturn = errors.Join(errReturn, err)
		if errReturn != nil {
			logError(errReturn, "Subscribe")
		}
		return TSubscribe{}, errReturn
	}

	if Sub.Fault.FaultCode != "" {
		errReturn = errors.Join(errReturn,
			errors.New(fmt.Sprintf(
				"Faultcode: %s, Faultstring: %s, Detail: %s",
				Sub.Fault.FaultCode, Sub.Fault.FaultString, Sub.Fault.Detail,
			)),
		)
	}
	if Sub.Response.Errors.Id != "" {
		errReturn = errors.Join(errReturn,
			errors.New(fmt.Sprintf(
				"Id: %s, Text: %s, Type: %s",
				Sub.Response.Errors.Id, Sub.Response.Errors.Text, Sub.Response.Errors.Type,
			)),
		)
	}

	if errReturn != nil {
		logError(errReturn, "Subscribe")
	}

	return Sub, errReturn
}

// SubscriptionCancel cancels a subscription on the server.
//
// Parameters:
// - ctx (context.Context): The context of the request.
// - serverSubHandle (string): The handle of the subscription to be canceled on the server (given by the server).
// - namespace (string): The namespace to be used. If empty, it defaults to "ns0".
// - ClientRequestHandle (*string): A pointer to a client request handle. If the value pointed to is empty, a new handle is generated.
//
// Returns:
// - (bool): True if the subscription cancellation was successful, false otherwise.
// - (error): An error object if an error occurred during the process, nil otherwise.
//
// Example:
//
//		    _url, _ := url.Parse("http://opc-addr-or-IP.local:8080")
//	     s := Server{_url, "en-US", 10}
//	     var clientRequestHandle string
//			success, err := s.SubscriptionCancel(context.Background, "subHandle123", "ns1", &ClientRequestHandle)
//			if err != nil {
//			    // Handle error
//			}
//			if success {
//			    // Handle successful cancellation
//			}
func (s *Server) SubscriptionCancel(ctx context.Context, serverSubHandle string, namespace string, ClientRequestHandle *string) (bool, error) {
	if namespace == "" {
		namespace = "ns0"
	}
	if *ClientRequestHandle == "" {
		clientRequestHandle, _, err := GenerateClientHandles(0)
		if err != nil {
			logError(err, "SubscriptionCancel")
			return false, err
		}
		*ClientRequestHandle = clientRequestHandle
	}
	payload := buildSubscriptionCancelPayload(serverSubHandle, namespace, ClientRequestHandle)

	var errReturn error
	response, err := send(ctx, s, payload)
	if err != nil {
		errReturn = errors.Join(errReturn, err)
	}

	var SC TSubscriptionCancel
	if err = xml.Unmarshal(response, &SC); err != nil {
		errReturn = errors.Join(errReturn, err)
		if errReturn != nil {
			logError(errReturn, "SubscriptionCancel")
		}
		return false, errReturn
	}

	if SC.Fault.FaultCode != "" {
		errReturn = errors.Join(errReturn,
			errors.New(fmt.Sprintf(
				"Faultcode: %s, Faultstring: %s, Detail: %s",
				SC.Fault.FaultCode, SC.Fault.FaultString, SC.Fault.Detail,
			)),
		)
	}
	if SC.Response.Errors.Id != "" {
		errReturn = errors.Join(errReturn,
			errors.New(fmt.Sprintf(
				"Id: %s, Text: %s, Type: %s",
				SC.Response.Errors.Id, SC.Response.Errors.Text, SC.Response.Errors.Type,
			)),
		)
	}

	if errReturn != nil {
		logError(errReturn, "SubscriptionCancel")
	}

	return true, nil
}

// SubscriptionPolledRefresh is a method of the Server struct that refreshes a subscription
// by sending a polled refresh request to the server.
//
// Parameters:
// - ctx (context.Context): The context of the request.
// - serverSubHandle (string): The handle of the server subscription to refresh (given by the server).
// - SubscriptionPingRate (uint): The rate at which the subscription should be pinged.
// - namespace (string): The namespace to be used for the subscription. If empty, defaults to "ns0".
// - ClientRequestHandle (*string): A pointer to a string representing the client request handle. If empty, a new handle will be generated.
// - options (map[string]interface{}): A map of additional options for the subscription refresh request.
// - ServerTime (T_ServerTime): The server time to be used in the request.
//
// Returns:
// - T_SubscriptionPolledRefresh: The response from the server containing the refreshed subscription details.
// - error: An error object if an error occurred during the process.
//
// Example:
//
//		  _url, _ := url.Parse("http://opc-addr-or-IP.local:8080")
//			 s := Server{_url, "en-US", 10}
//	      var ClientRequestHandle string
//			 response, err := s.SubscriptionPolledRefresh(context.Background, "subHandle123", 1000, "ns1", &ClientRequestHandle, map[string]interface{}{}, T_ServerTime{})
//			 if err != nil {
//				 log.Fatal(err)
//			 } else {
//				 // do something with the response-object T_SubscriptionPolledRefresh
//			 }
func (s *Server) SubscriptionPolledRefresh(ctx context.Context, serverSubHandle string, SubscriptionPingRate uint, namespace string,
	ClientRequestHandle *string, options map[string]interface{}, ServerTime TServerTime) (TSubscriptionPolledRefresh, error) {
	if namespace == "" {
		namespace = "ns0"
	}
	if *ClientRequestHandle == "" {
		clientRequestHandle, _, err := GenerateClientHandles(0)
		if err != nil {
			logError(err, "SubscriptionPolledRefresh")
			return TSubscriptionPolledRefresh{}, err
		}
		*ClientRequestHandle = clientRequestHandle
	}
	payload, err := buildSubscriptionPolledRefreshPayload(serverSubHandle, namespace, ClientRequestHandle,
		SubscriptionPingRate, options, ServerTime)
	if err != nil {
		logError(err, "SubscriptionPolledRefresh")
		return TSubscriptionPolledRefresh{}, err
	}

	var errReturn error
	response, err := send(ctx, s, payload)
	if err != nil {
		errReturn = errors.Join(errReturn, err)
	}

	var SPR TSubscriptionPolledRefresh
	if err = xml.Unmarshal(response, &SPR); err != nil {
		errReturn = errors.Join(errReturn, err)
		if errReturn != nil {
			logError(errReturn, "SubscriptionPolledRefresh")
		}
		return TSubscriptionPolledRefresh{}, errReturn
	}

	if SPR.Fault.FaultCode != "" {
		errReturn = errors.Join(errReturn,
			errors.New(fmt.Sprintf(
				"Faultcode: %s, Faultstring: %s, Detail: %s",
				SPR.Fault.FaultCode, SPR.Fault.FaultString, SPR.Fault.Detail,
			)),
		)
	}
	if SPR.Response.Errors.Id != "" {
		errReturn = errors.Join(errReturn,
			errors.New(fmt.Sprintf(
				"Id: %s, Text: %s, Type: %s",
				SPR.Response.Errors.Id, SPR.Response.Errors.Text, SPR.Response.Errors.Type,
			)),
		)
	}

	if errReturn != nil {
		logError(errReturn, "SubscriptionPolledRefresh")
	}

	return SPR, errReturn
}

// GetProperties is a method of the Server struct that retrieves the properties of a set of items from the server.
//
// Parameters:
// - ctx (context.Context): The context of the request.
// - items ([]TItem): A slice of TItem representing the items to retrieve properties for.
// - PropertyOptions (TPropertyOptions): The options for the properties request.
// - ClientRequestHandle (*string): A pointer to a string representing the client request handle. If empty, a new handle will be generated.
// - namespace (string): The namespace to be used for the request. If empty, defaults to "ns0".
//
// Returns:
// - TGetProperties: The response from the server containing the properties of the requested items.
// - error: An error object if an error occurred during the process.
//
// Example:
//
//		     _url, _ := url.Parse("http://opc-addr-or-IP.local:8080")
//			 s := Server{_url, "en-US", 10}
//			 items := []TItem{
//				 {
//					 ItemName: "My/Item",
//				 },
//			 }
//			 propertyOptions := TPropertyOptions{
//		 		ReturnAllProperties:  true,
//				ReturnPropertyValues: true,
//				ReturnErrorText:      true,
//			 }
//	      var ClientRequestHandle string
//			 response, err := s.GetProperties(context.Background, items, propertyOptions, &ClientRequestHandle, "")
//			 if err != nil {
//				 log.Fatal(err)
//			 } else {
//				 // do something with the response-object TGetProperties
//			 }
func (s *Server) GetProperties(ctx context.Context, items []TItem, PropertyOptions TPropertyOptions,
	ClientRequestHandle *string, namespace string) (TGetProperties, error) {
	if namespace == "" {
		namespace = "ns0"
	}
	if *ClientRequestHandle == "" {
		clientRequestHandle, _, err := GenerateClientHandles(len(items))
		if err != nil {
			logError(err, "GetProperties")
			return TGetProperties{}, err
		}
		if *ClientRequestHandle == "" {
			*ClientRequestHandle = clientRequestHandle
		}
	}
	payload := buildGetPropertiesPayload(s, ClientRequestHandle, namespace, items, PropertyOptions)

	var errReturn error
	response, err := send(ctx, s, payload)
	if err != nil {
		errReturn = errors.Join(errReturn, err)
	}

	var P TGetProperties
	if err = xml.Unmarshal(response, &P); err != nil {
		errReturn = errors.Join(errReturn, err)
		if errReturn != nil {
			logError(errReturn, "GetProperties")
		}
		return TGetProperties{}, errReturn
	}

	if P.Fault.FaultCode != "" {
		errReturn = errors.Join(errReturn,
			errors.New(fmt.Sprintf(
				"Faultcode: %s, Faultstring: %s, Detail: %s",
				P.Fault.FaultCode, P.Fault.FaultString, P.Fault.Detail,
			)),
		)
	}
	if P.Response.Errors.Id != "" {
		errReturn = errors.Join(errReturn,
			errors.New(fmt.Sprintf(
				"Id: %s, Text: %s, Type: %s",
				P.Response.Errors.Id, P.Response.Errors.Text, P.Response.Errors.Type,
			)),
		)
	}

	if errReturn != nil {
		logError(errReturn, "GetProperties")
	}

	return P, errReturn
}
