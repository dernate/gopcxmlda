package gopcxmlda

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// MaxAgeOption is the key under which Read's options map carries a list-level MaxAge,
// i.e. the MaxAge attribute of the ReadRequestItemList that applies to every item that
// doesn't set TItem.MaxAge itself.
//
// It is deliberately handled apart from the other options: everything else in that map
// is rendered as an attribute of the request's <Options> element (the OPC-XML-DA
// RequestOptions type), and MaxAge is not one of RequestOptions' attributes - it lives
// on the item list. Read therefore removes this key from the map and renders it in the
// right place. The caller's map is never modified.
//
// Accepted values are any signed or unsigned integer type, a float with no fractional
// part, a time.Duration (truncated to whole milliseconds), a *int (nil meaning "not
// set"), or a decimal string. Values outside 0 to math.MaxInt32 are rejected, as are
// values of any other type.
const MaxAgeOption = "MaxAge"

// maxAgeLimit is the upper bound of the MaxAge attribute, which the specification
// declares as an xs:int - a signed 32-bit integer.
const maxAgeLimit = math.MaxInt32

// MaxAgeMillis returns a *int suitable for TItem.MaxAge, expressing the maximum age in
// milliseconds that a cached value may have. It is a convenience for taking the address
// of an int literal, which Go doesn't allow directly:
//
//	items := []TItem{{ItemName: "My/Item", MaxAge: MaxAgeMillis(500)}}
func MaxAgeMillis(milliseconds int) *int {
	return &milliseconds
}

// MaxAgeDevice returns a *int suitable for TItem.MaxAge that requests the most accurate
// data available - MaxAge 0, which the specification describes as analogous to a DEVICE
// read in classic OPC DA. Use it when a cached value must not be accepted:
//
//	items := []TItem{{ItemName: "My/Item", MaxAge: MaxAgeDevice()}}
func MaxAgeDevice() *int {
	return MaxAgeMillis(0)
}

// validateMaxAge checks that a MaxAge value fits the xs:int range the specification
// requires and isn't negative. A nil pointer (MaxAge not set) is valid and returns nil.
// context names the source of the value so the error points at the offending item.
func validateMaxAge(maxAge *int, context string) error {
	if maxAge == nil {
		return nil
	}
	if *maxAge < 0 || *maxAge > maxAgeLimit {
		return fmt.Errorf("gopcxmlda: MaxAge %d for %s is out of range, must be between 0 and %d milliseconds",
			*maxAge, context, maxAgeLimit)
	}
	return nil
}

// copyMaxAge returns a pointer to a copy of *maxAge, so that the rendered payload can't
// be changed by a caller mutating the int they handed over in a TItem.
func copyMaxAge(maxAge *int) *int {
	if maxAge == nil {
		return nil
	}
	value := *maxAge
	return &value
}

// splitMaxAgeOption separates the list-level MaxAge (see MaxAgeOption) from the
// remaining request options. It returns the parsed MaxAge (nil if the key is absent or
// explicitly nil) and the options map to render as <Options> attributes. The caller's
// map is left untouched; when the key is absent the same map is returned unchanged.
func splitMaxAgeOption(options map[string]interface{}) (*int, map[string]interface{}, error) {
	raw, ok := options[MaxAgeOption]
	if !ok {
		return nil, options, nil
	}

	maxAge, err := parseMaxAgeOption(raw)
	if err != nil {
		return nil, nil, err
	}

	rest := make(map[string]interface{}, len(options)-1)
	for key, value := range options {
		if key != MaxAgeOption {
			rest[key] = value
		}
	}
	return maxAge, rest, nil
}

// parseMaxAgeOption converts an options-map value into a number of milliseconds. The
// map is typed map[string]interface{}, so the value can arrive in whatever numeric
// shape the caller happened to have (including a float64 from decoded JSON or a
// time.Duration from a config package); everything that unambiguously denotes a whole
// number of milliseconds is accepted, everything else is an error rather than a
// silently mangled attribute.
func parseMaxAgeOption(raw interface{}) (*int, error) {
	switch value := raw.(type) {
	case nil:
		return nil, nil
	case *int:
		if value == nil {
			return nil, nil
		}
		return maxAgeFromInt64(int64(*value), *value)
	case time.Duration:
		return maxAgeFromInt64(value.Milliseconds(), raw)
	case int:
		return maxAgeFromInt64(int64(value), raw)
	case int8:
		return maxAgeFromInt64(int64(value), raw)
	case int16:
		return maxAgeFromInt64(int64(value), raw)
	case int32:
		return maxAgeFromInt64(int64(value), raw)
	case int64:
		return maxAgeFromInt64(value, raw)
	case uint:
		return maxAgeFromUint64(uint64(value), raw)
	case uint8:
		return maxAgeFromUint64(uint64(value), raw)
	case uint16:
		return maxAgeFromUint64(uint64(value), raw)
	case uint32:
		return maxAgeFromUint64(uint64(value), raw)
	case uint64:
		return maxAgeFromUint64(value, raw)
	case float32:
		return maxAgeFromFloat64(float64(value), raw)
	case float64:
		return maxAgeFromFloat64(value, raw)
	case string:
		milliseconds, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("gopcxmlda: %q option %q is not a whole number of milliseconds",
				MaxAgeOption, value)
		}
		return maxAgeFromInt64(milliseconds, raw)
	default:
		return nil, fmt.Errorf("gopcxmlda: %q option has unsupported type %T, want a whole number of milliseconds",
			MaxAgeOption, raw)
	}
}

func maxAgeFromInt64(milliseconds int64, raw interface{}) (*int, error) {
	if milliseconds < 0 || milliseconds > maxAgeLimit {
		return nil, outOfRangeMaxAgeError(raw)
	}
	return MaxAgeMillis(int(milliseconds)), nil
}

func maxAgeFromUint64(milliseconds uint64, raw interface{}) (*int, error) {
	if milliseconds > maxAgeLimit {
		return nil, outOfRangeMaxAgeError(raw)
	}
	return MaxAgeMillis(int(milliseconds)), nil
}

func maxAgeFromFloat64(milliseconds float64, raw interface{}) (*int, error) {
	if math.IsNaN(milliseconds) || math.IsInf(milliseconds, 0) || milliseconds != math.Trunc(milliseconds) {
		return nil, fmt.Errorf("gopcxmlda: %q option %v is not a whole number of milliseconds",
			MaxAgeOption, raw)
	}
	return maxAgeFromInt64(int64(milliseconds), raw)
}

func outOfRangeMaxAgeError(raw interface{}) error {
	return fmt.Errorf("gopcxmlda: %q option %v is out of range, must be between 0 and %d milliseconds",
		MaxAgeOption, raw, maxAgeLimit)
}
