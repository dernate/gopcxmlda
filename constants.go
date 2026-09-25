package gopcxmlda

// DefaultContentType is the HTTP Content-Type sent with every request unless
// Server.ContentType overrides it.
//
// OPC XML-DA 1.0 is built on SOAP 1.1: the specification's WSDL uses the SOAP 1.1
// binding (http://schemas.xmlsoap.org/wsdl/soap/), per-operation soapAction values,
// and the SOAP 1.1 envelope namespace, and SOAP 1.2 is not mentioned anywhere. SOAP
// 1.1 requires the media type text/xml for HTTP (SOAP 1.1, section 6.1.1). The
// charset parameter matches the encoding declared in every payload's XML
// declaration.
//
// application/soap+xml, which earlier versions of this package sent, is the SOAP 1.2
// media type. Lenient servers ignore the mismatch with the SOAP 1.1 envelope, strict
// ones (e.g. .NET ASMX/WCF basicHttpBinding endpoints) reject it with
// 415 Unsupported Media Type.
const DefaultContentType = "text/xml; charset=utf-8"

// soapActions maps each OPC XML-DA operation to its soapAction URI as defined in the
// specification's WSDL, sent as the SOAP 1.1 SOAPAction HTTP header. It is
// intentionally unexported: being a mutable package-level map, exporting it would let
// any importer corrupt shared global state for every user of the package.
var soapActions = map[string]string{
	"GetStatus":                 "http://opcfoundation.org/webservices/XMLDA/1.0/GetStatus",
	"GetProperties":             "http://opcfoundation.org/webservices/XMLDA/1.0/GetProperties",
	"Read":                      "http://opcfoundation.org/webservices/XMLDA/1.0/Read",
	"Write":                     "http://opcfoundation.org/webservices/XMLDA/1.0/Write",
	"Browse":                    "http://opcfoundation.org/webservices/XMLDA/1.0/Browse",
	"Subscribe":                 "http://opcfoundation.org/webservices/XMLDA/1.0/Subscribe",
	"SubscriptionPolledRefresh": "http://opcfoundation.org/webservices/XMLDA/1.0/SubscriptionPolledRefresh",
	"SubscriptionCancel":        "http://opcfoundation.org/webservices/XMLDA/1.0/SubscriptionCancel",
}

const XmlVersion = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>"

const EnvelopeOpen1 = "<SOAP-ENV:Envelope " +
	"xmlns:SOAP-ENV=\"http://schemas.xmlsoap.org/soap/envelope/\" " +
	"xmlns:SOAP-ENC=\"http://schemas.xmlsoap.org/soap/encoding/\" " +
	"xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" " +
	"xmlns:xsd=\"http://www.w3.org/2001/XMLSchema\" " +
	"xmlns:"

// namespace in between ENVELOPE_OPEN_1 and ENVELOPE_OPEN_2 (ENVELOPE_OPEN_1 + namespace + ENVELOPE_OPEN_2)

const EnvelopeOpen2 = "=\"http://opcfoundation.org/webservices/XMLDA/1.0/\">"

const EnvelopeHeader = "<SOAP-ENV:Header></SOAP-ENV:Header>"
const EnvelopeBodyOpenNs1 = "<SOAP-ENV:Body xmlns:"

const EnvelopeHeaderToBody = EnvelopeOpen2 + EnvelopeHeader + EnvelopeBodyOpenNs1

const EnvelopeBodyOpenNs2 = "=\"http://opcfoundation.org/webservices/XMLDA/1.0/\">"

// PAYLOAD GOES HERE

const EnvelopeBodyClose = "</SOAP-ENV:Body>"
const EnvelopeClose = "</SOAP-ENV:Envelope>"
const Footer = EnvelopeBodyClose + EnvelopeClose
