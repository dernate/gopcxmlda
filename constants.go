package gopcxmlda

var HeadersSoap = map[string]string{
	"content-type": "application/soap+xml",
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
