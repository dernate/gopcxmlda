package gopcxmlda

import "fmt"

// SoapFaultError represents a SOAP-level fault (<SOAP-ENV:Fault>) returned by the
// server. Use errors.As to detect it in an error returned by a Server method.
type SoapFaultError struct {
	FaultCode   string
	FaultString string
	Detail      string
}

func (e *SoapFaultError) Error() string {
	return fmt.Sprintf("Faultcode: %s, Faultstring: %s, Detail: %s", e.FaultCode, e.FaultString, e.Detail)
}

// OpcResponseError represents an OPC-XML-DA level error reported in a response's
// <Errors> element (as opposed to a SOAP fault). Use errors.As to detect it in an
// error returned by a Server method.
type OpcResponseError struct {
	Id   string
	Text []string
	Type string
}

func (e *OpcResponseError) Error() string {
	return fmt.Sprintf("Id: %s, Text: %s, Type: %s", e.Id, e.Text, e.Type)
}

// InvalidServerSubHandlesError is returned by SubscriptionPolledRefresh when the
// server reports one or more subscription handles it no longer recognizes. Use
// errors.As to detect it in an error returned by SubscriptionPolledRefresh.
type InvalidServerSubHandlesError struct {
	Handles []string
}

func (e *InvalidServerSubHandlesError) Error() string {
	return fmt.Sprintf("InvalidServerSubHandles: %v", e.Handles)
}
