package gopcxmlda

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultTimeout is used for a Server whose Timeout field is left at its zero value.
const DefaultTimeout = 10 * time.Second

// validate checks that the Server is usable before a request is built or sent.
func (s *Server) validate() error {
	if s == nil {
		return errors.New("gopcxmlda: Server must not be nil")
	}
	if s.Url == nil {
		return errors.New("gopcxmlda: Server.Url must not be nil")
	}
	return nil
}

func buildHeader(builder *strings.Builder, namespace string) string {
	builder.WriteString(EnvelopeOpen1)
	builder.WriteString(namespace)
	builder.WriteString(EnvelopeHeaderToBody)
	builder.WriteString(namespace)
	builder.WriteString(EnvelopeBodyOpenNs2)

	return builder.String()
}

// send sends a payload to the server and returns the byte response and an error if any.
func send(ctx context.Context, s *Server, payload string, SOAPAction string) ([]byte, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if s.Timeout <= 0 {
		s.Timeout = DefaultTimeout
	}
	if _, ok := HeadersSoap[fmt.Sprintf("SOAPAction-%s", SOAPAction)]; !ok {
		return nil, fmt.Errorf("unknown SOAPAction: %s", SOAPAction)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.Url.String(), bytes.NewBufferString(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", HeadersSoap["content-type"])
	req.Header.Set("SOAPAction", HeadersSoap[fmt.Sprintf("SOAPAction-%s", SOAPAction)])

	// The HTTP client is created once and reused across requests so that connections
	// (keep-alive) are pooled instead of being re-established on every call.
	if s.Client == nil {
		s.Client = &http.Client{Timeout: s.Timeout}
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		if cerr := Body.Close(); cerr != nil {
			logError(cerr, "send")
		}
	}(resp.Body)

	var errReturn error
	if resp.StatusCode != http.StatusOK {
		errReturn = errors.Join(errReturn, fmt.Errorf("unexpected response status: %s", resp.Status))
	}
	respbody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Join(errReturn, err)
	}
	return respbody, errReturn
}

// doRequest sends payload for action, unmarshals the response into a value of type T,
// and aggregates SOAP faults and OPC-XML-DA errors into the returned error. It captures
// the request/response/error-handling flow shared by every public Server method.
func doRequest[T soapResponse](ctx context.Context, s *Server, payload string, action string) (T, error) {
	var result T

	response, sendErr := send(ctx, s, payload, action)
	var errReturn error
	if sendErr != nil {
		errReturn = errors.Join(errReturn, sendErr)
	}

	if err := xml.Unmarshal(response, &result); err != nil {
		errReturn = errors.Join(errReturn, err)
		logError(errReturn, action)
		var zero T
		return zero, errReturn
	}

	if f := result.fault(); f.FaultCode != "" {
		errReturn = errors.Join(errReturn, fmt.Errorf(
			"Faultcode: %s, Faultstring: %s, Detail: %s", f.FaultCode, f.FaultString, f.Detail,
		))
	}
	if e := result.responseErrors(); e.Id != "" {
		errReturn = errors.Join(errReturn, fmt.Errorf(
			"Id: %s, Text: %s, Type: %s", e.Id, e.Text, e.Type,
		))
	}

	if errReturn != nil {
		logError(errReturn, action)
	}
	return result, errReturn
}
