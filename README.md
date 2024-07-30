# gopcxmlda
An Implementation of the OPC-XML-DA Protocol in Go. See [https://www.opcconnect.com/xml.php#xmlspec](https://www.opcconnect.com/xml.php#xmlspec) or [here in the repository](https://github.com/dernate/gopcxmlda/blob/master/docs/OPCDataAccessXMLSpecification.pdf)

## Status
This is a work in progress. The goal is to implement the OPC-XML-DA protocol for client side interaction with Go. The Basic functionalities are implemented, except for the GetProperties method.

Currently supported:
- [ ] GetStatus
- [ ] Browse
- [ ] Read
- [ ] Write
- [ ] Subscribe

Not yet supported:
- [ ] GetProperties

## Usage

### Basic Procedure
Basic usage is as follows:

```go
package main
import (
    "github.com/dernate/gopcxmlda"
)

func main() {
	s := Server{
		"http://your.opc-xml-da.server", 
		8080, 
		"en-US", 
		10,
	}
}
```

### GetStatus
```go
var ClientRequestHandle string
status, err := s.GetStatus(ClientRequestHandle, "ns1")
```

### Browse
```go
options := T_BrowseOptions{}
var ClientRequestHandle string
browse_response, err := s.Browse("my/OPC/path", ClientRequestHandle, "ns1", options)
```

### Read
```go
items := []T_Item{
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
read_response, err := s.Read(items, ClientRequestHandle, ClientItemHandles, "ns1", options)
```

### Write
```go
items := []T_Item{
    {
        ItemName: "my/OPC/path",
        Value: T_Value{
            Value: 123,
        },
    },
}
options := map[string]string{}
var ClientRequestHandle string
var ClientItemHandles []string
write_response, err := s.Write(items, ClientRequestHandle, ClientItemHandles, "ns1", options)
```

### Subscribe
```go
items := []T_Item{
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
subscribe_response, err := s.Subscribe(items, ClientRequestHandle, ClientItemHandles, "ns1", true, SubscriptionPingRate, false, options)
// for the SubscriptionPolledRefresh and SubscriptionCancel functionality see client_test.go
```
