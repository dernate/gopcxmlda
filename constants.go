package gopcxmlda

var HEADERS = map[string]string{
	"content-type": "text/xml",
}

var HEADERS_SOAP = map[string]string{
	"content-type": "application/soap+xml",
}

const XML_VERSION = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>"

const ENVELOPE_OPEN_1 = "<SOAP-ENV:Envelope " +
	"xmlns:SOAP-ENV=\"http://schemas.xmlsoap.org/soap/envelope/\" " +
	"xmlns:SOAP-ENC=\"http://schemas.xmlsoap.org/soap/encoding/\" " +
	"xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" " +
	"xmlns:xsd=\"http://www.w3.org/2001/XMLSchema\" " +
	"xmlns:"

// namespace in between ENVELOPE_OPEN_1 and ENVELOPE_OPEN_2 (ENVELOPE_OPEN_1 + namespace + ENVELOPE_OPEN_2)

const ENVELOPE_OPEN_2 = "=\"http://opcfoundation.org/webservices/XMLDA/1.0/\">"

const ENVELOPE_HEADER = "<SOAP-ENV:Header></SOAP-ENV:Header>"
const ENVELOPE_BODY_OPEN = "<SOAP-ENV:Body>"
const ENVELOPE_BODY_OPEN_NS_1 = "<SOAP-ENV:Body xmlns:"
const ENVELOPE_BODY_OPEN_NS_2 = "=\"http://opcfoundation.org/webservices/XMLDA/1.0/\">"

// PAYLOAD GOES HERE

const ENVELOPE_BODY_CLOSE = "</SOAP-ENV:Body>"
const ENVELOPE_CLOSE = "</SOAP-ENV:Envelope>"
