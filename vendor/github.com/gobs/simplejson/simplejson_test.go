package simplejson

import (
	"bytes"
	"encoding/json"
	"github.com/bmizerany/assert"
	"io/ioutil"
	"log"
	"strconv"
	"testing"
)

func TestSimplejson(t *testing.T) {
	var ok bool
	var err error

	log.SetOutput(ioutil.Discard)

	js, err := LoadBytes([]byte(`{ 
		"test": { 
			"string_array": ["asdf", "ghjk", "zxcv"],
			"array": [1, "2", 3],
			"arraywithsubs": [{"subkeyone": 1},
			{"subkeytwo": 2, "subkeythree": 3}],
			"int": 10,
			"float": 0.150,
			"bignum": 9223372036854775807,
			"string": "simplejson",
			"bool": true 
		}
	}`))

	assert.NotEqual(t, nil, js)
	assert.Equal(t, nil, err)

	_, ok = js.CheckGet("test")
	assert.Equal(t, true, ok)

	_, ok = js.CheckGet("missing_key")
	assert.Equal(t, false, ok)

	arr, _ := js.Get("test").Get("array").Array()
	assert.NotEqual(t, nil, arr)
	for i, v := range arr {
		var iv int
		switch v.(type) {
		case float64:
			iv = int(v.(float64))
		case string:
			iv, _ = strconv.Atoi(v.(string))
		}
		assert.Equal(t, i+1, iv)
	}

	aws := js.Get("test").Get("arraywithsubs")
	assert.NotEqual(t, nil, aws)
	var awsval int
	awsval, _ = aws.GetIndex(0).Get("subkeyone").Int()
	assert.Equal(t, 1, awsval)
	awsval, _ = aws.GetIndex(1).Get("subkeytwo").Int()
	assert.Equal(t, 2, awsval)
	awsval, _ = aws.GetIndex(1).Get("subkeythree").Int()
	assert.Equal(t, 3, awsval)

	i, _ := js.Get("test").Get("int").Int()
	assert.Equal(t, 10, i)

	i = js.Get("test").Get("int").MustInt()
	assert.Equal(t, 10, i)

	f, _ := js.Get("test").Get("float").Float64()
	assert.Equal(t, 0.150, f)

	f = js.Get("test").Get("float").MustFloat64()
	assert.Equal(t, 0.150, f)

	s, _ := js.Get("test").Get("string").String()
	assert.Equal(t, "simplejson", s)

	b, _ := js.Get("test").Get("bool").Bool()
	assert.Equal(t, true, b)

	mi := js.Get("test").Get("int").MustInt()
	assert.Equal(t, 10, mi)

	mi2 := js.Get("test").Get("missing_int").MustInt(5150)
	assert.Equal(t, 5150, mi2)

	ms := js.Get("test").Get("string").MustString()
	assert.Equal(t, "simplejson", ms)

	ms2 := js.Get("test").Get("missing_string").MustString("fyea")
	assert.Equal(t, "fyea", ms2)

	ma := js.Get("test").Get("array").MustArray()
	assert.Equal(t, ma, []interface{}{float64(1), "2", float64(3)})

	ma2 := js.Get("test").Get("missing_array").MustArray([]interface{}{"1", 2, "3"})
	assert.Equal(t, ma2, []interface{}{"1", 2, "3"})

	mm := js.Get("test").Get("arraywithsubs").GetIndex(0).MustMap()
	assert.Equal(t, mm, map[string]interface{}{"subkeyone": float64(1)})

	mm2 := js.Get("test").Get("missing_map").MustMap(map[string]interface{}{"found": false})
	assert.Equal(t, mm2, map[string]interface{}{"found": false})

	strs, err := js.Get("test").Get("string_array").StringArray()
	assert.Equal(t, err, nil)
	assert.Equal(t, strs[0], "asdf")
	assert.Equal(t, strs[1], "ghjk")
	assert.Equal(t, strs[2], "zxcv")

	gp, _ := js.GetPath("test", "string").String()
	assert.Equal(t, "simplejson", gp)

	gp2, _ := js.GetPath("test", "int").Int()
	assert.Equal(t, 10, gp2)

	js.Set("test", "setTest")
	assert.Equal(t, "setTest", js.Get("test").MustString())
}

func TestStdlibInterfaces(t *testing.T) {
	val := new(struct {
		Name   string `json:"name"`
		Params *Json  `json:"params"`
	})
	val2 := new(struct {
		Name   string `json:"name"`
		Params *Json  `json:"params"`
	})

	raw := `{"name":"myobject","params":{"string":"simplejson"}}`

	assert.Equal(t, nil, json.Unmarshal([]byte(raw), val))

	assert.Equal(t, "myobject", val.Name)
	assert.NotEqual(t, nil, val.Params.data)
	s, _ := val.Params.Get("string").String()
	assert.Equal(t, "simplejson", s)

	p, err := json.Marshal(val)
	assert.Equal(t, nil, err)
	assert.Equal(t, nil, json.Unmarshal(p, val2))
	assert.Equal(t, val, val2) // stable
}

func TestNil(t *testing.T) {
	var jp *Json
	assert.Equal(t, jp.Nil(), true)

	var j Json
	assert.Equal(t, j.Nil(), true)
}

func TestDump(t *testing.T) {
	v := map[string]interface{}{
		"name":  "test",
		"value": ">=10",
	}

	res, err := DumpString(v, EscapeHTML(false))
	assert.Equal(t, nil, err)
	assert.Equal(t, res, `{"name":"test","value":">=10"}`+"\n")
}

func TestDumpIndent(t *testing.T) {
	v := map[string]interface{}{
		"name":  "test",
		"value": ">=10",
	}

	res, err := DumpString(v, Indent("  "))
	assert.Equal(t, nil, err)
	assert.Equal(t, res, "{\n  \"name\": \"test\",\n  \"value\": \">=10\"\n}\n")
}

func TestRaw(t *testing.T) {
	v := map[string]interface{}{
		"name":  "test",
		"value": Raw(`{"a":1,"b":2}`),
	}

	res, err := DumpString(v)
	assert.Equal(t, nil, err)
	assert.Equal(t, res, `{"name":"test","value":{"a":1,"b":2}}`+"\n")
}

func TestLoadPartial(t *testing.T) {
	input := []byte(`{"a":1,"b":2} [1,2,3,4] {"hello": "there"}`)

	for len(input) > 0 {
		j, rest, err := LoadPartial(input)
		if err == nil {
			t.Log(j)
		} else {
			t.Log("ERROR", err)
			break
		}

		input = bytes.TrimSpace(rest)
	}
}
