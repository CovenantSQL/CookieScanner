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

package parser

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gobs/args"
	"github.com/pkg/errors"
	"github.com/raff/godet"
)

func (t *Task) Start() (err error) {
	if t.cfg.ChromeApp == "" {
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

			chromeapp += fmt.Sprintf(" --remote-debugging-port=%d --no-default-browser-check --no-first-run --hide-scrollbars --bwsi --disable-gpu",
				t.cfg.DebuggerPort)

			if dir, err := ioutil.TempDir("", "gdpr_cookie"); err == nil {
				t.userDir = dir
				chromeapp += " --user-data-dir="
				chromeapp += dir
			}

			chromeapp += " about:blank"
		}

		t.cfg.ChromeApp = chromeapp
	}

	// start debugger
	if !t.cfg.Headless {
		t.cfg.ChromeApp = strings.ReplaceAll(t.cfg.ChromeApp, "--headless", "")
	}

	parts := args.GetArgs(t.cfg.ChromeApp)
	cmd := exec.Command(parts[0], parts[1:]...)
	if err = cmd.Start(); err != nil {
		return
	}

	t.debugger = cmd

	// connect debugger
	for i := 0; i < 10; i++ {
		if i > 0 {
			time.Sleep(500 * time.Millisecond)
		}

		t.remote, err = godet.Connect(fmt.Sprintf("localhost:%d", t.cfg.DebuggerPort), t.cfg.Verbose)
		if err == nil {
			break
		}

		log.Printf("connect to debugger failed: %v", err)
	}

	return
}

func (t *Task) Cleanup() {
	if t.userDir != "" {
		_ = os.RemoveAll(t.userDir)
	}
	if t.remote != nil {
		t.remote.CloseBrowser()
		_ = t.remote.Close()
	}
	if t.debugger != nil {
		_ = t.debugger.Process.Signal(syscall.SIGTERM)
		_ = t.debugger.Wait()
	}
}

func (t *Task) Version() (*godet.Version, error) {
	return t.remote.Version()
}

func (t *Task) Parse(site string) (err error) {
	siteURL, err := url.Parse(site)
	if err != nil {
		err = errors.Wrap(err, "parse url failed")
		return
	}

	if siteURL.Scheme == "" {
		siteURL.Scheme = "http"
	}

	site = siteURL.String()
	rc := newRecordCollector()

	// request callbacks
	t.remote.CallbackEvent("Network.requestWillBeSent", func(params godet.Params) {
		req := params.Map("request")
		var reqURL string
		if rawReqURL, ok := req["url"]; !ok {
			return
		} else if reqURL, ok = rawReqURL.(string); !ok {
			return
		}
		if strings.HasPrefix(reqURL, "data:") {
			// data uri is ignored
			return
		}

		rc.addRequest(params)
	})

	// response callbacks
	t.remote.CallbackEvent("Network.responseReceived", func(params godet.Params) {
		resp := params.Map("response")
		var respURL string
		if rawRespURL, ok := resp["url"]; !ok {
			return
		} else if respURL, ok = rawRespURL.(string); !ok {
			return
		}
		if strings.HasPrefix(respURL, "data:") {
			// data uri is ignored
			return
		}

		rc.addResponse(params)
	})

	pageWait := make(chan struct{}, 1)

	time.AfterFunc(t.cfg.Timeout, func() {
		log.Println("timeout triggered")
		select {
		case pageWait <- struct{}{}:
		default:
		}
	})

	// page stopped loading event
	t.remote.CallbackEvent("Page.frameStoppedLoading", func(params godet.Params) {
		log.Println("page frame stopped loading")
		go func() {
			time.Sleep(t.cfg.WaitAfterPageLoad)
			select {
			case pageWait <- struct{}{}:
			default:
			}
		}()
	})

	// page load event fired
	t.remote.CallbackEvent("Page.loadEventFired", func(params godet.Params) {
		log.Println("page load fired")
		go func() {
			time.Sleep(t.cfg.WaitAfterPageLoad)
			select {
			case pageWait <- struct{}{}:
			default:
			}
		}()
	})

	// debugger log
	if t.cfg.Verbose {
		t.remote.CallbackEvent("Log.entryAdded", func(params godet.Params) {
			entry := params.Map("entry")
			log.Println("LOG", entry["type"], entry["level"], entry["text"])
		})

		// console log
		t.remote.CallbackEvent("Runtime.consoleAPICalled", func(params godet.Params) {
			l := []interface{}{"CONSOLE", params["type"].(string)}

			for _, a := range params["args"].([]interface{}) {
				arg := a.(map[string]interface{})

				if arg["value"] != nil {
					l = append(l, arg["value"])
				} else if arg["preview"] != nil {
					arg := arg["preview"].(map[string]interface{})

					v := arg["description"].(string) + "{"

					for i, p := range arg["properties"].([]interface{}) {
						if i > 0 {
							v += ", "
						}

						prop := p.(map[string]interface{})
						if prop["name"] != nil {
							v += fmt.Sprintf("%q: ", prop["name"])
						}

						v += fmt.Sprintf("%v", prop["value"])
					}

					v += "}"
					l = append(l, v)
				} else {
					l = append(l, arg["type"].(string))
				}

			}

			log.Println(l...)
		})
	}

	_ = t.remote.RuntimeEvents(true)
	_ = t.remote.NetworkEvents(true)
	_ = t.remote.PageEvents(true)
	_ = t.remote.DOMEvents(true)
	_ = t.remote.LogEvents(true)
	_ = t.remote.EmulationEvents(true)
	//_ = remote.EnableRequestInterception(true)

	_, err = t.remote.Navigate(site)

	if err != nil {
		err = errors.Wrap(err, "send request failed")
		return
	}

	<-pageWait

	// parse response
	var (
		cookieCount   int
		reportRecords []*reportRecord
	)
	cookieCount, reportRecords, err = t.parseResponse(rc)
	if err != nil {
		return
	}

	// assemble with other page info
	t.reportData = &reportData{
		ScanTime:    t.startTime,
		ScanURL:     site,
		CookieCount: cookieCount,
		Records:     reportRecords,
	}

	return
}

func (t *Task) OutputJSON(prefix string, indent string) (str string, err error) {
	jsonBlob, err := json.MarshalIndent(t.reportData, prefix, indent)
	str = string(jsonBlob)
	return
}

func (t *Task) OutputHTML(filename string) (err error) {
	return outputAsHTML(t.reportData, filename)
}

func (t *Task) OutputPDF(filename string) (err error) {
	var f *os.File
	if f, err = ioutil.TempFile("", "gdpr_cookie*.html"); err != nil {
		return
	}

	_ = f.Close()
	tempHTML := f.Name()
	defer func() {
		_ = os.Remove(tempHTML)
	}()

	if err = outputAsHTML(t.reportData, tempHTML); err != nil {
		return
	}

	err = outputAsPDF(t.remote, tempHTML, filename)

	return
}
