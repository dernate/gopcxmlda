package gopcxmlda

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/xml"
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
// It returns the status of the client request as a T_GetStatus struct and an error, if any.
//
// Parameters:
// - ClientRequestHandle (*string): The client request handle to use for the request.
// - namespace (string): The namespace to use for the request.
//
// Returns:
// - (T_GetStatus): The status of the OPC-XML-DA Server as a T_GetStatus struct.
// - (error): An error if any issues occur during the request.
//
// Example:
//
//	 s := Server{"opc-addr-or-IP.local", 12345, "en-US", 10}
//		response, ClientRequestHandle, err := s.GetStatus("", "")
//		if err != nil {
//			log.Fatal(err)
//		} else {
//			// do something with the response-object T_GetStatus
//		}
func (s *Server) GetStatus(ClientRequestHandle *string, namespace string) (T_GetStatus, error) {
	if namespace == "" {
		namespace = "ns0"
	}
	if *ClientRequestHandle == "" {
		clientRequestHandle, _, err := GenerateClientHandles(0)
		if err != nil {
			logError(err.Error(), "GetStatus")
			return T_GetStatus{}, err
		}
		*ClientRequestHandle = clientRequestHandle
	}
	payload := buildGetStatusPayload(s, namespace, ClientRequestHandle)
	logDebug("send payload", "GetStatus", payload)

	response, err := send(s, payload, s.Timeout)
	if err != nil {
		logError(err.Error(), "GetStatus")
		return T_GetStatus{}, err
	}

	var Status T_GetStatus
	if err := xml.Unmarshal(response, &Status); err != nil {
		logError(err.Error(), "GetStatus")
	}

	return Status, nil
}

// Read reads items from the specified namespace using the given options.
// It takes the items, namespace, and options as parameters.
// It returns the read result and an error if any.
//
// Parameters:
// - items ([]T_Item): The items to read from the server.
// - ClientRequestHandle (*string): The client request handle to use for the request.
// - ClientItemHandles (*[]string): The client item handles to use for the request.
// - namespace (string): The namespace to use for the request.
// - options (map[string]string): The options to use for the request.
//
// Returns:
// - (T_Read): The read result as a T_Read struct.
// - (error): An error if any issues occur during the request.
//
// Example:
//
//	 s := Server{"opc-addr-or-IP.local", 12345, "en-US", 10}
//		items := []T_Item{
//			{
//				ItemName: "My/Item",
//			},
//		}
//		options := map[string]string{
//			"ReturnItemTime": "true",
//			"returnItemPath": "true",
//		}
//		response, err := s.Read(items, "", options)
//		if err != nil {
//			log.Fatal(err)
//		} else {
//			// do something with the response-object T_Read
//		}
func (s *Server) Read(items []T_Item, ClientRequestHandle *string, ClientItemHandles *[]string,
	namespace string, options map[string]interface{}) (T_Read, error) {
	if namespace == "" {
		namespace = "ns0"
	}
	if *ClientRequestHandle == "" || len(*ClientItemHandles) == 0 {
		clientRequestHandle, clientItemHandles, err := GenerateClientHandles(len(items))
		if err != nil {
			logError(err.Error(), "Read")
			return T_Read{}, err
		}
		if *ClientRequestHandle == "" {
			*ClientRequestHandle = clientRequestHandle
		}
		if len(*ClientItemHandles) == 0 {
			*ClientItemHandles = clientItemHandles
		}
	}
	payload := buildReadPayload(s, ClientRequestHandle, ClientItemHandles, namespace, items, options)
	logDebug("send payload", "Read", payload)

	response, err := send(s, payload, s.Timeout)
	if err != nil {
		logError(err.Error(), "Read")
		return T_Read{}, err
	}

	var R T_Read
	if err := xml.Unmarshal(response, &R); err != nil {
		logError(err.Error(), "Read")
		return T_Read{}, err
	}

	return R, nil
}

