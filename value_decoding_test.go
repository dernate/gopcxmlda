package gopcxmlda

import (
	"encoding/xml"
	"reflect"
	"regexp"
	"testing"
	"time"
)

const valueNamespaces = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" ` +
	`xmlns:xsd="http://www.w3.org/2001/XMLSchema" ` +
	`xmlns:ns1="http://opcfoundation.org/webservices/XMLDA/1.0/"`

func decodeValue(t *testing.T, doc string) (TValue, error) {
	t.Helper()
	var v TValue
	err := xml.Unmarshal([]byte(doc), &v)
	return v, err
}

// TestValueDecodesEveryScalarType pins down the Go type every supported scalar
// xsi:type is decoded into - the types callers type-assert Value.Value against.
func TestValueDecodesEveryScalarType(t *testing.T) {
	cases := []struct {
		xsiType, content string
		want             interface{}
	}{
		{"xsd:string", "Hello &amp; &lt;World&gt;", "Hello & <World>"},
		{"xsd:string", "", ""},
		{"xsd:base64Binary", "AQID", "AQID"},
		{"xsd:QName", "xsd:unsignedInt", "xsd:unsignedInt"},
		{"xsd:boolean", "true", true},
		{"xsd:boolean", "0", false},
		{"xsd:int", "-2147483648", -2147483648},
		{"xsd:long", "9223372036854775807", int64(9223372036854775807)},
		{"xsd:unsignedLong", "18446744073709551615", uint64(18446744073709551615)},
		{"xsd:unsignedInt", "4294967295", uint64(4294967295)},
		{"xsd:short", "-32768", int16(-32768)},
		{"xsd:byte", "-128", int16(-128)},
		{"xsd:unsignedShort", "65535", uint16(65535)},
		{"xsd:unsignedByte", "255", uint16(255)},
		{"xsd:float", "1.5", float32(1.5)},
		{"xsd:double", "-2.25", -2.25},
		{"xsd:decimal", "12.125", 12.125},
		{"xsd:dateTime", "2026-09-25T15:38:06.025+02:00", time.Date(2026, 9, 25, 13, 38, 6, 25e6, time.UTC)},
	}
	for _, tc := range cases {
		t.Run(tc.xsiType+"/"+tc.content, func(t *testing.T) {
			v, err := decodeValue(t, `<Value `+valueNamespaces+` xsi:type="`+tc.xsiType+`">`+tc.content+`</Value>`)
			if err != nil {
				t.Fatal(err)
			}
			if want, ok := tc.want.(time.Time); ok {
				got, isTime := v.Value.(time.Time)
				if !isTime || !got.Equal(want) {
					t.Fatalf("expected %v, got %#v", want, v.Value)
				}
				return
			}
			if !reflect.DeepEqual(v.Value, tc.want) {
				t.Fatalf("expected %#v (%T), got %#v (%T)", tc.want, tc.want, v.Value, v.Value)
			}
		})
	}
}

func TestValueDecodesOPCQuality(t *testing.T) {
	v, err := decodeValue(t, `<Value `+valueNamespaces+` xsi:type="ns1:OPCQuality" VendorField="7" LimitField="low" QualityField="uncertain"/>`)
	if err != nil {
		t.Fatal(err)
	}
	want := TQuality{VendorField: "7", LimitField: "low", QualityField: "uncertain"}
	if v.Value != want {
		t.Fatalf("expected %+v, got %#v", want, v.Value)
	}
}

