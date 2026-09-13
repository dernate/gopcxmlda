package gopcxmlda

import (
	"math"
	"strings"
	"testing"
	"time"
)

// TestBuildReadPayloadOmitsMaxAgeWhenUnset pins down that the payload of a Read without
// any MaxAge is unchanged from before the attribute existed, so existing callers keep
// sending exactly what they used to.
func TestBuildReadPayloadOmitsMaxAgeWhenUnset(t *testing.T) {
	crh := "crh1"
	handles := []string{"h0"}
	items := []TItem{{ItemName: "Item1"}}
	payload, err := buildReadPayload(testServer(), &crh, &handles, "ns1", items, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(payload, "MaxAge") {
		t.Fatalf("expected no MaxAge attribute when none was requested, got: %s", payload)
	}
}

// TestBuildReadPayloadRendersItemMaxAgeZero is the central case: MaxAge 0 requests a
// device read and is also the zero value of an int, so it must survive rather than be
// dropped as "empty".
func TestBuildReadPayloadRendersItemMaxAgeZero(t *testing.T) {
	crh := "crh1"
	handles := []string{"h0"}
	items := []TItem{{ItemName: "Item1", MaxAge: MaxAgeDevice()}}
	payload, err := buildReadPayload(testServer(), &crh, &handles, "ns1", items, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload, `MaxAge="0"`) {
		t.Fatalf("expected MaxAge=\"0\" on the item, got: %s", payload)
	}
}

// TestBuildReadPayloadRendersPerItemMaxAge checks that each item carries its own value
// and that an item without one stays free of the attribute, so the list-level default
// can apply to it.
func TestBuildReadPayloadRendersPerItemMaxAge(t *testing.T) {
	crh := "crh1"
	handles := []string{"h0", "h1", "h2"}
	items := []TItem{
		{ItemName: "Item1", MaxAge: MaxAgeMillis(500)},
		{ItemName: "Item2", MaxAge: MaxAgeDevice()},
		{ItemName: "Item3"},
	}
	payload, err := buildReadPayload(testServer(), &crh, &handles, "ns1", items, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload, `ItemName="Item1" ClientItemHandle="h0" MaxAge="500"`) {
		t.Fatalf("expected MaxAge=\"500\" on Item1, got: %s", payload)
	}
	if !strings.Contains(payload, `ItemName="Item2" ClientItemHandle="h1" MaxAge="0"`) {
		t.Fatalf("expected MaxAge=\"0\" on Item2, got: %s", payload)
	}
	if !strings.Contains(payload, `ItemName="Item3" ClientItemHandle="h2">`) {
		t.Fatalf("expected no MaxAge attribute on Item3, got: %s", payload)
	}
}

// TestBuildReadPayloadRendersListMaxAgeOnItemList pins down that the list-level MaxAge
// lands on the ItemList element and not among the <Options> attributes - MaxAge is not
// part of the RequestOptions type, so rendering it there would be invalid.
func TestBuildReadPayloadRendersListMaxAgeOnItemList(t *testing.T) {
	crh := "crh1"
	handles := []string{"h0"}
	items := []TItem{{ItemName: "Item1"}}
	options := map[string]interface{}{"ReturnItemTime": true, MaxAgeOption: 1000}
	payload, err := buildReadPayload(testServer(), &crh, &handles, "ns1", items, options)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload, `<ns1:ItemList MaxAge="1000">`) {
		t.Fatalf("expected MaxAge=\"1000\" on the ItemList element, got: %s", payload)
	}
	optionsElement := payload[strings.Index(payload, "<ns1:Options"):]
	optionsElement = optionsElement[:strings.Index(optionsElement, ">")]
	if strings.Contains(optionsElement, "MaxAge") {
		t.Fatalf("expected MaxAge not to be rendered as an Options attribute, got: %s", optionsElement)
	}
	if !strings.Contains(optionsElement, `ReturnItemTime="true"`) {
		t.Fatalf("expected the other options to survive, got: %s", optionsElement)
	}
}

// TestBuildReadPayloadListMaxAgeOnlyDoesNotEmitOptions checks that a MaxAge-only
// options map doesn't produce an empty <Options/> element, matching the behavior for a
// map that was empty to begin with.
func TestBuildReadPayloadListMaxAgeOnlyDoesNotEmitOptions(t *testing.T) {
	crh := "crh1"
	handles := []string{"h0"}
	items := []TItem{{ItemName: "Item1"}}
	payload, err := buildReadPayload(testServer(), &crh, &handles, "ns1", items,
		map[string]interface{}{MaxAgeOption: 250})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(payload, "ns1:Options") {
		t.Fatalf("expected no Options element for a MaxAge-only options map, got: %s", payload)
	}
	if !strings.Contains(payload, `<ns1:ItemList MaxAge="250">`) {
		t.Fatalf("expected MaxAge=\"250\" on the ItemList element, got: %s", payload)
	}
}

// TestBuildReadPayloadDoesNotMutateOptions pins down that extracting MaxAge leaves the
// caller's map untouched, so a shared options map can be reused across requests.
func TestBuildReadPayloadDoesNotMutateOptions(t *testing.T) {
	crh := "crh1"
	handles := []string{"h0"}
	items := []TItem{{ItemName: "Item1"}}
	options := map[string]interface{}{"ReturnItemTime": true, MaxAgeOption: 1000}
	if _, err := buildReadPayload(testServer(), &crh, &handles, "ns1", items, options); err != nil {
		t.Fatal(err)
	}
	if _, ok := options[MaxAgeOption]; !ok {
		t.Fatalf("expected the caller's options map to keep its MaxAge key, got: %v", options)
	}
	if len(options) != 2 {
		t.Fatalf("expected the caller's options map to be unchanged, got: %v", options)
	}
}

