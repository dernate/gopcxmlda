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
		{"xsd:base64Binary", "AQID", []byte{1, 2, 3}},
		{"xsd:base64Binary", "AQ\n ID", []byte{1, 2, 3}}, // whitespace is allowed inside
		{"xsd:base64Binary", "", []byte{}},
		{"xsd:duration", "P1Y2M3DT4H5M6.5S", "P1Y2M3DT4H5M6.5S"},
		{"xsd:duration", " -PT0.5S ", "-PT0.5S"},
		{"xsd:date", "2026-09-25", time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)},
		{"xsd:date", "2026-09-25+02:00", time.Date(2026, 9, 24, 22, 0, 0, 0, time.UTC)},
		{"xsd:time", "15:38:06.5", time.Date(0, 1, 1, 15, 38, 6, 500000000, time.UTC)},
		{"xsd:time", "15:38:06Z", time.Date(0, 1, 1, 15, 38, 6, 0, time.UTC)},
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
		"invalid date":            `<Value ` + valueNamespaces + ` xsi:type="xsd:date">25.09.2026</Value>`,
		"invalid time":            `<Value ` + valueNamespaces + ` xsi:type="xsd:time">3pm</Value>`,
		"invalid base64Binary":    `<Value ` + valueNamespaces + ` xsi:type="xsd:base64Binary">not*base64</Value>`,
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
		{ts, ts},
		{time.Date(2026, 9, 25, 15, 38, 6, 123456789, time.FixedZone("CEST", 2*3600)),
			time.Date(2026, 9, 25, 13, 38, 6, 123456789, time.UTC)},
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
		{[]byte{1, 2, 3}, []byte{1, 2, 3}},
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

// TestValueAcceptsUnprefixedXsiType covers the form the specification's own examples
// use: a type in the default namespace, without a prefix.
func TestValueAcceptsUnprefixedXsiType(t *testing.T) {
	const opcDefault = `xmlns="http://opcfoundation.org/webservices/XMLDA/1.0/" ` +
		`xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`
	v, err := decodeValue(t, `<Value `+opcDefault+` xsi:type="ArrayOfInt"><int>-2147483648</int><int>0</int></Value>`)
	if err != nil {
		t.Fatal(err)
	}
	if v.Type != "ArrayOfInt" || v.Namespace != "" || !reflect.DeepEqual(v.Value, []interface{}{-2147483648, 0}) {
		t.Fatalf("unexpected value: %+v", v)
	}

	v, err = decodeValue(t, `<Value `+opcDefault+` xsi:type="string">plain</Value>`)
	if err != nil {
		t.Fatal(err)
	}
	if v.Type != "string" || v.Value != "plain" {
		t.Fatalf("unexpected value: %+v", v)
	}
}

// TestValueDecodesArrayOfAnyType uses the specification's example: mixed simple
// types plus a nested array.
func TestValueDecodesArrayOfAnyType(t *testing.T) {
	doc := `<Value xmlns="http://opcfoundation.org/webservices/XMLDA/1.0/" ` +
		`xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" ` +
		`xsi:type="ArrayOfAnyType">
  <anyType xsi:type="xsd:byte">127</anyType>
  <anyType xsi:type="xsd:unsignedByte">255</anyType>
  <anyType xsi:type="xsd:string">Hello&lt;&gt;World</anyType>
  <anyType xsi:type="ArrayOfInt">
    <int>-2147483648</int>
    <int>0</int>
    <int>2147483647</int>
  </anyType>
  <anyType xsi:type="xsd:dateTime">2026-09-25T15:38:06</anyType>
</Value>`
	v, err := decodeValue(t, doc)
	if err != nil {
		t.Fatal(err)
	}
	want := []interface{}{
		int16(127), uint16(255), "Hello<>World",
		[]interface{}{-2147483648, 0, 2147483647},
		time.Date(2026, 9, 25, 15, 38, 6, 0, time.UTC),
	}
	if v.Type != "ArrayOfAnyType" || !reflect.DeepEqual(v.Value, want) {
		t.Fatalf("expected %#v, got %#v", want, v.Value)
	}

	v, err = decodeValue(t, `<Value `+valueNamespaces+` xsi:type="ns1:ArrayOfAnyType"></Value>`)
	if err != nil || len(v.Value.([]interface{})) != 0 {
		t.Fatalf("expected an empty ArrayOfAnyType, got %+v, %v", v, err)
	}

	if _, err := decodeValue(t, `<Value `+valueNamespaces+` xsi:type="ns1:ArrayOfAnyType"><ns1:anyType>1</ns1:anyType></Value>`); err == nil {
		t.Fatal("expected an error for an ArrayOfAnyType element without xsi:type")
	}
}

