package simplejson

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
)

var (
	ErrNoMap       = errors.New("type assertion to map[string]interface{} failed")
	ErrNoArray     = errors.New("type assertion to []interface{} failed")
	ErrNoBool      = errors.New("type assertion to bool failed")
	ErrNoString    = errors.New("type assertion to string failed")
	ErrNoFloat     = errors.New("type assertion to float64 failed")
	ErrNoByteArray = errors.New("type assertion to []byte failed")
)

// returns the current implementation version
func Version() string {
	return "0.4.3-gobs"
}

type Json struct {
	data interface{}
}

// Cast to Json{}
func AsJson(obj interface{}) *Json {
	return &Json{obj}
}

// Load loads JSON from `reader` io.Reader and return a new `Json` object
func Load(reader io.Reader) (*Json, error) {
	j := new(Json)

	dec := json.NewDecoder(reader)
	err := dec.Decode(&j.data)
	if err != nil {
		return nil, err
	} else {
		return j, nil
	}
}

// LoadBytes load JSON from `body` []byte and return a new `Json` object
func LoadBytes(body []byte) (*Json, error) {
	j := new(Json)
	err := j.UnmarshalJSON(body)
	if err != nil {
		return nil, err
	}
	return j, nil
}

// LoadString loads JSON from `body` string and return a new `Json` object
func LoadString(body string) (*Json, error) {
	return LoadBytes([]byte(body))
}

// LoadPartial load JSON from `body` []byte and return a new `Json` object.
// It also returns any remaining bytes from the input.
func LoadPartial(body []byte) (*Json, []byte, error) {
	j := new(Json)
	b := bytes.NewReader(body)

	dec := json.NewDecoder(b)
	err := dec.Decode(&j.data)

	n := (dec.Buffered().(*bytes.Reader)).Len() + b.Len()
	rest := body[len(body)-n:]

	if err != nil {
		return nil, rest, err
	}
	return j, rest, nil
}

// LoadPartialString load JSON from `body` string and return a new `Json` object.
// It also returns any remaining bytes from the input.
func LoadPartialString(body string) (*Json, string, error) {
	j, rest, err := LoadPartial([]byte(body))
	return j, string(rest), err
}

// DumpOption is the type of DumpBytes/DumpString options
type DumpOption func(enc *json.Encoder)

// EscapeHTML instruct DumpBytes/DumpString to escape HTML in JSON valus (if true)
func EscapeHTML(escape bool) DumpOption {
	return func(enc *json.Encoder) {
		enc.SetEscapeHTML(escape)
	}
}

// Indent sets the indentation level (passed as string of spaces) for DumpBytes/DumpString
func Indent(s string) DumpOption {
	return func(enc *json.Encoder) {
		enc.SetIndent("", s)
	}
}

// DumpBytes convert Go data object to JSON []byte
func DumpBytes(obj interface{}, options ...DumpOption) ([]byte, error) {
	// turns out json.Marsh HTMLEncodes string values to please some browsers
	// result, err := json.Marshal(obj)
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)

	for _, opt := range options {
		opt(enc)
	}

	err := enc.Encode(obj)
	return b.Bytes(), err
}

// DumpString encode Go data object to JSON string
func DumpString(obj interface{}, options ...DumpOption) (string, error) {
	if result, err := DumpBytes(obj, options...); err != nil {
		return "", err
	} else {
		return string(result), nil
	}
}

// MustDumpBytes encode Go data object to JSON []byte (panic in case of error)
func MustDumpBytes(obj interface{}, options ...DumpOption) []byte {
	if res, err := DumpBytes(obj, options...); err != nil {
		panic(err)
	} else {
		return res
	}
}

// MustDumpString encode Go data object to JSON string (panic in case of error)
func MustDumpString(obj interface{}, options ...DumpOption) string {
	if res, err := DumpString(obj, options...); err != nil {
		panic(err)
	} else {
		return res
	}
}

// Encode returns its marshaled data as `[]byte`
func (j *Json) Encode() ([]byte, error) {
	return j.MarshalJSON()
}

// Implements the json.Unmarshaler interface.
func (j *Json) UnmarshalJSON(p []byte) error {
	return json.Unmarshal(p, &j.data)
}

