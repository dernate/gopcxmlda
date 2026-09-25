package gopcxmlda

import (
	"fmt"
	"strings"
)

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
// error returned by a Server method. The response itself is still returned alongside
// it.
type OpcResponseError struct {
	Id   string   // ID of the first <Errors> element
	Text []string // texts of all <Errors> elements, in order
	Type string

	// Errors holds every <Errors> element with its own ID and text.
	Errors []OpcError
	// Items holds every item of the response that carries a ResultID, each paired
	// with its text - i.e. which items failed, and why.
	Items []ItemResult
}

// Error keeps the "Id: ..., Text: ..., Type: ..." form of earlier versions and appends
// the individual errors (when there is more than one) and the affected items.
func (e *OpcResponseError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Id: %s, Text: %s, Type: %s", e.Id, e.Text, e.Type)
	if len(e.Errors) > 1 {
		b.WriteString("; errors:")
		for i, entry := range e.Errors {
			if i > 0 {
				b.WriteString(",")
			}
			fmt.Fprintf(&b, " %s (%s)", entry.ID, entry.Text)
		}
	}
	if len(e.Items) > 0 {
		b.WriteString("; items:")
		for i, item := range e.Items {
			if i > 0 {
				b.WriteString(",")
			}
			fmt.Fprintf(&b, " %s%s: %s", item.ItemPath, item.ItemName, item.ResultID)
			if item.Text != "" {
				fmt.Fprintf(&b, " (%s)", item.Text)
			}
		}
	}
	return b.String()
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
