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
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/raff/godet"
)

func handleRequest(remote *godet.RemoteDebugger) (err error) {
	if flag.NArg() == 0 {
		// no sites
		return
	}

	// normalized url
	site := flag.Args()[0]
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
	remote.CallbackEvent("Network.requestWillBeSent", func(params godet.Params) {
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
	remote.CallbackEvent("Network.responseReceived", func(params godet.Params) {
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

	time.AfterFunc(timeout, func() {
		log.Println("timeout triggered")
		select {
		case pageWait <- struct{}{}:
		default:
		}
	})

	// page stopped loading event
	remote.CallbackEvent("Page.frameStoppedLoading", func(params godet.Params) {
		log.Println("page frame stopped loading")
		go func() {
			time.Sleep(waitAfterPageLoad)
			select {
			case pageWait <- struct{}{}:
			default:
			}
		}()
	})

	// page load event fired
	remote.CallbackEvent("Page.loadEventFired", func(params godet.Params) {
		log.Println("page load fired")
		go func() {
			time.Sleep(waitAfterPageLoad)
			select {
			case pageWait <- struct{}{}:
			default:
			}
		}()
	})

	// debugger log
	if verbose {
		remote.CallbackEvent("Log.entryAdded", func(params godet.Params) {
			entry := params.Map("entry")
			log.Println("LOG", entry["type"], entry["level"], entry["text"])
		})

		// console log
		remote.CallbackEvent("Runtime.consoleAPICalled", func(params godet.Params) {
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

	_ = remote.RuntimeEvents(true)
	_ = remote.NetworkEvents(true)
	_ = remote.PageEvents(true)
	_ = remote.DOMEvents(true)
	_ = remote.LogEvents(true)
	_ = remote.EmulationEvents(true)
	//_ = remote.EnableRequestInterception(true)

	_, err = remote.Navigate(site)

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
	cookieCount, reportRecords, err = parseResponse(rc)
	if err != nil {
		return
	}

	// assemble with other page info
	rData := &reportData{
		ScanTime:    startTime,
		ScanURL:     site,
		CookieCount: cookieCount,
		Records:     reportRecords,
	}

	if outputJSON {
		jsonBlob, _ := json.MarshalIndent(rData, "", "  ")
		fmt.Println(string(jsonBlob))
	}

	if outputHTML != "" {
		if err = outputAsHTML(rData, outputHTML); err != nil {
			return
		}
	}

	if outputPDF != "" {
		var f *os.File
		if f, err = ioutil.TempFile("", "gdpr_cookie*.html"); err != nil {
			return
		}

		_ = f.Close()
		tempHTML := f.Name()
		defer func() {
			_ = os.Remove(tempHTML)
		}()

		if err = outputAsHTML(rData, tempHTML); err != nil {
			return
		}

		if err = outputAsPDF(remote, tempHTML, outputPDF); err != nil {
			return
		}
	}

	return
}