// TestBuildReadPayloadItemMaxAgeIsCopied pins down that mutating the int a caller
// pointed TItem.MaxAge at after the call can't change what was rendered.
func TestBuildReadPayloadItemMaxAgeIsCopied(t *testing.T) {
	crh := "crh1"
	handles := []string{"h0"}
	maxAge := 500
	items := []TItem{{ItemName: "Item1", MaxAge: &maxAge}}
	payload, err := buildReadPayload(testServer(), &crh, &handles, "ns1", items, nil)
	if err != nil {
		t.Fatal(err)
	}
	maxAge = 9999
	if !strings.Contains(payload, `MaxAge="500"`) {
		t.Fatalf("expected the rendered MaxAge to be a copy taken at build time, got: %s", payload)
	}
}

func TestBuildReadPayloadRejectsInvalidMaxAge(t *testing.T) {
	crh := "crh1"

	t.Run("negative item MaxAge", func(t *testing.T) {
		handles := []string{"h0"}
		items := []TItem{{ItemName: "Item1", MaxAge: MaxAgeMillis(-1)}}
		if _, err := buildReadPayload(testServer(), &crh, &handles, "ns1", items, nil); err == nil {
			t.Fatal("expected an error for a negative item MaxAge")
		}
	})

	t.Run("item MaxAge above xs:int", func(t *testing.T) {
		handles := []string{"h0"}
		items := []TItem{{ItemName: "Item1", MaxAge: MaxAgeMillis(math.MaxInt32 + 1)}}
		if _, err := buildReadPayload(testServer(), &crh, &handles, "ns1", items, nil); err == nil {
			t.Fatal("expected an error for an item MaxAge above the xs:int range")
		}
	})

	t.Run("unsupported option type", func(t *testing.T) {
		handles := []string{"h0"}
		items := []TItem{{ItemName: "Item1"}}
		options := map[string]interface{}{MaxAgeOption: struct{}{}}
		if _, err := buildReadPayload(testServer(), &crh, &handles, "ns1", items, options); err == nil {
			t.Fatal("expected an error for an option value that is not a number of milliseconds")
		}
	})

	t.Run("fractional option value", func(t *testing.T) {
		handles := []string{"h0"}
		items := []TItem{{ItemName: "Item1"}}
		options := map[string]interface{}{MaxAgeOption: 12.5}
		if _, err := buildReadPayload(testServer(), &crh, &handles, "ns1", items, options); err == nil {
			t.Fatal("expected an error for a fractional number of milliseconds")
		}
	})
}

func TestParseMaxAgeOptionAcceptedTypes(t *testing.T) {
	cases := []struct {
		name string
		raw  interface{}
		want *int
	}{
		{"nil", nil, nil},
		{"int", 500, MaxAgeMillis(500)},
		{"int zero", 0, MaxAgeMillis(0)},
		{"int64", int64(500), MaxAgeMillis(500)},
		{"uint", uint(500), MaxAgeMillis(500)},
		{"float64 from JSON", float64(500), MaxAgeMillis(500)},
		{"string", " 500 ", MaxAgeMillis(500)},
		{"duration", 1500 * time.Millisecond, MaxAgeMillis(1500)},
		{"pointer", MaxAgeMillis(500), MaxAgeMillis(500)},
		{"nil pointer", (*int)(nil), nil},
		{"upper bound", math.MaxInt32, MaxAgeMillis(math.MaxInt32)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseMaxAgeOption(tc.raw)
			if err != nil {
				t.Fatalf("expected %v to be accepted, got: %v", tc.raw, err)
			}
			switch {
			case tc.want == nil && got != nil:
				t.Fatalf("expected no MaxAge, got %d", *got)
			case tc.want != nil && got == nil:
				t.Fatalf("expected MaxAge %d, got none", *tc.want)
			case tc.want != nil && *got != *tc.want:
				t.Fatalf("expected MaxAge %d, got %d", *tc.want, *got)
			}
		})
	}
}

func TestParseMaxAgeOptionRejectedValues(t *testing.T) {
	cases := []struct {
		name string
		raw  interface{}
	}{
		{"negative int", -1},
		{"negative duration", -time.Second},
		{"above xs:int", int64(math.MaxInt32) + 1},
		{"uint above xs:int", uint64(math.MaxInt32) + 1},
		{"non-numeric string", "soon"},
		{"NaN", math.NaN()},
		{"infinity", math.Inf(1)},
		{"bool", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseMaxAgeOption(tc.raw); err == nil {
				t.Fatalf("expected %v to be rejected", tc.raw)
			}
		})
	}
}

// TestMaxAgeHelpersReturnIndependentPointers pins down that the helpers hand out a
// fresh pointer each time, so reusing one across items can't couple them.
func TestMaxAgeHelpersReturnIndependentPointers(t *testing.T) {
	first := MaxAgeMillis(500)
	second := MaxAgeMillis(500)
	if first == second {
		t.Fatal("expected MaxAgeMillis to return a new pointer on every call")
	}
	if *MaxAgeDevice() != 0 {
		t.Fatal("expected MaxAgeDevice to request a device read (MaxAge 0)")
	}
}
