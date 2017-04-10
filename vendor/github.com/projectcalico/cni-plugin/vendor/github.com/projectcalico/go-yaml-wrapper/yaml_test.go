package yaml

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"testing"
)

type MarshalTest struct {
	A string
	B int64
	C float64
}

func TestMarshal(t *testing.T) {
	f64String := strconv.FormatFloat(math.MaxFloat64, 'g', -1, 64)
	s := MarshalTest{"a", math.MaxInt64, math.MaxFloat64}
	e := []byte(fmt.Sprintf("A: a\nB: %d\nC: %s\n", math.MaxInt64, f64String))

	y, err := Marshal(s)
	if err != nil {
		t.Errorf("error marshaling YAML: %v", err)
	}

	if !reflect.DeepEqual(y, e) {
		t.Errorf("marshal YAML was unsuccessful, expected: %#v, got: %#v",
			string(e), string(y))
	}
}

type UnmarshalString struct {
	A    string
	True string
}

type UnmarshalStringMap struct {
	A map[string]string
}

type UnmarshalNestedString struct {
	A NestedString
}

type NestedString struct {
	A string
}

type UnmarshalSlice struct {
	A []NestedSlice
}

type NestedSlice struct {
	B string
	C *string
}

type UnmarshalFloat64 struct {
	A float64
}

func TestUnmarshal(t *testing.T) {
	y := []byte("a: 1")
	s1 := UnmarshalString{}
	e1 := UnmarshalString{A: "1"}
	unmarshal(t, y, &s1, &e1)

	y = []byte("a: true")
	s1 = UnmarshalString{}
	e1 = UnmarshalString{A: "true"}
	unmarshal(t, y, &s1, &e1)

	y = []byte("true: 1")
	s1 = UnmarshalString{}
	e1 = UnmarshalString{True: "1"}
	unmarshal(t, y, &s1, &e1)

	y = []byte("a:\n  a: 1")
	s2 := UnmarshalNestedString{}
	e2 := UnmarshalNestedString{NestedString{"1"}}
	unmarshal(t, y, &s2, &e2)

	y = []byte("a:\n  - b: abc\n    c: def\n  - b: 123\n    c: 456\n")
	s3 := UnmarshalSlice{}
	e3 := UnmarshalSlice{[]NestedSlice{NestedSlice{"abc", strPtr("def")}, NestedSlice{"123", strPtr("456")}}}
	unmarshal(t, y, &s3, &e3)

	y = []byte("a:\n  b: 1")
	s4 := UnmarshalStringMap{}
	e4 := UnmarshalStringMap{map[string]string{"b": "1"}}
	unmarshal(t, y, &s4, &e4)

	f64String := strconv.FormatFloat(math.MaxFloat64, 'g', -1, 64)
	y = []byte("a: " + f64String)
	s5 := UnmarshalFloat64{}
	e5 := UnmarshalFloat64{A: math.MaxFloat64}
	unmarshal(t, y, &s5, &e5)
}

func unmarshal(t *testing.T, y []byte, s, e interface{}) {
	err := Unmarshal(y, s)
	if err != nil {
		t.Errorf("error unmarshaling YAML: %v", err)
	}

	if !reflect.DeepEqual(s, e) {
		t.Errorf("unmarshal YAML was unsuccessful, expected: %+#v, got: %+#v",
			e, s)
	}
}

func TestUnmarshalStrict(t *testing.T) {
	y := []byte("a: 1\nb: 2")
	s1 := UnmarshalString{}
	f1 := "error parsing document: field in document is not recognized or is in the wrong location: b"
	unmarshalStrict(t, y, &s1, f1)

	y = []byte("a: 1\nb: 2\nc: foo\nd: 123\ne: baz\nff: bar\nqq: baz")
	s1 = UnmarshalString{}
	f1 = "error parsing document: fields in document are not recognized or are in the wrong location: b, c, d, e (+ 2 more)"
	unmarshalStrict(t, y, &s1, f1)

	y = []byte("a: 1\nb: 2\nc: foo\nd: 123")
	s1 = UnmarshalString{}
	f1 = "error parsing document: fields in document are not recognized or are in the wrong location: b, c, d"
	unmarshalStrict(t, y, &s1, f1)

	y = []byte("a: 1")
	s1 = UnmarshalString{}
	f1 = ""
	unmarshalStrict(t, y, &s1, f1)

}

func unmarshalStrict(t *testing.T, y []byte, s interface{}, errmsg string) {
	err := UnmarshalStrict(y, s)
	if errmsg == "" && err != nil {
		t.Errorf("unexpected error unmarshaling YAML: %v", err)
	}

	if errmsg != "" {
		if err == nil {
			t.Errorf("expected error %s", errmsg)
		} else {
			actualmsg := fmt.Sprint(err)
			if actualmsg != errmsg {
				t.Errorf("expect %s, got %s", errmsg, actualmsg)
			}
		}
	}
}

