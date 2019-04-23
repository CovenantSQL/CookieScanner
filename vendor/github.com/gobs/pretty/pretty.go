/*
 Pretty-print Go data structures
*/
package pretty

import (
	"bytes"
	"fmt"
	"io"
	"os"
	r "reflect"
	"strconv"
	"strings"
)

const (
	DEFAULT_INDENT = "  "
	DEFAULT_NIL    = "nil"
)

// The context for printing
type Pretty struct {
	// indent string
	Indent string
	// output recipient
	Out io.Writer
	// string for nil
	NilString string
	// compact empty array and struct
	Compact bool
	// Maximum nesting level
	MaxLevel int
}

// pretty print the input value (to stdout)
func PrettyPrint(i interface{}) {
	PrettyPrintTo(os.Stdout, i, true)
}

// pretty print the input value (to a string)
func PrettyFormat(i interface{}) string {
	var out bytes.Buffer
	PrettyPrintTo(&out, i, false)
	return out.String()
}

// pretty print the input value (to specified writer)
func PrettyPrintTo(out io.Writer, i interface{}, nl bool) {
	p := &Pretty{Indent: DEFAULT_INDENT, Out: out, NilString: DEFAULT_NIL}
	if nl {
		p.Println(i)
	} else {
		p.Print(i)
	}
}

// pretty print the input value (no newline)
func (p *Pretty) Print(i interface{}) {
	p.PrintValue(r.ValueOf(i), 0)
}

// pretty print the input value (newline)
func (p *Pretty) Println(i interface{}) {
	p.PrintValue(r.ValueOf(i), 0)
	io.WriteString(p.Out, "\n")
}

// recursively print the input value
func (p *Pretty) PrintValue(val r.Value, level int) {
	if !val.IsValid() {
		io.WriteString(p.Out, p.NilString)
		return
	}

	cur := strings.Repeat(p.Indent, level)
	next := strings.Repeat(p.Indent, level+1)

	nl := "\n"
	if len(p.Indent) == 0 {
		nl = " "
	}

	if p.MaxLevel > 0 && level >= p.MaxLevel {
		io.WriteString(p.Out, val.String())
		return
	}

	switch val.Kind() {
	case r.Int, r.Int8, r.Int16, r.Int32, r.Int64:
		io.WriteString(p.Out, strconv.FormatInt(val.Int(), 10))

	case r.Uint, r.Uint8, r.Uint16, r.Uint32, r.Uint64:
		io.WriteString(p.Out, strconv.FormatUint(val.Uint(), 10))

	case r.Float32, r.Float64:
		io.WriteString(p.Out, strconv.FormatFloat(val.Float(), 'f', -1, 64))

	case r.String:
		io.WriteString(p.Out, strconv.Quote(val.String()))

	case r.Bool:
		io.WriteString(p.Out, strconv.FormatBool(val.Bool()))

	case r.Map:
		l := val.Len()

		io.WriteString(p.Out, "{"+nl)
		for i, k := range val.MapKeys() {
			io.WriteString(p.Out, next)
			io.WriteString(p.Out, strconv.Quote(k.String()))
			io.WriteString(p.Out, ": ")
			p.PrintValue(val.MapIndex(k), level+1)
			if i < l-1 {
				io.WriteString(p.Out, ","+nl)
			} else {
				io.WriteString(p.Out, nl)
			}
		}
		io.WriteString(p.Out, cur)
		io.WriteString(p.Out, "}")

	case r.Array, r.Slice:
		l := val.Len()

		if p.Compact && l == 0 {
			io.WriteString(p.Out, "[]")
		} else {
			io.WriteString(p.Out, "["+nl)
			for i := 0; i < l; i++ {
				io.WriteString(p.Out, next)
				p.PrintValue(val.Index(i), level+1)
				if i < l-1 {
					io.WriteString(p.Out, ","+nl)
				} else {
					io.WriteString(p.Out, nl)
				}
			}
			io.WriteString(p.Out, cur)
			io.WriteString(p.Out, "]")
		}

	case r.Interface, r.Ptr:
		p.PrintValue(val.Elem(), level)

	case r.Struct:
		if val.CanInterface() {
			i := val.Interface()
			if i, ok := i.(fmt.Stringer); ok {
				io.WriteString(p.Out, i.String())
			} else {
				l := val.NumField()

				sOpen := "struct {"

				if p.Compact {
					sOpen = "{"
				}

				if p.Compact && l == 0 {
					io.WriteString(p.Out, "{}")
				} else {
					io.WriteString(p.Out, sOpen+nl)
					for i := 0; i < l; i++ {
						io.WriteString(p.Out, next)
						io.WriteString(p.Out, val.Type().Field(i).Name)
						io.WriteString(p.Out, ": ")
						p.PrintValue(val.Field(i), level+1)
						if i < l-1 {
							io.WriteString(p.Out, ","+nl)
						} else {
							io.WriteString(p.Out, nl)
						}
					}
					io.WriteString(p.Out, cur)
					io.WriteString(p.Out, "}")
				}
			}
		} else {
			io.WriteString(p.Out, "protected")
		}

	default:
		io.WriteString(p.Out, "unsupported:")
		io.WriteString(p.Out, val.String())
	}
}