// TestDateTimeWithoutOffsetIsUTC pins down that an xsd:dateTime without a zone
// designator - valid per XML Schema - is read as UTC everywhere a dateTime occurs:
// scalar values, array elements, item timestamps and reply times.
func TestDateTimeWithoutOffsetIsUTC(t *testing.T) {
	want := time.Date(2026, 9, 25, 15, 38, 6, 500000000, time.UTC)

	v, err := decodeValue(t, `<Value `+valueNamespaces+` xsi:type="xsd:dateTime">2026-09-25T15:38:06.5</Value>`)
	if err != nil {
		t.Fatal(err)
	}
	if got := v.Value.(time.Time); !got.Equal(want) || got.Location() != time.UTC {
		t.Errorf("scalar: expected %v in UTC, got %v", want, got)
	}

	v, err = decodeValue(t, `<Value `+valueNamespaces+` xsi:type="ns1:ArrayOfDateTime"><ns1:dateTime> 2026-09-25T15:38:06.5 </ns1:dateTime></Value>`)
	if err != nil {
		t.Fatal(err)
	}
	if got := v.Value.([]interface{})[0].(time.Time); !got.Equal(want) {
		t.Errorf("array: expected %v, got %v", want, got)
	}

	doc := `<Envelope><Body><ReadResponse>
<ReadResult ReplyTime="2026-09-25T15:38:06.5" RcvTime="2026-09-25T15:38:06.5+02:00" ServerState="running"/>
<RItemList><Items ItemName="a" Timestamp="2026-09-25T15:38:06.5"/><Items ItemName="b"/></RItemList>
</ReadResponse></Body></Envelope>`
	var r TRead
	if err := xml.Unmarshal([]byte(doc), &r); err != nil {
		t.Fatal(err)
	}
	result := r.Response.Result
	if !result.ReplyTime.Equal(want) || !result.ReceiveTime.Equal(want.Add(-2*time.Hour)) {
		t.Errorf("reply times: expected %v / %v, got %v / %v", want, want.Add(-2*time.Hour), result.ReplyTime, result.ReceiveTime)
	}
	if result.ServerState != "running" {
		t.Errorf("expected the other TBaseResult attributes to still decode, got %+v", result)
	}
	items := r.Response.ItemList.Items
	if !items[0].Timestamp.Equal(want) || items[0].ItemName != "a" {
		t.Errorf("item timestamp: expected %v, got %+v", want, items[0])
	}
	if !items[1].Timestamp.IsZero() {
		t.Errorf("expected a zero Timestamp when the attribute is absent, got %v", items[1].Timestamp)
	}

	var bad TRead
	badDoc := `<Envelope><Body><ReadResponse><RItemList><Items Timestamp="yesterday"/></RItemList></ReadResponse></Body></Envelope>`
	if err := xml.Unmarshal([]byte(badDoc), &bad); err == nil {
		t.Error("expected an error for an invalid Timestamp attribute")
	}
}

// TestValueTypeQualifierIsExposed covers the transmission the specification
// recommends for types .NET couldn't handle: a duration sent as string (and a date as
// dateTime), with ValueTypeQualifier naming the intended type.
func TestValueTypeQualifierIsExposed(t *testing.T) {
	doc := soapEnvelope(`<ReadResponse xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><RItemList>` +
		`<Items ItemName="Runtime" ValueTypeQualifier="xsd:duration"><Value xsi:type="xsd:string">P1DT2H</Value></Items>` +
		`<Items ItemName="Commissioned" ValueTypeQualifier="xsd:date"><Value xsi:type="xsd:dateTime">2026-09-25T00:00:00Z</Value></Items>` +
		`<Items ItemName="Power"><Value xsi:type="xsd:int">74</Value></Items>` +
		`</RItemList></ReadResponse>`)
	var read TRead
	if err := xml.Unmarshal([]byte(doc), &read); err != nil {
		t.Fatal(err)
	}
	items := read.Response.ItemList.Items
	if items[0].ValueTypeQualifier != "xsd:duration" || items[0].Value.Value != "P1DT2H" {
		t.Errorf("unexpected duration item: %+v", items[0])
	}
	if items[1].ValueTypeQualifier != "xsd:date" || !items[1].Value.Value.(time.Time).Equal(time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("unexpected date item: %+v", items[1])
	}
	if items[2].ValueTypeQualifier != "" {
		t.Errorf("expected no qualifier on a plain value, got %q", items[2].ValueTypeQualifier)
	}
}