// Implements the json.Marshaler interface.
func (j *Json) MarshalJSON() ([]byte, error) {
	return DumpBytes(&j.data)
}

// Set modifies `Json` map by `key` and `value`
// Useful for changing single key/value in a `Json` object easily.
func (j *Json) Set(key string, val interface{}) {
	m, err := j.Map()
	if err != nil {
		return
	}
	m[key] = val
}

// Nil returns true if this json object is nil
func (j *Json) Nil() bool {
	return j == nil || j.data == nil
}

// Get returns a pointer to a new `Json` object
// for `key` in its `map` representation
//
// useful for chaining operations (to traverse a nested JSON):
//    js.Get("top_level").Get("dict").Get("value").Int()
func (j *Json) Get(key string) *Json {
	m, err := j.Map()
	if err == nil {
		if val, ok := m[key]; ok {
			return &Json{val}
		}
	}
	return &Json{nil}
}

// GetPath searches for the item as specified by the branch
// without the need to deep dive using Get()'s.
//
//   js.GetPath("top_level", "dict")
func (j *Json) GetPath(branch ...string) *Json {
	jin := j
	for i := range branch {
		m, err := jin.Map()
		if err != nil {
			return &Json{nil}
		}
		if val, ok := m[branch[i]]; ok {
			jin = &Json{val}
		} else {
			return &Json{nil}
		}
	}
	return jin
}

// GetIndex resturns a pointer to a new `Json` object
// for `index` in its `array` representation
//
// this is the analog to Get when accessing elements of
// a json array instead of a json object:
//    js.Get("top_level").Get("array").GetIndex(1).Get("key").Int()
func (j *Json) GetIndex(index int) *Json {
	a, err := j.Array()
	if err == nil {
		if len(a) > index {
			return &Json{a[index]}
		}
	}
	return &Json{nil}
}

// CheckGet returns a pointer to a new `Json` object and
// a `bool` identifying success or failure
//
// useful for chained operations when success is important:
//    if data, ok := js.Get("top_level").CheckGet("inner"); ok {
//        log.Println(data)
//    }
func (j *Json) CheckGet(key string) (*Json, bool) {
	m, err := j.Map()
	if err == nil {
		if val, ok := m[key]; ok {
			return &Json{val}, true
		}
	}
	return nil, false
}

// Return value as interface{}
func (j *Json) Data() interface{} {
	return j.data
}

// Map type asserts to `map`
func (j *Json) Map() (map[string]interface{}, error) {
	if m, ok := (j.data).(map[string]interface{}); ok {
		return m, nil
	}
	return nil, ErrNoMap
}

// Array type asserts to an `array`
func (j *Json) Array() ([]interface{}, error) {
	if a, ok := (j.data).([]interface{}); ok {
		return a, nil
	}
	return nil, ErrNoArray
}

// MakeArray always return an `array`
// (this is useful for HAL responses that can return either an array or a single element ):
func (j *Json) MakeArray() []interface{} {
	if a, ok := (j.data).([]interface{}); ok {
		return a
	} else {
		return []interface{}{j.data}
	}
}

// Bool type asserts to `bool`
func (j *Json) Bool() (bool, error) {
	if s, ok := (j.data).(bool); ok {
		return s, nil
	}
	return false, ErrNoBool
}

// String type asserts to `string`
func (j *Json) String() (string, error) {
	if s, ok := (j.data).(string); ok {
		return s, nil
	}
	return "", ErrNoString
}

// Float64 type asserts to `float64`
func (j *Json) Float64() (float64, error) {
	if i, ok := (j.data).(float64); ok {
		return i, nil
	}
	return -1, ErrNoFloat
}

// Int type asserts to `float64` then converts to `int`
func (j *Json) Int() (int, error) {
	if f, ok := (j.data).(float64); ok {
		return int(f), nil
	}

	return -1, ErrNoFloat
}

// Int type asserts to `float64` then converts to `int64`
func (j *Json) Int64() (int64, error) {
	if f, ok := (j.data).(float64); ok {
		return int64(f), nil
	}

	return -1, ErrNoFloat
}

// Bytes type asserts to `[]byte`
func (j *Json) Bytes() ([]byte, error) {
	if s, ok := (j.data).(string); ok {
		return []byte(s), nil
	}
	return nil, ErrNoByteArray
}

