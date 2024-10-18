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
    "github.com/dernate/gopcxmlda"
)

func main() {
	_url, _ := url.Parse("http://your.opc-xml-da.server"),
	s := Server{
		_url,
		"en-US", 
		10,
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