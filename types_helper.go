package gopcxmlda

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// xsiNamespace is the standard XML Schema Instance namespace URI that xsi:type is
// defined in.
const xsiNamespace = "http://www.w3.org/2001/XMLSchema-instance"

// findTypeAttr locates the xsi:type attribute among attrs by name, rather than
// assuming it is always the first attribute on the element. Attribute order is not
// semantically significant in XML, so relying on positional order (e.g. attrs[0])
// breaks as soon as a server emits any other attribute (including a locally-scoped
// xmlns declaration) before xsi:type. Prefers an attribute explicitly in the xsi
// namespace, falling back to any attribute simply named "type" for servers that
// don't resolve the namespace as expected.
func findTypeAttr(attrs []xml.Attr) *xml.Attr {
	var fallback *xml.Attr
	for i := range attrs {
		if attrs[i].Name.Local != "type" {
			continue
		}
		if attrs[i].Name.Space == xsiNamespace {
			return &attrs[i]
		}
		if fallback == nil {
			fallback = &attrs[i]
		}
	}
	return fallback
}

// UnmarshalXML Helper function to unmarshal XML into a TValue struct.
// The tipping point for this function is the switch statement that handles
// either single or array values.
// Array values are handled by the decodeArrayOf function, whereas single values
// are handled by the switch statement that handles the different types.
func (v *TValue) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typeAttr := findTypeAttr(start.Attr)
	if typeAttr == nil {
		return fmt.Errorf("gopcxmlda: missing xsi:type attribute on <%s> element", start.Name.Local)
	}
	// xsi:type is a QName. It is usually prefixed ("xsd:int", "ns1:ArrayOfDouble"), but
	// may also be unprefixed when the type is in the default namespace - the form the
	// specification's own examples use (xsi:type="ArrayOfInt"). Namespace then stays
	// empty.
	prefix, local, hasPrefix := strings.Cut(typeAttr.Value, ":")
	if !hasPrefix {
		prefix, local = "", typeAttr.Value
	}
	if local == "" || strings.Contains(local, ":") {
		return fmt.Errorf("gopcxmlda: unexpected xsi:type attribute %q on <%s> element", typeAttr.Value, start.Name.Local)
	}
	v.Namespace = prefix
	v.Type = local
	switch v.Type {
	case "string", "QName":
		var data string
		if err := d.DecodeElement(&data, &start); err != nil {
			return err
		}
		v.Value = data
	case "base64Binary":
		var data string
		if err := d.DecodeElement(&data, &start); err != nil {
			return err
		}
		decoded, err := decodeBase64Binary(data)
		if err != nil {
			return err
		}
		v.Value = decoded
	case "duration":
		// Kept in its lexical form ("P1DT2H"): time.Duration can't represent years and
		// months exactly, and a string is what the specification maps duration to
		// (VT_BSTR) - and how it recommends transmitting it in the first place.
		var data string
		if err := d.DecodeElement(&data, &start); err != nil {
			return err
		}
		v.Value = strings.TrimSpace(data)
	case "boolean":
		var data bool
		if err := d.DecodeElement(&data, &start); err != nil {
			return err
		}
		v.Value = data
	case "dateTime":
		var data xsdDateTime
		if err := d.DecodeElement(&data, &start); err != nil {
			return err
		}
		v.Value = time.Time(data)
	case "date", "time":
		var data string
		if err := d.DecodeElement(&data, &start); err != nil {
			return err
		}
		parsed, err := parseXsdDateOrTime(v.Type, data)
		if err != nil {
			return err
		}
		v.Value = parsed
	case "int":
		var data int
		if err := d.DecodeElement(&data, &start); err != nil {
			return err
		}
		v.Value = data
	case "long":
		var data int64
		if err := d.DecodeElement(&data, &start); err != nil {
			return err
		}
		v.Value = data
	case "unsignedLong", "unsignedInt":
		var data uint64
		if err := d.DecodeElement(&data, &start); err != nil {
			return err
		}
		v.Value = data
	case "short", "byte":
		var data int16
		if err := d.DecodeElement(&data, &start); err != nil {
			return err
		}
		v.Value = data
	case "unsignedShort", "unsignedByte":
		var data uint16
		if err := d.DecodeElement(&data, &start); err != nil {
			return err
		}
		v.Value = data
	case "float":
		var data float32
		if err := d.DecodeElement(&data, &start); err != nil {
			return err
		}
		v.Value = data
	case "double", "decimal":
		var data float64
		if err := d.DecodeElement(&data, &start); err != nil {
			return err
		}
		v.Value = data
	case "OPCQuality": // used at GetProperties
		var data TQuality
		if err := d.DecodeElement(&data, &start); err != nil {
			return err
		}
		v.Value = data
	default:
		switch v.Type {
		case "ArrayOfString", "ArrayOfBoolean", "ArrayOfDateTime", "ArrayOfLong", "ArrayOfInt", "ArrayOfUnsignedLong", "ArrayOfUnsignedInt", "ArrayOfShort", "ArrayOfByte", "ArrayOfUnsignedShort", "ArrayOfUnsignedByte", "ArrayOfFloat", "ArrayOfDouble", "ArrayOfDecimal":
			return v.decodeArrayOf(d, &start)
		case "ArrayOfAnyType":
			return v.decodeArrayOfAnyType(d, &start)
		default:
			return fmt.Errorf("unknown type: %s", v.Type)
		}
	}
	return nil
}