// StringArray type asserts to an `array` of `string`
func (j *Json) StringArray() ([]string, error) {
	arr, err := j.Array()
	if err != nil {
		return nil, err
	}
	retArr := make([]string, 0, len(arr))
	for _, a := range arr {
		s, ok := a.(string)
		if !ok {
			return nil, err
		}
		retArr = append(retArr, s)
	}
	return retArr, nil
}

// MustArray guarantees the return of a `[]interface{}` (with optional default)
//
// useful when you want to interate over array values in a succinct manner:
//		for i, v := range js.Get("results").MustArray() {
//			fmt.Println(i, v)
//		}
func (j *Json) MustArray(args ...[]interface{}) []interface{} {
	var def []interface{}
	switch len(args) {
	case 0:
		break
	case 1:
		def = args[0]
	default:
		log.Panicf("MustArray() received too many arguments %d", len(args))
	}

	a, err := j.Array()
	if err == nil {
		return a
	}

	return def
}

// MustMap guarantees the return of a `map[string]interface{}` (with optional default)
//
// useful when you want to interate over map values in a succinct manner:
//		for k, v := range js.Get("dictionary").MustMap() {
//			fmt.Println(k, v)
//		}
func (j *Json) MustMap(args ...map[string]interface{}) map[string]interface{} {
	var def map[string]interface{}
	switch len(args) {
	case 0:
		break
	case 1:
		def = args[0]
	default:
		log.Panicf("MustMap() received too many arguments %d", len(args))
	}

	a, err := j.Map()
	if err == nil {
		return a
	}

	return def
}

// MustString guarantees the return of a `string` (with optional default)
//
// useful when you explicitly want a `string` in a single value return context:
//     myFunc(js.Get("param1").MustString(), js.Get("optional_param").MustString("my_default"))
func (j *Json) MustString(args ...string) string {
	var def string

	switch len(args) {
	case 0:
		break
	case 1:
		def = args[0]
	default:
		log.Panicf("MustString() received too many arguments %d", len(args))
	}

	s, err := j.String()
	if err == nil {
		return s
	}

	return def
}

// MustInt guarantees the return of an `int` (with optional default)
//
// useful when you explicitly want an `int` in a single value return context:
//     myFunc(js.Get("param1").MustInt(), js.Get("optional_param").MustInt(5150))
func (j *Json) MustInt(args ...int) int {
	var def int

	switch len(args) {
	case 0:
		break
	case 1:
		def = args[0]
	default:
		log.Panicf("MustInt() received too many arguments %d", len(args))
	}

	i, err := j.Int()
	if err == nil {
		return i
	}

	return def
}

// MustInt64 guarantees the return of an `int64` (with optional default)
//
// useful when you explicitly want an `int64` in a single value return context:
//     myFunc(js.Get("param1").MustInt64(), js.Get("optional_param").MustInt64(5150))
func (j *Json) MustInt64(args ...int64) int64 {
	var def int64

	switch len(args) {
	case 0:
		break
	case 1:
		def = args[0]
	default:
		log.Panicf("MustInt64() received too many arguments %d", len(args))
	}

	i, err := j.Int64()
	if err == nil {
		return i
	}

	return def
}

// MustFloat64 guarantees the return of a `float64` (with optional default)
//
// useful when you explicitly want a `float64` in a single value return context:
//     myFunc(js.Get("param1").MustFloat64(), js.Get("optional_param").MustFloat64(5.150))
func (j *Json) MustFloat64(args ...float64) float64 {
	var def float64

	switch len(args) {
	case 0:
		break
	case 1:
		def = args[0]
	default:
		log.Panicf("MustFloat64() received too many arguments %d", len(args))
	}

	i, err := j.Float64()
	if err == nil {
		return i
	}

	return def
}

//
// basic type for quick conversion to JSON
//
type Bag map[string]interface{}

// this is an alias for json.RawMessage, for clients that don't include encoding/json
type Raw = json.RawMessage

// this is an alias for json.Marshal, for clients that don't include encoding/json
func Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// this is an alias for json.Unmarshal, for clients that don't include encoding/json
func Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