// Browse sends a browse request to the server and returns the browse response.
// It takes the itemPath, namespace, and options as parameters.
// The itemPath specifies the path of the item to browse.
// It returns the browse response and an error if any.
//
// Parameters:
// - itemPath (string): The path of the item to browse.
// - ClientRequestHandle (*string): The client request handle to use for the request.
// - namespace (string): The namespace to use for the request.
// - options (T_BrowseOptions): The options to use for the request.
//
// Returns:
// - (T_Browse): The browse response as a T_Browse struct.
// - (error): An error if any issues occur during the request.
//
// Example:
//
//	 s := Server{"opc-addr-or-IP.local", 12345, "en-US", 10}
//		response, err := s.Browse("My/Item", "", T_BrowseOptions{})
//		if err != nil {
//			log.Fatal(err)
//		} else {
//			// do something with the response-object T_Browse
//		}
func (s *Server) Browse(itemPath string, ClientRequestHandle *string,
	namespace string, options T_BrowseOptions) (T_Browse, error) {
	if namespace == "" {
		namespace = "ns0"
	}
	if *ClientRequestHandle == "" {
		clientRequestHandle, _, err := GenerateClientHandles(0)
		if err != nil {
			logError(err.Error(), "Browse")
			return T_Browse{}, err
		}
		*ClientRequestHandle = clientRequestHandle
	}
	payload := buildBrowsePayload(s, ClientRequestHandle, itemPath, namespace, options)
	logDebug("send payload", "Browse", payload)

	response, err := send(s, payload, s.Timeout)
	if err != nil {
		logError(err.Error(), "Browse")
		return T_Browse{}, err
	}

	var B T_Browse
	if err := xml.Unmarshal(response, &B); err != nil {
		logError(err.Error(), "Browse")
		return T_Browse{}, err
	}

	return B, nil
}

// Write items to the specified namespace using the given options.
// It takes the items, namespace, and options as parameters.
// It returns the write result and an error if any.
//
// Parameters:
// - items ([]T_Item): The items to write to the server.
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
//	 s := Server{"opc-addr-or-IP.local", 12345, "en-US", 10}
//	 items := []T_Item{
//		{
//			ItemName: "My/Item",
//	 		Value: T_Value{
//	 			Value: []int{0, 0, 0},
//	 		},
//	 	},
//		{
//			ItemName: "My/Item2",
//	 		Value: T_Value{
//	 			Value: 1.234,
//	 		},
//	 	},
//	 }
//	 response, err := s.Write(items, "", map[string]string{})
//	 if err != nil {
//		t.Fatal(err)
//	 } else {
//	 	t.Log(response)
//	 }
func (s *Server) Write(items []T_Item, ClientRequestHandle *string, ClientItemHandles *[]string,
	namespace string, options map[string]interface{}) (T_Write, error) {
	if namespace == "" {
		namespace = "ns0"
	}
	if *ClientRequestHandle == "" || len(*ClientItemHandles) == 0 {
		clientRequestHandle, clientItemHandles, err := GenerateClientHandles(len(items))
		if err != nil {
			logError(err.Error(), "Read")
			return T_Write{}, err
		}
		if *ClientRequestHandle == "" {
			*ClientRequestHandle = clientRequestHandle
		}
		if len(*ClientItemHandles) == 0 {
			*ClientItemHandles = clientItemHandles
		}
	}
	payload := buildWritePayload(s, namespace, items, ClientRequestHandle, ClientItemHandles, options)
	logDebug("send payload", "Write", payload)

	response, err := send(s, payload, s.Timeout)
	if err != nil {
		logError(err.Error(), "Write")
		return T_Write{}, err
	}

	var W T_Write
	if err := xml.Unmarshal(response, &W); err != nil {
		logError(err.Error(), "Write")
		return T_Write{}, err
	}

	return W, nil
}

