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
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gobs/args"
	"github.com/gobs/pretty"
	"github.com/raff/godet"
)

var (
	cmd               string
	headless          bool
	port              string
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
	var chromeapp string

	switch runtime.GOOS {
	case "darwin":
		for _, c := range []string{
			"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		} {
			// MacOS apps are actually folders
			if _, err := exec.LookPath(c); err == nil {
				chromeapp = fmt.Sprintf("%q", c)
				break
			}
		}

	case "linux":
		for _, c := range []string{
			"headless_shell",
			"chromium",
			"google-chrome-beta",
			"google-chrome-unstable",
			"google-chrome-stable"} {
			if _, err := exec.LookPath(c); err == nil {
				chromeapp = c
				break
			}
		}

	case "windows":
	}

	if chromeapp != "" {
		if chromeapp == "headless_shell" {
			chromeapp += " --no-sandbox"
		} else {
			chromeapp += " --headless"
		}

		chromeapp += " --remote-debugging-port=9222 --no-default-browser-check --no-first-run --hide-scrollbars --bwsi --disable-gpu"

		if dir, err := ioutil.TempDir("", "gdpr_cookie"); err == nil {
			defer func() {
				_ = os.RemoveAll(dir)
			}()
			chromeapp += " --user-data-dir="
			chromeapp += dir
		}

		chromeapp += " about:blank"
	}

	flag.StringVar(&cmd, "cmd", chromeapp, "command to execute to start the browser")
	flag.BoolVar(&headless, "headless", false, "headless mode")
	flag.StringVar(&port, "port", "localhost:9222", "Chrome remote debugger port")
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

	if cmd != "" {
		if !headless {
			cmd = strings.ReplaceAll(cmd, "--headless", "")
		}

		log.Println("start chrome debugger process")

		parts := args.GetArgs(cmd)
		cmdObj := exec.Command(parts[0], parts[1:]...)
		if err := cmdObj.Start(); err != nil {
			log.Fatalf("start chrome debugger failed: %v\n", err)
			return
		}

		defer func() {
			_ = cmdObj.Process.Signal(syscall.SIGTERM)
			_ = cmdObj.Wait()
		}()
	}

	var (
		remote *godet.RemoteDebugger
		err    error
	)

	for i := 0; i < 10; i++ {
		if i > 0 {
			time.Sleep(500 * time.Millisecond)
		}

		remote, err = godet.Connect(port, verbose)
		if err == nil {
			break
		}

		log.Println("connect", err)
	}

	if err != nil {
		log.Printf("cannot connect to browser: %v\n", err)
		return
	}

	defer func() {
		remote.CloseBrowser()
		_ = remote.Close()
	}()

	v, err := remote.Version()
	if err != nil {
		log.Printf("cannot get debugger version: %v\n", err)
		return
	}

	if version {
		pretty.PrettyPrint(v)
		return
	} else {
		log.Printf("connected to %s with protocol version %s", v.Browser, v.ProtocolVersion)
	}

	if err = handleRequest(remote); err != nil {
		log.Printf("get cookie data failed: %v", err)
		return
	}
}
