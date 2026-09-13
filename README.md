# gopcxmlda
An Implementation of the OPC-XML-DA Protocol in Go. See [https://www.opcconnect.com/xml.php#xmlspec](https://www.opcconnect.com/xml.php#xmlspec) or [here in the repository](https://github.com/dernate/gopcxmlda/blob/master/docs/OPCDataAccessXMLSpecification.pdf)

## Status
The basic functions of the OPC-XML-DA protocol are implemented in gopcxmlda. The project is therefore in maintenance or "bug-fix" status. No new functions are planned.

Supported OPC-XML-DA Methods:
- [x] GetStatus
- [x] Browse
- [x] Read
- [x] Write
- [x] Subscribe
- [x] GetProperties


## Usage

### Basic Procedure
Basic usage is as follows:

```go
package main
import (
    "net/url"
    "time"

    "github.com/dernate/gopcxmlda"
)

func main() {
	_url, _ := url.Parse("http://your.opc-xml-da.server")
	s := gopcxmlda.Server{
		Url:      _url,
		LocaleID: "en-US",
		Timeout:  10 * time.Second,
	}
}
```

### GetStatus
```go
var ClientRequestHandle string
status, err := s.GetStatus(context.Background(), ClientRequestHandle, "ns1")
```

### Browse
```go
options := TBrowseOptions{}
var ClientRequestHandle string
browseResponse, err := s.Browse(context.Background(), "my/OPC/path", ClientRequestHandle, "ns1", options)
```

### Read
```go
items := []TItem{
    {
        ItemName: "my/OPC/path",
    },
    {
        ItemName: "my/OPC/path2",
    },
}
options := map[string]string{
    "ReturnItemTime": true,
	"ReturnItemPath": true,
}
var ClientRequestHandle string
var ClientItemHandles []string
readResponse, err := s.Read(context.Background(), items, ClientRequestHandle, ClientItemHandles, "ns1", options)
```

#### Data freshness (MaxAge)
A server may answer a `Read` from its cache. `MaxAge` is the mechanism the specification
provides to bound how stale that value may be: a maximum age in milliseconds, where `0`
requests the most accurate data available (a device read).

Set it per item, or for the whole item list via the `"MaxAge"` option key. An item's own
value overrides the list-level one:

```go
items := []TItem{
    {
        ItemName: "my/OPC/path",
        MaxAge:   gopcxmlda.MaxAgeDevice(), // never accept a cached value
    },
    {
        ItemName: "my/OPC/path2",
        MaxAge:   gopcxmlda.MaxAgeMillis(500), // a cached value up to 500 ms old is fine
    },
    {
        ItemName: "my/OPC/path3", // no MaxAge: falls back to the list-level value below
    },
}
options := map[string]interface{}{
    "ReturnItemTime": true,
    "MaxAge":         1000,
}
```

Omitting `MaxAge` at both levels means the same as `0` per the specification, but leaves
it to the server to actually implement that default.

### Write
```go
items := []TItem{
    {
        ItemName: "my/OPC/path",
        Value: TValue{
            Value: 123,
        },
    },
}
options := map[string]string{}
var ClientRequestHandle string
var ClientItemHandles []string
writeResponse, err := s.Write(context.Background(), items, ClientRequestHandle, ClientItemHandles, "ns1", options)
```

### Subscribe
```go
items := []TItem{
    {
        ItemName: "my/OPC/path",
    },
}
options := map[string]string{
    "ReturnItemTime": true,
    "ReturnItemPath": true,
    "ReturnItemName": true,
}
var ClientRequestHandle string
var ClientItemHandles []string
SubscriptionPingRate := 5000
subscribeResponse, err := s.Subscribe(context.Background(), items, ClientRequestHandle, ClientItemHandles, "ns1", true, SubscriptionPingRate, false, options)
// for the SubscriptionPolledRefresh and SubscriptionCancel functionality see client_test.go
```

### GetProperties
```go
items := []TItem{
    {
        ItemName: "my/OPC/path",
    },
}
propertyOptions := TPropertyOptions{
    ReturnAllProperties:  true,
    ReturnPropertyValues: true,
    ReturnErrorText:      true,
}
var ClientRequestHandle string
properties, err := s.GetProperties(context.Background(), items, propertyOptions, &ClientRequestHandle, "ns1")
```