// Subscribe subscribes a client to a set of items, enabling the client to receive updates about the items' states.
//
// Parameters:
// - items: A slice of T_Item representing the items to be subscribed to.
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
//	 s := Server{"opc-addr-or-IP.local", 12345, "en-US", 10}
//	 items := []T_Item{
//		 {
//			 ItemName: "My/Item",
//		 },
//	 }
//	 response, err := s.Subscribe(items, "", "", "", false, 0, false, map[string]interface{})
//	 if err != nil {
//		 log.Fatal(err)
//	 } else {
//		 // do something with the response-object T_Subscribe
//	 }
func (s *Server) Subscribe(items []T_Item, ClientRequestHandle *string, ClientItemHandles *[]string,
	namespace string, returnValuesOnReply bool, subscriptionPingRate uint, enableBuffering bool,
	options map[string]interface{}) (T_Subscribe, error) {
	if namespace == "" {
		namespace = "ns0"
	}
	if *ClientRequestHandle == "" || len(*ClientItemHandles) == 0 {
		clientRequestHandle, clientItemHandles, err := GenerateClientHandles(len(items))
		if err != nil {
			logError(err.Error(), "Read")
			return T_Subscribe{}, err
		}
		if *ClientRequestHandle == "" {
			*ClientRequestHandle = clientRequestHandle
		}
		if len(*ClientItemHandles) == 0 {
			*ClientItemHandles = clientItemHandles
		}
	}
	payload := buildSubscribePayload(namespace, items, ClientRequestHandle, ClientItemHandles,
		returnValuesOnReply, subscriptionPingRate, enableBuffering, options)
	logDebug("send payload", "Subscribe", payload)

	response, err := send(s, payload, s.Timeout)
	if err != nil {
		logError(err.Error(), "Subscribe")
		return T_Subscribe{}, err
	}

	var S T_Subscribe
	if err := xml.Unmarshal(response, &S); err != nil {
		logError(err.Error(), "Subscribe")
		return T_Subscribe{}, err
	}

	return S, nil
}

// SubscriptionCancel cancels a subscription on the server.
//
// Parameters:
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
//	success, err := s.SubscriptionCancel("subHandle123", "ns1", &clientHandle)
//	if err != nil {
//	    // Handle error
//	}
//	if success {
//	    // Handle successful cancellation
//	}
func (s *Server) SubscriptionCancel(serverSubHandle string, namespace string, ClientRequestHandle *string) (bool, error) {
	if namespace == "" {
		namespace = "ns0"
	}
	if *ClientRequestHandle == "" {
		clientRequestHandle, _, err := GenerateClientHandles(0)
		if err != nil {
			logError(err.Error(), "Browse")
			return false, err
		}
		*ClientRequestHandle = clientRequestHandle
	}
	payload := buildSubscriptionCancelPayload(serverSubHandle, namespace, ClientRequestHandle)
	logDebug("send payload", "SubscriptionCancel", payload)

	response, err := send(s, payload, s.Timeout)
	if err != nil {
		logError(err.Error(), "SubscriptionCancel")
		return false, err
	}

	var SC T_SubscriptionCancel
	if err := xml.Unmarshal(response, &SC); err != nil {
		logError(err.Error(), "SubscriptionCancel")
		return false, err
	}

	return true, nil
}

// SubscriptionPolledRefresh is a method of the Server struct that refreshes a subscription
// by sending a polled refresh request to the server.
//
// Parameters:
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
//	 s := Server{"opc-addr-or-IP.local", 12345, "en-US", 10}
//	 response, err := s.SubscriptionPolledRefresh("subHandle123", 1000, "ns1", &clientHandle, map[string]interface{}{}, T_ServerTime{})
//	 if err != nil {
//		 log.Fatal(err)
//	 } else {
//		 // do something with the response-object T_SubscriptionPolledRefresh
//	 }
func (s *Server) SubscriptionPolledRefresh(serverSubHandle string, SubscriptionPingRate uint, namespace string,
	ClientRequestHandle *string, options map[string]interface{}, ServerTime T_ServerTime) (T_SubscriptionPolledRefresh, error) {
	if namespace == "" {
		namespace = "ns0"
	}
	if *ClientRequestHandle == "" {
		clientRequestHandle, _, err := GenerateClientHandles(0)
		if err != nil {
			logError(err.Error(), "Browse")
			return T_SubscriptionPolledRefresh{}, err
		}
		*ClientRequestHandle = clientRequestHandle
	}
	payload, err := buildSubscriptionPolledRefreshPayload(serverSubHandle, namespace, ClientRequestHandle,
		SubscriptionPingRate, options, ServerTime)
	if err != nil {
		logError(err.Error(), "SubscriptionPolledRefresh")
		return T_SubscriptionPolledRefresh{}, err
	}
	logDebug("send payload", "SubscriptionPolledRefresh", payload)

	response, err := send(s, payload, s.Timeout)
	if err != nil {
		logError(err.Error(), "SubscriptionPolledRefresh")
		return T_SubscriptionPolledRefresh{}, err
	}

	var SPR T_SubscriptionPolledRefresh
	if err := xml.Unmarshal([]byte(response), &SPR); err != nil {
		logError(err.Error(), "SubscriptionPolledRefresh")
		return T_SubscriptionPolledRefresh{}, err
	}

	return SPR, nil
}
