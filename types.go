package gopcxmlda

import (
	"time"
)

// Server represents a server connection with address, port, locale ID, and timeout.
type Server struct {
	Addr     string        // Address of the server
	Port     string        // Port number of the server
	LocaleID string        // Locale ID of the server
	timeout  time.Duration // Timeout duration for the connection
}

// T_GetStatus represents the structure for getting the status of the server.
type T_GetStatus struct {
	Body T_Body_GS `xml:"Body"`
}

type T_Body_GS struct {
	GetStatusResponse T_GetStatusResponse_GS `xml:"GetStatusResponse"`
}

type T_GetStatusResponse_GS struct {
	GetStatusResult T_GetStatusResult_GS `xml:"GetStatusResult"`
	Status          T_Status_GS          `xml:"Status"`
}

type T_GetStatusResult_GS struct {
	ServerState         string    `xml:"ServerState,attr"`
	RevisedLocaleID     string    `xml:"RevisedLocaleID,attr"`
	ReplyTime           time.Time `xml:"ReplyTime,attr"`
	RcvTime             time.Time `xml:"RcvTime,attr"`
	ClientRequestHandle string    `xml:"ClientRequestHandle,attr"`
}

type T_Status_GS struct {
	ProductVersion             string `xml:"ProductVersion,attr"`
	StartTime                  string `xml:"StartTime,attr"`
	StatusInfo                 string `xml:"StatusInfo"`
	VendorInfo                 string `xml:"VendorInfo"`
	SupportedLocaleIDs         string `xml:"SupportedLocaleIDs"`
	SupportedInterfaceVersions string `xml:"SupportedInterfaceVersions"`
}

// T_Read represents the structure for reading values from the server.
type T_Read struct {
	Body T_Body_R `xml:"Body"`
}

type T_Body_R struct {
	ReadResponse T_ReadResponse_R `xml:"ReadResponse"`
}

type T_ReadResponse_R struct {
	ReadResult T_ReadResult_R `xml:"ReadResult"`
	RItemList  T_ItemList_R   `xml:"RItemList"`
}

type T_ReadResult_R struct {
	ServerState         string    `xml:"ServerState,attr"`
	RevisedLocaleID     string    `xml:"RevisedLocaleID,attr"`
	ReplyTime           time.Time `xml:"ReplyTime,attr"`
	RcvTime             time.Time `xml:"RcvTime,attr"`
	ClientRequestHandle string    `xml:"ClientRequestHandle,attr"`
}

type T_ItemList_R struct {
	Items []T_Item `xml:"Items"`
}

type T_Item struct {
	Timestamp        time.Time `xml:"Timestamp,attr"`
	ClientItemHandle string    `xml:"ClientItemHandle,attr"`
	ItemName         string    `xml:"ItemName,attr"`
	Value            T_Value   `xml:"Value"`
	Quality          T_Quality `xml:"Quality"`
	ItemPath         string    `xml:"ItemPath,attr"`
	Error            string    `xml:"ResultID,attr"`
}

type T_Value struct {
	Type      string `xml:"type,attr"` // Can be set manually to force a specific type
	Value     interface{}
	Namespace string
}

// T_Quality represents the structure for the quality of an item.
type T_Quality struct {
	VendorField  string `xml:"VendorField,attr"`
	LimitField   string `xml:"LimitField,attr"`
	QualityField string `xml:"QualityField,attr"`
}

// T_BrowseOptions represents the structure for the browse options.
type T_BrowseOptions struct {
	ItemName             string
	clientRequestHandle  string
	continuationPoint    string
	maxElementsReturned  int
	browseFilter         string
	elementNameFilter    string
	vendorFilter         string
	returnAllProperties  bool
	returnPropertyValues bool
	returnErrorText      bool
}

// T_Browse represents the structure for browsing items on the server.
type T_Browse struct {
	Body T_Body_B `xml:"Body"`
}

type T_Body_B struct {
	BrowseResponse T_BrowseResponse_B `xml:"BrowseResponse"`
}

type T_BrowseResponse_B struct {
	MoreElements      string              `xml:"MoreElements,attr"`
	ContinuationPoint string              `xml:"ContinuationPoint,attr"`
	BrowseResult      T_ReplyBase_B       `xml:"BrowseResult"`
	Elements          []T_BrowseElement_B `xml:"Elements"`
}

type T_ReplyBase_B struct {
	ServerState         string    `xml:"ServerState,attr"`
	RevisedLocaleID     string    `xml:"RevisedLocaleID,attr"`
	ReplyTime           time.Time `xml:"ReplyTime,attr"`
	RcvTime             time.Time `xml:"RcvTime,attr"`
	ClientRequestHandle string    `xml:"ClientRequestHandle,attr"`
}

type T_BrowseElement_B struct {
	HasChildren bool   `xml:"HasChildren,attr"`
	IsItem      bool   `xml:"IsItem,attr"`
	Name        string `xml:"Name,attr"`
	ItemName    string `xml:"ItemName,attr"`
	ItemPath    string `xml:"ItemPath,attr"`
}

