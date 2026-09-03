package gopcxmlda

import (
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
	split := strings.Split(typeAttr.Value, ":")
	if len(split) < 2 {
		return fmt.Errorf("gopcxmlda: unexpected xsi:type attribute %q on <%s> element", typeAttr.Value, start.Name.Local)
	}
	v.Namespace = split[0]
	v.Type = split[1]
	switch v.Type {
	case "string", "base64Binary", "QName":
		var data string
		if err := d.DecodeElement(&data, &start); err != nil {
			return err
		}
		v.Value = data
	case "boolean":
		var data bool
		if err := d.DecodeElement(&data, &start); err != nil {
			return err
		}
		v.Value = data
	case "dateTime", "time", "date", "duration":
		var data time.Time
		if err := d.DecodeElement(&data, &start); err != nil {
			return err
		}
		v.Value = data
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
				var t time.Time
				err := d.DecodeElement(&t, &se)
				if err != nil {
					return err
				}
				value = t
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
