/*
 * Copyright 2019 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cli

import (
	"fmt"
	"os"

	"github.com/CovenantSQL/CookieScanner/cmd"
	"github.com/CovenantSQL/CookieScanner/parser"
	"github.com/pkg/errors"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	port       int
	headless   bool
	outputJSON bool
	outputHTML string
	outputPDF  string
	site       string
)

func RegisterCommand(app *kingpin.Application, opts *cmd.CommonOptions) {
	c := app.Command("cli", "generate report for a single website")
	c.Flag("headless", "run chrome in headless mode").BoolVar(&headless)
	c.Flag("port", "chrome remote debugger listen port").Default("9222").IntVar(&port)
	c.Flag("json", "print report as json").BoolVar(&outputJSON)
	c.Flag("html", "save report as html").StringVar(&outputHTML)
	c.Flag("pdf", "save report as pdf").StringVar(&outputPDF)
	c.Arg("site", "site url").Required().StringVar(&site)
	c.Action(func(context *kingpin.ParseContext) error {
		return handler(opts)
	})
}

func handler(opts *cmd.CommonOptions) (err error) {
	if !outputJSON && outputHTML == "" && outputPDF == "" {
		outputJSON = true
	}

	if outputPDF != "" {
		headless = true
	}

	t := parser.NewTask(&parser.TaskConfig{
		Timeout:           opts.Timeout,
		WaitAfterPageLoad: opts.WaitAfterPageLoad,
		Verbose:           opts.Verbose,
		ChromeApp:         opts.ChromeApp,
		DebuggerPort:      port,
		Headless:          headless,
		Classifier:        opts.ClassifierHandler,
	})

	if err = t.Start(); err != nil {
		err = errors.Wrapf(err, "start debugger failed")
		return
	}

	defer t.Cleanup()

	if err = t.Parse(site); err != nil {
		err = errors.Wrapf(err, "get site cookie info failed")
		return
	}

	if outputJSON {
		if jsonData, err := t.OutputJSON(true); err == nil {
			fmt.Println(jsonData)
		} else {
			err = errors.Wrapf(err, "generate json report failed")
		}

		return
	}

	if outputHTML != "" {
		var htmlData string
		if htmlData, err = t.OutputHTML(); err != nil {
			err = errors.Wrapf(err, "generate html report failed")
		} else {
			var f *os.File
			if f, err = os.OpenFile(outputHTML, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755); err != nil {
				err = errors.Wrap(err, "write html report failed")
				return
			}
			_, _ = f.WriteString(htmlData)
			_ = f.Sync()
			_ = f.Close()
		}
		return
	}

	if outputPDF != "" {
		err = errors.Wrapf(t.OutputPDFToFile(outputPDF), "generate pdf report failed")
		return
	}

	return
}