// Helper function to decode array values into a TValue struct.
func (v *TValue) decodeArrayOf(d *xml.Decoder, start *xml.StartElement) error {
	var tempSlice []interface{}
	for {
		t, err := d.Token()
		if err != nil {
			return fmt.Errorf("gopcxmlda: error decoding %s: %w", v.Type, err)
		}

		switch se := t.(type) {
		case xml.StartElement:
			var value interface{}

			switch v.Type {
			case "ArrayOfString", "ArrayOfQName":
				var s string
				err := d.DecodeElement(&s, &se)
				if err != nil {
					return err
				}
				value = s
			case "ArrayOfBoolean":
				var b bool
				err := d.DecodeElement(&b, &se)
				if err != nil {
					return err
				}
				value = b
			case "ArrayOfDateTime":
				var t xsdDateTime
				err := d.DecodeElement(&t, &se)
				if err != nil {
					return err
				}
				value = time.Time(t)
			case "ArrayOfLong":
				var l int64
				err := d.DecodeElement(&l, &se)
				if err != nil {
					return err
				}
				value = l
			case "ArrayOfInt":
				var i int
				err := d.DecodeElement(&i, &se)
				if err != nil {
					return err
				}
				value = i
			case "ArrayOfUnsignedLong":
				var l uint64
				err := d.DecodeElement(&l, &se)
				if err != nil {
					return err
				}
				value = l
			case "ArrayOfUnsignedInt":
				var i uint
				err := d.DecodeElement(&i, &se)
				if err != nil {
					return err
				}
				value = i
			case "ArrayOfShort":
				var s int16
				err := d.DecodeElement(&s, &se)
				if err != nil {
					return err
				}
				value = s
			case "ArrayOfByte":
				var b int8
				err := d.DecodeElement(&b, &se)
				if err != nil {
					return err
				}
				value = b
			case "ArrayOfUnsignedShort":
				var s uint16
				err := d.DecodeElement(&s, &se)
				if err != nil {
					return err
				}
				value = s
			case "ArrayOfUnsignedByte":
				var b uint8
				err := d.DecodeElement(&b, &se)
				if err != nil {
					return err
				}
				value = b
			case "ArrayOfFloat":
				var f float32
				err := d.DecodeElement(&f, &se)
				if err != nil {
					return err
				}
				value = f
			case "ArrayOfDouble":
				var db float64
				err := d.DecodeElement(&db, &se)
				if err != nil {
					return err
				}
				value = db
			case "ArrayOfDecimal":
				var dc float64
				err := d.DecodeElement(&dc, &se)
				if err != nil {
					return err
				}
				value = dc
			default:
				return fmt.Errorf("unknown type: %s", v.Type)
			}
			tempSlice = append(tempSlice, value)
		case xml.EndElement:
			if se == start.End() {
				v.Value = tempSlice
				return nil
			}
		}
	}
}

