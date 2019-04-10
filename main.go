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

package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/CovenantSQL/CookieTester/parser"
	"github.com/gobs/pretty"
)

var (
	cmd               string
	headless          bool
	port              int
	version           bool
	verbose           bool
	startTime         = time.Now().UTC()
	outputJSON        bool
	outputHTML        string
	outputPDF         string
	waitAfterPageLoad time.Duration
	timeout           time.Duration
)

func init() {
	flag.StringVar(&cmd, "cmd", "", "command to execute to start the browser")
	flag.BoolVar(&headless, "headless", false, "headless mode")
	flag.IntVar(&port, "port", 9222, "Chrome remote debugger port")
	flag.BoolVar(&version, "version", false, "display remote devtools version")
	flag.BoolVar(&verbose, "verbose", false, "verbose logging")
	flag.BoolVar(&outputJSON, "json", false, "output as json")
	flag.StringVar(&outputHTML, "html", "", "output as html (save with specified file name)")
	flag.StringVar(&outputPDF, "pdf", "", "output as pdf (save with specified file name)")
	flag.DurationVar(&waitAfterPageLoad, "wait", 0, "wait duration after page load (capturing ajax/deferred requests)")
	flag.DurationVar(&timeout, "timeout", time.Minute, "timeout for cookie scan")
}

func main() {
	flag.Parse()

	if outputPDF != "" {
		headless = true
	}

	if !version && flag.NArg() == 0 {
		log.Fatalf("site to scan must be provided")
		return
	}

	t := parser.NewTask(&parser.TaskConfig{
		Timeout:           timeout,
		WaitAfterPageLoad: waitAfterPageLoad,
		Verbose:           verbose,
		ChromeApp:         cmd,
		DebuggerPort:      port,
		Headless:          headless,
	})

	if err := t.Start(); err != nil {
		log.Fatalf("start debugger failed: %v", err)
		return
	}

	defer t.Cleanup()

	if version {
		if v, err := t.Version(); err != nil {
			log.Printf("get debugger version failed: %v", err)
		} else {
			pretty.PrettyPrint(v)
		}
		return
	}

	if err := t.Parse(flag.Arg(0)); err != nil {
		log.Printf("get cookie data failed: %v", err)
		return
	}

	if outputJSON {
		if jsonData, err := t.OutputJSON("", "  "); err != nil {
			log.Printf("get json cookie data failed: %v", err)
		} else {
			fmt.Println(jsonData)
		}
		return
	}

	if outputHTML != "" {
		if err := t.OutputHTML(outputHTML); err != nil {
			log.Printf("get html cookie data report failed: %v", err)
		}

		return
	}

	if outputPDF != "" {
		if err := t.OutputPDF(outputPDF); err != nil {
			log.Printf("get pdf cookie data report failed: %v", err)
		}

		return
	}
}
