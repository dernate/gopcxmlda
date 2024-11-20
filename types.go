package gopcxmlda

import (
	"net/url"
	"time"
)

// Server represents a server connection with address, port, locale ID, and timeout.
type Server struct {
	Url      *url.URL      // URL of the server
	LocaleID string        // Locale ID of the server
	Timeout  time.Duration // Timeout duration for the connection
}

type TBaseResult struct {
	ServerState         string    `xml:"ServerState,attr"`
	RevisedLocaleID     string    `xml:"RevisedLocaleID,attr"`
	ReplyTime           time.Time `xml:"ReplyTime,attr"`
	ReceiveTime         time.Time `xml:"RcvTime,attr"`
	ClientRequestHandle string    `xml:"ClientRequestHandle,attr"`
}

type TBodyBase struct {
	Fault TSoapError `xml:"Body>Fault"`
	//Id    string     `xml:"Body>id,attr"`
}

// TGetStatus represents the structure for getting the status of the server.
type TGetStatus struct {
	TBodyBase
	Response TGetStatusResponse `xml:"Body>GetStatusResponse"`
}

type TGetStatusResponse struct {
	Result TBaseResult `xml:"GetStatusResult"`
	Status TStatus     `xml:"Status"`
	Errors OpcErrors   `xml:"Errors"`
}

type TStatus struct {
	ProductVersion             string `xml:"ProductVersion,attr"`
	StartTime                  string `xml:"StartTime,attr"`
	StatusInfo                 string `xml:"StatusInfo"`
	VendorInfo                 string `xml:"VendorInfo"`
	SupportedLocaleIDs         string `xml:"SupportedLocaleIDs"`
	SupportedInterfaceVersions string `xml:"SupportedInterfaceVersions"`
}

// TRead represents the structure for reading values from the server.
type TRead struct {
	TBodyBase
	Response TReadResponseR `xml:"Body>ReadResponse"`
}

type TReadResponseR struct {
	Result   TBaseResult `xml:"ReadResult"`
	ItemList TItemList   `xml:"RItemList"`
	Errors   OpcErrors   `xml:"Errors"`
}

type TItemList struct {
	Items []TItem `xml:"Items"`
}

// TItem represents the structure for an item.
type TItem struct {
	Timestamp             time.Time `xml:"Timestamp,attr"`
	ClientItemHandle      string    `xml:"ClientItemHandle,attr"`
	ItemName              string    `xml:"ItemName,attr"`
	Value                 TValue    `xml:"Value"`
	Quality               TQuality  `xml:"Quality"`
	ItemPath              string    `xml:"ItemPath,attr"`
	Error                 string    `xml:"ResultID,attr"`
	RequestedSamplingRate uint
	EnableBuffering       bool
	DeadBand              float64
}

// TValue represents the structure for the value of an item.
type TValue struct {
	Type      string `xml:"type,attr"` // Can be set manually to force a specific type
	Value     interface{}
	Namespace string
}

// TQuality represents the structure for the quality of an item.
type TQuality struct {
	VendorField  string `xml:"VendorField,attr"`
	LimitField   string `xml:"LimitField,attr"`
	QualityField string `xml:"QualityField,attr"`
}

// TBrowseOptions represents the structure for the browse options.
type TBrowseOptions struct {
	ItemName             string
	ClientRequestHandle  string
	ContinuationPoint    string
	MaxElementsReturned  int
	BrowseFilter         string
	ElementNameFilter    string
	VendorFilter         string
	ReturnAllProperties  bool
	ReturnPropertyValues bool
	ReturnErrorText      bool
}

// TBrowse represents the structure for browsing items on the server.
type TBrowse struct {
	TBodyBase
	Response TBrowseResponse `xml:"Body>BrowseResponse"`
}

type TBrowseResponse struct {
	MoreElements      string           `xml:"MoreElements,attr"`
	ContinuationPoint string           `xml:"ContinuationPoint,attr"`
	Result            TBaseResult      `xml:"BrowseResult"`
	Elements          []TBrowseElement `xml:"Elements"`
	Errors            OpcErrors        `xml:"Errors"`
}