// decodeArrayOfAnyType decodes an ArrayOfAnyType, whose elements each carry their own
// xsi:type and may be of different simple types or arrays themselves, e.g.
//
//	<Value xsi:type="ArrayOfAnyType">
//	  <anyType xsi:type="xsd:byte">127</anyType>
//	  <anyType xsi:type="ArrayOfInt"><int>1</int></anyType>
//	</Value>
//
// Each element is decoded like a <Value> of its own, so Value becomes a []interface{}
// holding the elements' decoded values - a nested array as a nested []interface{}.
func (v *TValue) decodeArrayOfAnyType(d *xml.Decoder, start *xml.StartElement) error {
	var values []interface{}
	for {
		t, err := d.Token()
		if err != nil {
			return fmt.Errorf("gopcxmlda: error decoding %s: %w", v.Type, err)
		}
		switch se := t.(type) {
		case xml.StartElement:
			var elem TValue
			if err := elem.UnmarshalXML(d, se); err != nil {
				return err
			}
			values = append(values, elem.Value)
		case xml.EndElement:
			if se == start.End() {
				v.Value = values
				return nil
			}
		}
	}
}

// xsdDateTime decodes an xsd:dateTime. Unlike time.Time's own text decoding (strict RFC
// 3339), it also accepts the timezone-less form XML Schema allows
// ("2026-09-25T15:38:06.5"), which is interpreted as UTC.
type xsdDateTime time.Time

func (t *xsdDateTime) UnmarshalText(text []byte) error {
	parsed, err := parseXsdDateTime(string(text))
	if err != nil {
		return err
	}
	*t = xsdDateTime(parsed)
	return nil
}

func parseXsdDateTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed, nil
	}
	// Without a zone designator, time.Parse returns the time in UTC.
	parsed, err := time.Parse("2006-01-02T15:04:05.999999999", value)
	if err != nil {
		return time.Time{}, fmt.Errorf("gopcxmlda: %q is not a valid xsd:dateTime", value)
	}
	return parsed, nil
}

// parseXsdDateOrTime decodes an xsd:date ("2026-09-25", midnight) or an xsd:time
// ("15:38:06.5", on January 1st of year 0) into a time.Time, the way the specification
// maps both to VT_DATE. The zone designator is optional; without one, UTC applies (as
// for xsd:dateTime).
func parseXsdDateOrTime(typ, value string) (time.Time, error) {
	layout := "2006-01-02"
	if typ == "time" {
		layout = "15:04:05.999999999"
	}
	value = strings.TrimSpace(value)
	if parsed, err := time.Parse(layout+"Z07:00", value); err == nil {
		return parsed, nil
	}
	parsed, err := time.Parse(layout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("gopcxmlda: %q is not a valid xsd:%s", value, typ)
	}
	return parsed, nil
}

// decodeBase64Binary decodes an xsd:base64Binary. XML Schema allows whitespace (e.g.
// line breaks in long values) inside the encoding, so it is removed first.
func decodeBase64Binary(value string) ([]byte, error) {
	compact := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n', '\r':
			return -1
		}
		return r
	}, value)
	decoded, err := base64.StdEncoding.DecodeString(compact)
	if err != nil {
		return nil, fmt.Errorf("gopcxmlda: invalid xsd:base64Binary value: %w", err)
	}
	return decoded, nil
}

// UnmarshalXML decodes a TItem, reading its Timestamp attribute as an xsd:dateTime
// (see xsdDateTime) instead of as strict RFC 3339. All other fields decode as declared
// on TItem.
func (i *TItem) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type plainItem TItem // same fields, without this method - avoids infinite recursion
	aux := struct {
		*plainItem
		Timestamp xsdDateTime `xml:"Timestamp,attr"` // shadows plainItem.Timestamp
	}{plainItem: (*plainItem)(i)}
	if err := d.DecodeElement(&aux, &start); err != nil {
		return err
	}
	i.Timestamp = time.Time(aux.Timestamp)
	return nil
}

// UnmarshalXML decodes a TBaseResult, reading ReplyTime and RcvTime as xsd:dateTime
// (see xsdDateTime) instead of as strict RFC 3339.
func (r *TBaseResult) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type plainResult TBaseResult
	aux := struct {
		*plainResult
		ReplyTime   xsdDateTime `xml:"ReplyTime,attr"`
		ReceiveTime xsdDateTime `xml:"RcvTime,attr"`
	}{plainResult: (*plainResult)(r)}
	if err := d.DecodeElement(&aux, &start); err != nil {
		return err
	}
	r.ReplyTime = time.Time(aux.ReplyTime)
	r.ReceiveTime = time.Time(aux.ReceiveTime)
	return nil
}