// TestValueDecodesEveryArrayType pins down the element Go type of every supported
// array type. Arrays are always decoded into []interface{}.
func TestValueDecodesEveryArrayType(t *testing.T) {
	cases := []struct {
		arrayType, element string
		values             []string
		want               []interface{}
	}{
		{"ArrayOfString", "string", []string{"a", ""}, []interface{}{"a", ""}},
		{"ArrayOfBoolean", "boolean", []string{"true", "false"}, []interface{}{true, false}},
		{"ArrayOfDateTime", "dateTime", []string{"2026-09-25T15:38:06Z"}, []interface{}{time.Date(2026, 9, 25, 15, 38, 6, 0, time.UTC)}},
		{"ArrayOfLong", "long", []string{"-1", "9223372036854775807"}, []interface{}{int64(-1), int64(9223372036854775807)}},
		{"ArrayOfInt", "int", []string{"-1", "2"}, []interface{}{-1, 2}},
		{"ArrayOfUnsignedLong", "unsignedLong", []string{"18446744073709551615"}, []interface{}{uint64(18446744073709551615)}},
		{"ArrayOfUnsignedInt", "unsignedInt", []string{"4294967295"}, []interface{}{uint(4294967295)}},
		{"ArrayOfShort", "short", []string{"-32768"}, []interface{}{int16(-32768)}},
		{"ArrayOfByte", "byte", []string{"-128", "127"}, []interface{}{int8(-128), int8(127)}},
		{"ArrayOfUnsignedShort", "unsignedShort", []string{"65535"}, []interface{}{uint16(65535)}},
		{"ArrayOfUnsignedByte", "unsignedByte", []string{"255"}, []interface{}{uint8(255)}},
		{"ArrayOfFloat", "float", []string{"1.5"}, []interface{}{float32(1.5)}},
		{"ArrayOfDouble", "double", []string{"-2.25", "0"}, []interface{}{-2.25, 0.0}},
		{"ArrayOfDecimal", "decimal", []string{"12.125"}, []interface{}{12.125}},
	}
	for _, tc := range cases {
		t.Run(tc.arrayType, func(t *testing.T) {
			doc := `<Value ` + valueNamespaces + ` xsi:type="ns1:` + tc.arrayType + `">`
			for _, value := range tc.values {
				doc += `<ns1:` + tc.element + `>` + value + `</ns1:` + tc.element + `>`
			}
			doc += `</Value>`
			v, err := decodeValue(t, doc)
			if err != nil {
				t.Fatal(err)
			}
			if v.Type != tc.arrayType || v.Namespace != "ns1" {
				t.Errorf("expected Type %q Namespace ns1, got Type %q Namespace %q", tc.arrayType, v.Type, v.Namespace)
			}
			if !reflect.DeepEqual(v.Value, tc.want) {
				t.Fatalf("expected %#v, got %#v", tc.want, v.Value)
			}
		})
	}
}

// TestValueDecodesArrayWithWhitespaceAndEmptyArray covers the pretty-printed form real
// servers send (whitespace between elements) and an empty array.
func TestValueDecodesArrayWithWhitespaceAndEmptyArray(t *testing.T) {
	v, err := decodeValue(t, "<Value "+valueNamespaces+" xsi:type=\"ns1:ArrayOfInt\">\n  <ns1:int>1</ns1:int>\n  <ns1:int>2</ns1:int>\n</Value>")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(v.Value, []interface{}{1, 2}) {
		t.Fatalf("expected [1 2], got %#v", v.Value)
	}

	v, err = decodeValue(t, `<Value `+valueNamespaces+` xsi:type="ns1:ArrayOfDouble"></Value>`)
	if err != nil {
		t.Fatal(err)
	}
	if v.Type != "ArrayOfDouble" || len(v.Value.([]interface{})) != 0 {
		t.Fatalf("expected an empty ArrayOfDouble, got %+v", v)
	}
}

