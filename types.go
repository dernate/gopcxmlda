package gopcxmlda

import (
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Server represents a server connection with address, port, locale ID, and timeout.
//
// A *Server is safe for concurrent use by multiple goroutines once constructed: the
// lazy initialization of Client/Timeout on first request is guarded by mu. Do not
// copy a Server after it has been used (the same rule that applies to sync.Mutex and
// http.Client).
type Server struct {
	Url      *url.URL      // URL of the server
	LocaleID string        // Locale ID of the server
	Timeout  time.Duration // Timeout duration for the connection
	Client   *http.Client  // HTTP client used for requests. Created lazily (using Timeout) if nil, and then reused.
	mu       sync.Mutex    // guards lazy initialization of Client/Timeout in send()
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
	ProductVersion string `xml:"ProductVersion,attr"`
	// StartTime is intentionally a string rather than time.Time: unlike the
	// ReplyTime/ReceiveTime/Timestamp attributes (which this library controls end
	// to end), StartTime's exact dateTime format is entirely up to the server
	// implementation and is not guaranteed to be strict RFC3339. Forcing time.Time
	// here would silently turn a parseable value into a zero time.Time on any
	// server whose format doesn't match. Parse it yourself if you need a time.Time.
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

// TItem represents both a request-side item (Read/Write/Subscribe input) and a
// response-side item (populated by the server). Which fields are meaningful depends
// on which direction the TItem travels:
//
//   - Read/Write requests use ItemName, ItemPath and Value (Value.Value/Value.Type on
//     Write; on Read only ItemName/ItemPath need to be set).
//   - Subscribe requests additionally use RequestedSamplingRate, EnableBuffering and
//     DeadBand, which have no effect outside of Subscribe.
//   - Responses (Read/Write/Subscribe/SubscriptionPolledRefresh results) populate
//     Timestamp, ClientItemHandle, Value, Quality and Error; RequestedSamplingRate,
//     EnableBuffering and DeadBand are never set by the server and stay zero-valued.
type TItem struct {
	Timestamp        time.Time `xml:"Timestamp,attr"`
	ClientItemHandle string    `xml:"ClientItemHandle,attr"`
	ItemName         string    `xml:"ItemName,attr"`
	Value            TValue    `xml:"Value"`
	Quality          TQuality  `xml:"Quality"`
	ItemPath         string    `xml:"ItemPath,attr"`
	Error            string    `xml:"ResultID,attr"`

	// Request-only fields, used solely by Subscribe(); ignored by Read()/Write() and
	// never populated on a response.
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
	Response TResponseSC `xml:"Body>SubscriptionCancelResponse"`
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

// soapResponse is implemented by every SOAP response wrapper and lets doRequest
// extract the fault/error information common to all of them, regardless of the
// concrete response type.
type soapResponse interface {
	fault() TSoapError
	responseErrors() OpcErrors
}

func (b TBodyBase) fault() TSoapError { return b.Fault }

func (t TGetStatus) responseErrors() OpcErrors                 { return t.Response.Errors }
func (t TRead) responseErrors() OpcErrors                      { return t.Response.Errors }
func (t TBrowse) responseErrors() OpcErrors                    { return t.Response.Errors }
func (t TWrite) responseErrors() OpcErrors                     { return t.Response.Errors }
func (t TSubscribe) responseErrors() OpcErrors                 { return t.Response.Errors }
func (t TSubscriptionCancel) responseErrors() OpcErrors        { return t.Response.Errors }
func (t TSubscriptionPolledRefresh) responseErrors() OpcErrors { return t.Response.Errors }
func (t TGetProperties) responseErrors() OpcErrors             { return t.Response.Errors }