func valueIsArrayOrSlice(value interface{}) bool {
	if value == nil {
		return false
	}
	valueType := reflect.TypeOf(value)

	if valueType.Kind() == reflect.Slice || valueType.Kind() == reflect.Array {
		return true
	} else {
		return false
	}
}

// setOpcXmlDaTypes infers and fills in the OPC-XML-DA wire type (Value.Type) for
// every item whose Value.Type isn't already set. It returns an error - rather than
// silently leaving Value.Type empty - for any item whose Value.Value is nil or of an
// unsupported Go type, since building a Write/Subscribe payload from such an item
// would otherwise fail deep inside XML marshalling with a much less useful error (or,
// prior to this check, panic outright on a nil Value.Value).
func setOpcXmlDaTypes(items []TItem) ([]TItem, error) {
	for i := range items {
		if items[i].Value.Type == "" {
			// Only set the type if it is not already set
			item, err := getOpcXmlDaType(items[i].Value.Value)
			if err != nil {
				return nil, fmt.Errorf("gopcxmlda: item %d (%q): %w", i, items[i].ItemName, err)
			}
			items[i].Value.Type = item
		}
	}
	return items, nil
}

func getOpcXmlDaType(value interface{}) (string, error) {
	if value == nil {
		return "", fmt.Errorf("Value.Value must not be nil - set it before calling Write/Subscribe, or set Value.Type explicitly")
	}
	// A []byte is sent as a single base64Binary value: the specification explicitly
	// excludes ArrayOfUnsignedByte, as base64 is the more efficient encoding for bytes.
	if _, ok := value.([]byte); ok {
		return "base64Binary", nil
	}
	var arrayType bool
	var elemType reflect.Type
	vo := reflect.ValueOf(value)
	if vo.Kind() == reflect.Slice && vo.Len() > 0 {
		elemType = vo.Index(0).Type()
		arrayType = true
	} else if vo.Kind() == reflect.Slice {
		elemType = vo.Type().Elem()
		arrayType = true
	} else {
		elemType = vo.Type()
		arrayType = false
	}
	switch elemType {
	case reflect.TypeOf(true):
		if arrayType {
			return "ArrayOfBoolean", nil
		} else {
			return "boolean", nil
		}
	case reflect.TypeOf(""):
		if arrayType {
			return "ArrayOfString", nil
		} else {
			return "string", nil
		}
	case reflect.TypeOf(float32(0.0)):
		if arrayType {
			return "ArrayOfFloat", nil
		} else {
			return "float", nil
		}
	case reflect.TypeOf(float64(0.0)):
		if arrayType {
			return "ArrayOfDouble", nil
		} else {
			return "double", nil
		}
	case reflect.TypeOf(time.Time{}):
		if arrayType {
			return "ArrayOfDateTime", nil
		} else {
			return "dateTime", nil
		}
	case reflect.TypeOf(int8(0)):
		if arrayType {
			return "ArrayOfByte", nil
		} else {
			return "byte", nil
		}
	case reflect.TypeOf(uint8(0)):
		if arrayType {
			return "ArrayOfUnsignedByte", nil
		} else {
			return "unsignedByte", nil
		}
	case reflect.TypeOf(int16(0)):
		if arrayType {
			return "ArrayOfShort", nil
		} else {
			return "short", nil
		}
	case reflect.TypeOf(uint16(0)):
		if arrayType {
			return "ArrayOfUnsignedShort", nil
		} else {
			return "unsignedShort", nil
		}
	case reflect.TypeOf(int64(0)):
		if arrayType {
			return "ArrayOfLong", nil
		} else {
			return "long", nil
		}
	case reflect.TypeOf(uint64(0)):
		if arrayType {
			return "ArrayOfUnsignedLong", nil
		} else {
			return "unsignedLong", nil
		}
	case reflect.TypeOf(int(0)), reflect.TypeOf(int32(0)):
		if arrayType {
			return "ArrayOfInt", nil
		} else {
			return "int", nil
		}
	case reflect.TypeOf(uint(0)), reflect.TypeOf(uint32(0)):
		if arrayType {
			return "ArrayOfUnsignedInt", nil
		} else {
			return "unsignedInt", nil
		}
	default:
		return "", fmt.Errorf("unknown type: %v", reflect.TypeOf(value))
	}
}