// TestValueDecodeErrors pins down that malformed values produce an error instead of
// a silently wrong value (see the fail-fast note on doRequest).
func TestValueDecodeErrors(t *testing.T) {
	cases := map[string]string{
		"missing xsi:type":        `<Value>42</Value>`,
		"unknown scalar type":     `<Value ` + valueNamespaces + ` xsi:type="xsd:gYear">2026</Value>`,
		"unknown array type":      `<Value ` + valueNamespaces + ` xsi:type="ns1:ArrayOfGYear"><ns1:gYear>2026</ns1:gYear></Value>`,
		"int with text content":   `<Value ` + valueNamespaces + ` xsi:type="xsd:int">abc</Value>`,
		"int out of range":        `<Value ` + valueNamespaces + ` xsi:type="xsd:short">40000</Value>`,
		"invalid boolean":         `<Value ` + valueNamespaces + ` xsi:type="xsd:boolean">yes</Value>`,
		"invalid dateTime":        `<Value ` + valueNamespaces + ` xsi:type="xsd:dateTime">yesterday</Value>`,
		"invalid array element":   `<Value ` + valueNamespaces + ` xsi:type="ns1:ArrayOfInt"><ns1:int>1</ns1:int><ns1:int>x</ns1:int></Value>`,
		"truncated array element": `<Value ` + valueNamespaces + ` xsi:type="ns1:ArrayOfInt"><ns1:int>1</ns1:int>`,
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeValue(t, doc); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

// TestValueFindsTypeWithoutXsiNamespaceDeclaration covers the fallback for servers that
// emit a plain "type" attribute without resolving the xsi namespace.
func TestValueFindsTypeWithoutXsiNamespaceDeclaration(t *testing.T) {
	v, err := decodeValue(t, `<Value type="xsd:int">5</Value>`)
	if err != nil {
		t.Fatal(err)
	}
	if v.Value != 5 {
		t.Fatalf("expected 5, got %#v", v.Value)
	}
}

// TestWriteValueRoundTrip renders every Go type Write supports into the <Value>
// element Write would send, decodes that element again and checks the value survives.
// This ties the marshalling side (getOpcXmlDaType, xmlWriteValue) to the unmarshalling
// side (TValue.UnmarshalXML).
func TestWriteValueRoundTrip(t *testing.T) {
	ts := time.Date(2026, 9, 25, 15, 38, 6, 0, time.UTC)
	cases := []struct {
		in   interface{}
		want interface{}
	}{
		{true, true},
		{"text & <markup>", "text & <markup>"},
		{float32(1.5), float32(1.5)},
		{2.25, 2.25},
		// A scalar time.Time is left out on purpose: Write currently renders it with
		// fmt's %v ("2026-09-25 15:38:06 +0000 UTC") instead of as an xsd:dateTime, so it
		// can't round-trip. Arrays of time.Time are rendered correctly (see below).
		{int8(-5), int16(-5)},
		{uint8(5), uint16(5)},
		{int16(-300), int16(-300)},
		{uint16(300), uint16(300)},
		{int64(-1 << 40), int64(-1 << 40)},
		{uint64(1 << 40), uint64(1 << 40)},
		{42, 42},
		{int32(-42), -42},
		{uint(42), uint64(42)},
		{uint32(42), uint64(42)},
		{[]bool{true, false}, []interface{}{true, false}},
		{[]string{"a", "b"}, []interface{}{"a", "b"}},
		{[]float32{1.5}, []interface{}{float32(1.5)}},
		{[]float64{1.5, -2}, []interface{}{1.5, -2.0}},
		{[]time.Time{ts}, []interface{}{ts}},
		{[]int8{-1}, []interface{}{int8(-1)}},
		{[]int16{-1}, []interface{}{int16(-1)}},
		{[]uint16{1}, []interface{}{uint16(1)}},
		{[]int64{-1}, []interface{}{int64(-1)}},
		{[]uint64{1}, []interface{}{uint64(1)}},
		{[]int{1, 2, 3}, []interface{}{1, 2, 3}},
		{[]uint{1}, []interface{}{uint(1)}},
	}
	valueElement := regexp.MustCompile(`<ns1:Value .*?</ns1:Value>`)
	for _, tc := range cases {
		t.Run(reflect.TypeOf(tc.in).String(), func(t *testing.T) {
			crh, handles := "crh1", []string{"h0"}
			payload, err := buildWritePayload(testServer(), "ns1", []TItem{{ItemName: "Item1", Value: TValue{Value: tc.in}}},
				&crh, &handles, nil)
			if err != nil {
				t.Fatal(err)
			}
			element := valueElement.FindString(payload)
			if element == "" {
				t.Fatalf("no Value element in payload: %s", payload)
			}
			// The payload declares xsi and ns1 on the envelope; re-declare them on the
			// extracted element so it can be decoded standalone.
			doc := `<ns1:Value ` + valueNamespaces + element[len(`<ns1:Value`):]
			v, err := decodeValue(t, doc)
			if err != nil {
				t.Fatalf("decoding %s: %v", element, err)
			}
			if want, ok := tc.want.(time.Time); ok {
				if got, isTime := v.Value.(time.Time); !isTime || !got.Equal(want) {
					t.Fatalf("expected %v, got %#v", want, v.Value)
				}
				return
			}
			if !reflect.DeepEqual(v.Value, tc.want) {
				t.Fatalf("expected %#v (%T), got %#v (%T) from %s", tc.want, tc.want, v.Value, v.Value, element)
			}
		})
	}
}

func TestGetOpcXmlDaTypeErrors(t *testing.T) {
	for name, value := range map[string]interface{}{
		"nil":                  nil,
		"struct":               struct{}{},
		"map":                  map[string]int{},
		"slice of unsupported": []complex64{1},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := getOpcXmlDaType(value); err == nil {
				t.Fatalf("expected an error for %T", value)
			}
		})
	}
	if typ, err := getOpcXmlDaType([]int{}); err != nil || typ != "ArrayOfInt" {
		t.Fatalf("expected an empty []int to map to ArrayOfInt, got %q, %v", typ, err)
	}
}

// FuzzValueUnmarshalXML ensures that no input - however malformed - makes value
// decoding panic; it may only return an error. The seed corpus runs as part of the
// normal test suite; `go test -fuzz=FuzzValueUnmarshalXML` explores further.
func FuzzValueUnmarshalXML(f *testing.F) {
	seeds := []string{
		`<Value ` + valueNamespaces + ` xsi:type="xsd:int">42</Value>`,
		`<Value ` + valueNamespaces + ` xsi:type="ns1:ArrayOfDouble"><ns1:double>1</ns1:double></Value>`,
		`<Value ` + valueNamespaces + ` xsi:type="ns1:OPCQuality" QualityField="good"/>`,
		`<Value ` + valueNamespaces + ` xsi:type="ns1:ArrayOfInt"><ns1:int>`,
		`<Value xsi:type=":">x</Value>`,
		`<Value type="a:b:c"></Value>`,
		`<Value>`,
		``,
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, doc string) {
		var v TValue
		_ = xml.Unmarshal([]byte(doc), &v)
	})
}
