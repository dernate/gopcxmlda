package gopcxmlda

import (
	"encoding/xml"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// UnmarshalXML Helper function to unmarshal XML into a TValue struct.
// The tipping point for this function is the switch statement that handles
// either single or array values.
// Array values are handled by the decodeArrayOf function, whereas single values
// are handled by the switch statement that handles the different types.
func (v *TValue) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	split := strings.Split(start.Attr[0].Value, ":")
	v.Namespace = split[0]
	v.Type = split[1]
	switch v.Type {
	case "string", "base64Binary":
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
		var t xml.Token
		var err error
		if t, err = d.Token(); err != nil {
			break
		}

		switch se := t.(type) {
		case xml.StartElement:
			var value interface{}

			switch v.Type {
			case "ArrayOfString":
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
	return fmt.Errorf("error in decoding ArrayOf* types")
}

func valueIsArrayOrSlice(value interface{}) bool {
	valueType := reflect.TypeOf(value)

	if valueType.Kind() == reflect.Slice || valueType.Kind() == reflect.Array {
		return true
	} else {
		return false
	}
}

func setOpcXmlDaTypes(items []TItem) []TItem {
	for i := range items {
		if items[i].Value.Type == "" {
			// Only set the type if it is not already set
			item, err := getOpcXmlDaType(items[i].Value.Value)
			if err != nil {
				logError(err, "setOpcXmlDaTypes")
			} else {
				items[i].Value.Type = item
			}
		}
	}
	return items
}

func getOpcXmlDaType(value interface{}) (string, error) {
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