type T_Write struct {
	Body T_Body_W `xml:"Body"`
}

type T_Body_W struct {
	WriteResponse T_WriteResponse_W `xml:"WriteResponse"`
}

type T_WriteResponse_W struct {
	WriteResult T_ReplyBase_W     `xml:"WriteResult"`
	RItemList   T_ReplyItemList_W `xml:"RItemList"`
}

type T_ReplyBase_W struct {
	ServerState         string    `xml:"ServerState,attr"`
	RevisedLocaleID     string    `xml:"RevisedLocaleID,attr"`
	ReplyTime           time.Time `xml:"ReplyTime,attr"`
	RcvTime             time.Time `xml:"RcvTime,attr"`
	ClientRequestHandle string    `xml:"ClientRequestHandle,attr"`
}

type T_ReplyItemList_W struct {
	Items []T_ItemValue_W `xml:"Items"`
}

type T_ItemValue_W struct {
	ClientItemHandle string    `xml:"ClientItemHandle,attr"`
	Value            T_Value   `xml:"Value"`
	Quality          T_Quality `xml:"Quality"`
}

type T_Subscribe struct {
	Body T_Body_S `xml:"Body"`
}

type T_Body_S struct {
	Response T_SubscribeResponse_S `xml:"SubscribeResponse"`
}

type T_SubscribeResponse_S struct {
	ServerSubHandle string              `xml:"ServerSubHandle,attr"`
	Result          T_SubscribeResult_S `xml:"SubscribeResult"`
	ItemList        T_RItemList_S       `xml:"RItemList"`
}

type T_SubscribeResult_S struct {
	ServerState         string    `xml:"ServerState,attr"`
	RevisedLocaleID     string    `xml:"RevisedLocaleID,attr"`
	ReplyTime           time.Time `xml:"ReplyTime,attr"`
	RcvTime             time.Time `xml:"RcvTime,attr"`
	ClientRequestHandle string    `xml:"ClientRequestHandle,attr"`
}

type T_RItemList_S struct {
	RevisedSamplingRate int                      `xml:"RevisedSamplingRate,attr"`
	Items               []T_SubscribeItemValue_S `xml:"Items"`
}

type T_SubscribeItemValue_S struct {
	RevisedSamplingRate int           `xml:"RevisedSamplingRate,attr"`
	ItemValue           T_ItemValue_S `xml:"ItemValue"`
}

type T_ItemValue_S struct {
	ClientItemHandle string    `xml:"ClientItemHandle,attr"`
	ItemName         string    `xml:"ItemName,attr"`
	ItemPath         string    `xml:"ItemPath,attr"`
	Value            T_Value   `xml:"Value"`
	Quality          T_Quality `xml:"Quality"`
}

type T_SubscriptionCancel struct {
	Body T_Body_SC `xml:"Body"`
}

type T_Body_SC struct {
	SubscriptionCancelResponse *T_Response_SC `xml:"http://opcfoundation.org/webservices/XMLDA/1.0/ SubscriptionCancelResponse"`
	Fault                      *T_Fault_SC    `xml:"Fault"`
}

type T_Response_SC struct {
	ClientRequestHandle string `xml:"ClientRequestHandle,attr"`
}

type T_Fault_SC struct {
	Faultcode   string `xml:"faultcode"`
	Faultstring string `xml:"faultstring"`
	Detail      string `xml:"detail"`
}

type T_SubscriptionPolledRefresh struct {
	Body T_Body_SPR `xml:"Body"`
}

type T_Body_SPR struct {
	SubscriptionPolledRefreshResponse T_Response_SPR `xml:"SubscriptionPolledRefreshResponse"`
}

type T_Response_SPR struct {
	DataBufferOverflow              bool            `xml:"DataBufferOverflow,attr"`
	SubscriptionPolledRefreshResult T_Result_SPR    `xml:"SubscriptionPolledRefreshResult"`
	RItemList                       T_RItemList_SPR `xml:"RItemList"`
}

type T_Result_SPR struct {
	ServerState         string    `xml:"ServerState,attr"`
	RevisedLocaleID     string    `xml:"RevisedLocaleID,attr"`
	ReplyTime           time.Time `xml:"ReplyTime,attr"`
	RcvTime             time.Time `xml:"RcvTime,attr"`
	ClientRequestHandle string    `xml:"ClientRequestHandle,attr"`
}

type T_RItemList_SPR struct {
	SubscriptionHandle string       `xml:"SubscriptionHandle,attr"`
	Items              []T_Item_SPR `xml:"Items"`
}

type T_Item_SPR struct {
	Timestamp        string    `xml:"Timestamp,attr"`
	ClientItemHandle string    `xml:"ClientItemHandle,attr"`
	ItemName         string    `xml:"ItemName,attr"`
	ItemPath         string    `xml:"ItemPath,attr"`
	Value            T_Value   `xml:"Value"`
	Quality          T_Quality `xml:"Quality"`
}

type T_ServerTime struct {
	ServerTime    time.Time
	UseClientTime bool
}
