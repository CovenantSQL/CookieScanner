package pretty

import (
	"fmt"
	"os"
	"text/tabwriter"
)

// A TabPrinter is an object that allows printing tab-aligned words on multiple lines,
// up to a maximum number per line
type TabPrinter struct {
	w            *tabwriter.Writer
	current, max int
}

// create a TabPrinter
//
// max specifies the maximum number of 'words' per line
func NewTabPrinter(max int) *TabPrinter {
	tp := &TabPrinter{w: new(tabwriter.Writer), max: max}
	tp.w.Init(os.Stdout, 0, 8, 1, '\t', 0)

	return tp
}

// update tab width (minimal space between words)
//
func (tp *TabPrinter) TabWidth(n int) {
	tp.w.Init(os.Stdout, n, 0, 1, ' ', 0)
}

// print a 'word'
//
// when the maximum number of words per lines is reached, this will print the formatted line
func (tp *TabPrinter) Print(arg interface{}) {
	if tp.current > 0 {
		if (tp.current % tp.max) == 0 {
			fmt.Fprintln(tp.w, "")
			tp.w.Flush()
			tp.current = 0
		} else {
			fmt.Fprint(tp.w, "\t")
		}
	}

	tp.current++
	fmt.Fprint(tp.w, arg)
}

// print current line
//
// terminate current line and print - call this after all words have been printed
func (tp *TabPrinter) Println() {
	if tp.current > 0 {
		fmt.Fprintln(tp.w, "")
		tp.w.Flush()
	}

	tp.current = 0
}
