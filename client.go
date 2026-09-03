package gopcxmlda

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

// resolveRequestHandle defaults namespace to "ns0" and, if ClientRequestHandle points
// to an empty string, fills it in with a freshly generated one. It returns an error -
// instead of the caller panicking on a nil dereference - if ClientRequestHandle is
// nil, which every exported method used to dereference unconditionally.
func resolveRequestHandle(namespace string, ClientRequestHandle *string) (string, error) {
	if ClientRequestHandle == nil {
		return namespace, errors.New("gopcxmlda: ClientRequestHandle must not be nil")
	}
	if namespace == "" {
		namespace = "ns0"
	}
	if *ClientRequestHandle == "" {
		clientRequestHandle, _, err := GenerateClientHandles(0)
		if err != nil {
			return namespace, err
		}
		*ClientRequestHandle = clientRequestHandle
	}
	return namespace, nil
}

// resolveRequestAndItemHandles is resolveRequestHandle plus generation of
// ClientItemHandles (one per item) when the caller didn't supply any. It also
// validates that a caller-supplied ClientItemHandles has exactly one handle per item;
// a mismatched length would otherwise cause an index-out-of-range panic deep inside
// payload construction.
func resolveRequestAndItemHandles(namespace string, ClientRequestHandle *string, ClientItemHandles *[]string,
	itemCount int) (string, error) {
	if ClientRequestHandle == nil {
		return namespace, errors.New("gopcxmlda: ClientRequestHandle must not be nil")
	}
	if ClientItemHandles == nil {
		return namespace, errors.New("gopcxmlda: ClientItemHandles must not be nil")
	}
	if namespace == "" {
		namespace = "ns0"
	}
	if *ClientRequestHandle == "" || len(*ClientItemHandles) == 0 {
		clientRequestHandle, clientItemHandles, err := GenerateClientHandles(itemCount)
		if err != nil {
			return namespace, err
		}
		if *ClientRequestHandle == "" {
			*ClientRequestHandle = clientRequestHandle
		}
		if len(*ClientItemHandles) == 0 {
			*ClientItemHandles = clientItemHandles
		}
	}
	if len(*ClientItemHandles) != itemCount {
		return namespace, fmt.Errorf("gopcxmlda: got %d ClientItemHandles for %d items, lengths must match",
			len(*ClientItemHandles), itemCount)
	}
	return namespace, nil
}

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
//			 s := Server{Url: _url, LocaleID: "en-US", Timeout: 10 * time.Second}
//	         var ClientRequestHandle string
//				response, err := s.GetStatus(context.Background(), &ClientRequestHandle, "")
//				if err != nil {
//					log.Fatal(err)
//				} else {
//					// do something with the response-object TGetStatus
//				}
func (s *Server) GetStatus(ctx context.Context, ClientRequestHandle *string, namespace string) (TGetStatus, error) {
	namespace, err := resolveRequestHandle(namespace, ClientRequestHandle)
	if err != nil {
		logError(err, "GetStatus")
		return TGetStatus{}, err
	}
	payload, err := buildGetStatusPayload(s, namespace, ClientRequestHandle)
	if err != nil {
		logError(err, "GetStatus")
		return TGetStatus{}, err
	}
	return doRequest[TGetStatus](ctx, s, payload, "GetStatus")
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
//			 s := Server{Url: _url, LocaleID: "en-US", Timeout: 10 * time.Second}
//				items := []TItem{
//					{
//						ItemName: "My/Item",
//					},
//				}
//				options := map[string]interface{}{
//					"ReturnItemTime": true,
//					"returnItemPath": true,
//				}
//	         var ClientRequestHandle string
//	         var ClientItemHandles []string
//				response, err := s.Read(context.Background(), items, &ClientRequestHandle, &ClientItemHandles, "", options)
//				if err != nil {
//					log.Fatal(err)
//				} else {
//					// do something with the response-object TRead
//				}
func (s *Server) Read(ctx context.Context, items []TItem, ClientRequestHandle *string, ClientItemHandles *[]string,
	namespace string, options map[string]interface{}) (TRead, error) {
	namespace, err := resolveRequestAndItemHandles(namespace, ClientRequestHandle, ClientItemHandles, len(items))
	if err != nil {
		logError(err, "Read")
		return TRead{}, err
	}
	payload, err := buildReadPayload(s, ClientRequestHandle, ClientItemHandles, namespace, items, options)
	if err != nil {
		logError(err, "Read")
		return TRead{}, err
	}
	return doRequest[TRead](ctx, s, payload, "Read")
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
//			 s := Server{Url: _url, LocaleID: "en-US", Timeout: 10 * time.Second}
//	         var ClientRequestHandle string
//				response, err := s.Browse(context.Background(), "My/Item", &ClientRequestHandle, "", TBrowseOptions{})
//				if err != nil {
//					log.Fatal(err)
//				} else {
//					// do something with the response-object TBrowse
//				}
func (s *Server) Browse(ctx context.Context, itemPath string, ClientRequestHandle *string,
	namespace string, options TBrowseOptions) (TBrowse, error) {
	namespace, err := resolveRequestHandle(namespace, ClientRequestHandle)
	if err != nil {
		logError(err, "Browse")
		return TBrowse{}, err
	}
	payload, err := buildBrowsePayload(s, ClientRequestHandle, itemPath, namespace, options)
	if err != nil {
		logError(err, "Browse")
		return TBrowse{}, err
	}
	return doRequest[TBrowse](ctx, s, payload, "Browse")
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
// - (TWrite): The write result as a TWrite struct.
// - (error): An error if any issues occur during the request.
//
// Example:
//
//		  _url, _ := url.Parse("http://opc-addr-or-IP.local:8080")
//			 s := Server{Url: _url, LocaleID: "en-US", Timeout: 10 * time.Second}
//			 items := []TItem{
//				{
//					ItemName: "My/Item",
//			 		Value: TValue{
//			 			Value: []int{0, 0, 0},
//			 		},
//			 	},
//				{
//					ItemName: "My/Item2",
//			 		Value: TValue{
//			 			Value: 1.234,
//			 		},
//			 	},
//			 }
//	      var ClientRequestHandle string
//	      var ClientItemHandles []string
//			 response, err := s.Write(context.Background(), items, &ClientRequestHandle, &ClientItemHandles, "", map[string]interface{}{})
//			 if err != nil {
//				log.Fatal(err)
//			 } else {
//			 	// do something with the response-object TWrite
//			 }
func (s *Server) Write(ctx context.Context, items []TItem, ClientRequestHandle *string, ClientItemHandles *[]string,
	namespace string, options map[string]interface{}) (TWrite, error) {
	namespace, err := resolveRequestAndItemHandles(namespace, ClientRequestHandle, ClientItemHandles, len(items))
	if err != nil {
		logError(err, "Write")
		return TWrite{}, err
	}
	payload, err := buildWritePayload(s, namespace, items, ClientRequestHandle, ClientItemHandles, options)
	if err != nil {
		logError(err, "Write")
		return TWrite{}, err
	}
	return doRequest[TWrite](ctx, s, payload, "Write")
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
// - options: A map of additional options for the subscription.
//
// Returns:
// - TSubscribe: The subscription object.
// - error: An error object if an error occurs.
//
// Example:
//
//		  _url, _ := url.Parse("http://opc-addr-or-IP.local:8080")
//			 s := Server{Url: _url, LocaleID: "en-US", Timeout: 10 * time.Second}
//			 items := []TItem{
//				 {
//					 ItemName: "My/Item",
//				 },
//			 }
//	  	 var ClientRequestHandle string
//	  	 var ClientItemHandles []string
//			 response, err := s.Subscribe(context.Background(), items, &ClientRequestHandle, &ClientItemHandles, "", false, 0, map[string]interface{}{})
//			 if err != nil {
//				 log.Fatal(err)
//			 } else {
//				 // do something with the response-object TSubscribe
//			 }
func (s *Server) Subscribe(ctx context.Context, items []TItem, ClientRequestHandle *string, ClientItemHandles *[]string,
	namespace string, returnValuesOnReply bool, subscriptionPingRate uint,
	options map[string]interface{}) (TSubscribe, error) {
	namespace, err := resolveRequestAndItemHandles(namespace, ClientRequestHandle, ClientItemHandles, len(items))
	if err != nil {
		logError(err, "Subscribe")
		return TSubscribe{}, err
	}
	payload, err := buildSubscribePayload(namespace, items, ClientRequestHandle, ClientItemHandles,
		returnValuesOnReply, subscriptionPingRate, options)
	if err != nil {
		logError(err, "Subscribe")
		return TSubscribe{}, err
	}
	return doRequest[TSubscribe](ctx, s, payload, "Subscribe")
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
//	     s := Server{Url: _url, LocaleID: "en-US", Timeout: 10 * time.Second}
//	     var ClientRequestHandle string
//			success, err := s.SubscriptionCancel(context.Background(), "subHandle123", "ns1", &ClientRequestHandle)
//			if err != nil {
//			    // Handle error
//			}
//			if success {
//			    // Handle successful cancellation
//			}
func (s *Server) SubscriptionCancel(ctx context.Context, serverSubHandle string, namespace string, ClientRequestHandle *string) (bool, error) {
	namespace, err := resolveRequestHandle(namespace, ClientRequestHandle)
	if err != nil {
		logError(err, "SubscriptionCancel")
		return false, err
	}
	payload, err := buildSubscriptionCancelPayload(serverSubHandle, namespace, ClientRequestHandle)
	if err != nil {
		logError(err, "SubscriptionCancel")
		return false, err
	}
	_, errReturn := doRequest[TSubscriptionCancel](ctx, s, payload, "SubscriptionCancel")
	return errReturn == nil, errReturn
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
// - ServerTime (TServerTime): The server time to be used in the request.
//
// Returns:
// - TSubscriptionPolledRefresh: The response from the server containing the refreshed subscription details.
// - error: An error object if an error occurred during the process.
//
// Example:
//
//		  _url, _ := url.Parse("http://opc-addr-or-IP.local:8080")
//			 s := Server{Url: _url, LocaleID: "en-US", Timeout: 10 * time.Second}
//	      var ClientRequestHandle string
//			 response, err := s.SubscriptionPolledRefresh(context.Background(), "subHandle123", 1000, "ns1", &ClientRequestHandle, map[string]interface{}{}, TServerTime{})
//			 if err != nil {
//				 log.Fatal(err)
//			 } else {
//				 // do something with the response-object TSubscriptionPolledRefresh
//			 }
func (s *Server) SubscriptionPolledRefresh(ctx context.Context, serverSubHandle string, SubscriptionPingRate uint, namespace string,
	ClientRequestHandle *string, options map[string]interface{}, ServerTime TServerTime) (TSubscriptionPolledRefresh, error) {
	namespace, err := resolveRequestHandle(namespace, ClientRequestHandle)
	if err != nil {
		logError(err, "SubscriptionPolledRefresh")
		return TSubscriptionPolledRefresh{}, err
	}
	payload, err := buildSubscriptionPolledRefreshPayload(serverSubHandle, namespace, ClientRequestHandle,
		SubscriptionPingRate, options, ServerTime)
	if err != nil {
		logError(err, "SubscriptionPolledRefresh")
		return TSubscriptionPolledRefresh{}, err
	}

	SPR, errReturn := doRequest[TSubscriptionPolledRefresh](ctx, s, payload, "SubscriptionPolledRefresh")
	if len(SPR.Response.InvalidServerSubHandles) > 0 {
		errReturn = errors.Join(errReturn, &InvalidServerSubHandlesError{Handles: SPR.Response.InvalidServerSubHandles})
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
//			 s := Server{Url: _url, LocaleID: "en-US", Timeout: 10 * time.Second}
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
//			 response, err := s.GetProperties(context.Background(), items, propertyOptions, &ClientRequestHandle, "")
//			 if err != nil {
//				 log.Fatal(err)
//			 } else {
//				 // do something with the response-object TGetProperties
//			 }
func (s *Server) GetProperties(ctx context.Context, items []TItem, PropertyOptions TPropertyOptions,
	ClientRequestHandle *string, namespace string) (TGetProperties, error) {
	namespace, err := resolveRequestHandle(namespace, ClientRequestHandle)
	if err != nil {
		logError(err, "GetProperties")
		return TGetProperties{}, err
	}
	payload, err := buildGetPropertiesPayload(s, ClientRequestHandle, namespace, items, PropertyOptions)
	if err != nil {
		logError(err, "GetProperties")
		return TGetProperties{}, err
	}
	return doRequest[TGetProperties](ctx, s, payload, "GetProperties")
}