type Case struct {
	input  string
	output string
	// By default we test that reversing the output == input. But if there is a
	// difference in the reversed output, you can optionally specify it here.
	reverse *string
}

type RunType int

const (
	RunTypeJSONToYAML RunType = iota
	RunTypeYAMLToJSON
)

func TestJSONToYAML(t *testing.T) {
	cases := []Case{
		{
			`{"t":"a"}`,
			"t: a\n",
			nil,
		}, {
			`{"t":null}`,
			"t: null\n",
			nil,
		},
	}

	runCases(t, RunTypeJSONToYAML, cases)
}

func TestYAMLToJSON(t *testing.T) {
	cases := []Case{
		{
			"t: a\n",
			`{"t":"a"}`,
			nil,
		}, {
			"t: \n",
			`{"t":null}`,
			strPtr("t: null\n"),
		}, {
			"t: null\n",
			`{"t":null}`,
			nil,
		}, {
			"1: a\n",
			`{"1":"a"}`,
			strPtr("\"1\": a\n"),
		}, {
			"1000000000000000000000000000000000000: a\n",
			`{"1e+36":"a"}`,
			strPtr("\"1e+36\": a\n"),
		}, {
			"1e+36: a\n",
			`{"1e+36":"a"}`,
			strPtr("\"1e+36\": a\n"),
		}, {
			"\"1e+36\": a\n",
			`{"1e+36":"a"}`,
			nil,
		}, {
			"\"1.2\": a\n",
			`{"1.2":"a"}`,
			nil,
		}, {
			"- t: a\n",
			`[{"t":"a"}]`,
			nil,
		}, {
			"- t: a\n" +
				"- t:\n" +
				"    b: 1\n" +
				"    c: 2\n",
			`[{"t":"a"},{"t":{"b":1,"c":2}}]`,
			nil,
		}, {
			`[{t: a}, {t: {b: 1, c: 2}}]`,
			`[{"t":"a"},{"t":{"b":1,"c":2}}]`,
			strPtr("- t: a\n" +
				"- t:\n" +
				"    b: 1\n" +
				"    c: 2\n"),
		}, {
			"- t: \n",
			`[{"t":null}]`,
			strPtr("- t: null\n"),
		}, {
			"- t: null\n",
			`[{"t":null}]`,
			nil,
		},
	}

	// Cases that should produce errors.
	_ = []Case{
		{
			"~: a",
			`{"null":"a"}`,
			nil,
		}, {
			"a: !!binary gIGC\n",
			"{\"a\":\"\x80\x81\x82\"}",
			nil,
		},
	}

	runCases(t, RunTypeYAMLToJSON, cases)
}

func runCases(t *testing.T, runType RunType, cases []Case) {
	var f func([]byte) ([]byte, error)
	var invF func([]byte) ([]byte, error)
	var msg string
	var invMsg string
	if runType == RunTypeJSONToYAML {
		f = JSONToYAML
		invF = YAMLToJSON
		msg = "JSON to YAML"
		invMsg = "YAML back to JSON"
	} else {
		f = YAMLToJSON
		invF = JSONToYAML
		msg = "YAML to JSON"
		invMsg = "JSON back to YAML"
	}

	for _, c := range cases {
		// Convert the string.
		t.Logf("converting %s\n", c.input)
		output, err := f([]byte(c.input))
		if err != nil {
			t.Errorf("Failed to convert %s, input: `%s`, err: %v", msg, c.input, err)
		}

		// Check it against the expected output.
		if string(output) != c.output {
			t.Errorf("Failed to convert %s, input: `%s`, expected `%s`, got `%s`",
				msg, c.input, c.output, string(output))
		}

		// Set the string that we will compare the reversed output to.
		reverse := c.input
		// If a special reverse string was specified, use that instead.
		if c.reverse != nil {
			reverse = *c.reverse
		}

		// Reverse the output.
		input, err := invF(output)
		if err != nil {
			t.Errorf("Failed to convert %s, input: `%s`, err: %v", invMsg, string(output), err)
		}

		// Check the reverse is equal to the input (or to *c.reverse).
		if string(input) != reverse {
			t.Errorf("Failed to convert %s, input: `%s`, expected `%s`, got `%s`",
				invMsg, string(output), reverse, string(input))
		}
	}

}

// To be able to easily fill in the *Case.reverse string above.
func strPtr(s string) *string {
	return &s
}