type TBrowseElement struct {
	HasChildren bool   `xml:"HasChildren,attr"`
	IsItem      bool   `xml:"IsItem,attr"`
	Name        string `xml:"Name,attr"`
	ItemName    string `xml:"ItemName,attr"`
	ItemPath    string `xml:"ItemPath,attr"`
}

type TWrite struct {
	TBodyBase
	Response TWriteResponse `xml:"Body>WriteResponse"`
}

type TWriteResponse struct {
	Result   TBaseResult `xml:"WriteResult"`
	ItemList TItemList   `xml:"RItemList"`
	Errors   OpcErrors   `xml:"Errors"`
}

type TSubscribe struct {
	TBodyBase
	Response TSubscribeResponse `xml:"Body>SubscribeResponse"`
}

type TSubscribeResponse struct {
	ServerSubHandle string      `xml:"ServerSubHandle,attr"`
	Result          TBaseResult `xml:"SubscribeResult"`
	ItemList        TItemListS  `xml:"RItemList"`
	Errors          OpcErrors   `xml:"Errors"`
}

type TItemListS struct {
	RevisedSamplingRate int                   `xml:"RevisedSamplingRate,attr"`
	Items               []TSubscribeItemValue `xml:"Items"`
}

type TSubscribeItemValue struct {
	RevisedSamplingRate int   `xml:"RevisedSamplingRate,attr"`
	ItemValue           TItem `xml:"ItemValue"`
}

type TSubscriptionCancel struct {
	TBodyBase
	Response TResponseSC `xml:"Body>https://opcfoundation.org/webservices/XMLDA/1.0/ SubscriptionCancelResponse"`
}

type TResponseSC struct {
	ClientRequestHandle string    `xml:"ClientRequestHandle,attr"`
	Errors              OpcErrors `xml:"Errors"`
}

type TSoapError struct {
	FaultCode   string `xml:"faultcode"`
	FaultString string `xml:"faultstring"`
	Detail      string `xml:"detail"`
}

type TSubscriptionPolledRefresh struct {
	TBodyBase
	Response TResponseSPR `xml:"Body>SubscriptionPolledRefreshResponse"`
}

type TResponseSPR struct {
	DataBufferOverflow      bool         `xml:"DataBufferOverflow,attr"`
	Result                  TBaseResult  `xml:"SubscriptionPolledRefreshResult"`
	ItemList                TItemListSPR `xml:"RItemList"`
	Errors                  OpcErrors    `xml:"Errors"`
	InvalidServerSubHandles []string     `xml:"InvalidServerSubHandles"`
}

type TItemListSPR struct {
	SubscriptionHandle string  `xml:"SubscriptionHandle,attr"`
	Items              []TItem `xml:"Items"`
}

type TGetProperties struct {
	TBodyBase
	Response TGetPropertiesResponse `xml:"Body>GetPropertiesResponse"`
}

type TGetPropertiesResponse struct {
	Result       TBaseResult     `xml:"GetPropertiesResult"`
	PropertyList []TPropertyList `xml:"PropertyLists"`
	Errors       OpcErrors       `xml:"Errors"`
}

type TPropertyList struct {
	ItemName   string        `xml:"ItemName,attr"`
	ItemPath   string        `xml:"ItemPath,attr"`
	Type       string        `xml:"type,attr"`
	ResultId   string        `xml:"ResultID,attr"`
	Properties []TProperties `xml:"Properties"`
}

type TProperties struct {
	Description string `xml:"Description,attr"`
	ItemName    string `xml:"ItemName,attr"`
	ItemPath    string `xml:"ItemPath,attr"`
	Name        string `xml:"Name,attr"`
	Type        string `xml:"type,attr"`
	Value       TValue `xml:"Value"`
}

type OpcErrors struct {
	Id   string   `xml:"ID,attr"`
	Type string   `xml:"type,attr"`
	Text []string `xml:"Text"`
}

type TServerTime struct {
	ServerTime    time.Time
	UseClientTime bool
}

type TPropertyOptions struct {
	ReturnAllProperties  bool
	PropertyNames        []string
	ReturnPropertyValues bool
	ReturnErrorText      bool
}
