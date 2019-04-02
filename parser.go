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
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jmoiron/jsonq"
)

func parseHeaders(isRequest bool, headers ...map[string]interface{}) []*http.Cookie {
	if isRequest {
		fakeReq := &http.Request{Header: http.Header{}}

		for _, hs := range headers {
			for k, vs := range hs {
				for _, v := range strings.Split(vs.(string), "\n") {
					fakeReq.Header.Add(k, v)
				}
			}
		}

		return fakeReq.Cookies()
	} else {
		fakeResp := &http.Response{Header: http.Header{}}

		for _, hs := range headers {
			for k, vs := range hs {
				for _, v := range strings.Split(vs.(string), "\n") {
					fakeResp.Header.Add(k, v)
				}
			}
		}

		cookies := fakeResp.Cookies()

		// re-parse expires field
		for _, c := range cookies {
			if c.RawExpires != "" {
				for _, f := range []string{
					"Mon, 02-Jan-06 15:04:05 MST",
					"Mon, 02 Jan 06 15:04:05 MST",
				} {
					etime, err := time.Parse(f, c.RawExpires)
					if err == nil {
						c.Expires = etime.UTC()
						break
					}
				}
			}
		}

		return cookies
	}
}

func parseResponse(rc *recordCollector) (cookieCount int, resultData []*reportRecord, err error) {
	resp := rc.get()
	var outputs []*outputRecord

	for _, records := range resp {
		var (
			lastHeaders map[string]interface{}
			lastRecord  *record
			output      = new(outputRecord)
		)

		for _, r := range records {
			q := jsonq.NewQuery(map[string]interface{}(r.params))

			if r.isRequest {
				if lastRecord != nil && lastRecord.isRequest {
					// this request should contains redirectResponse
					if redirectResponse, _ := q.Object("redirectResponse"); redirectResponse != nil {
						output.statusCode, _ = q.Int("redirectResponse", "status")
						headers, _ := q.Object("redirectResponse", "headers")
						output.setCookies = parseHeaders(false, headers)

						// set cookie default domain using current document domain
						if len(output.setCookies) > 0 {
							if reqURLObj, _ := url.Parse(output.url); reqURLObj != nil {
								for _, c := range output.setCookies {
									if c.Domain == "" {
										c.Domain = reqURLObj.Host
									}
								}
							}
						}

						output.mimeType, _ = q.String("redirectResponse", "mimeType")
						output.remoteAddr, _ = q.String("redirectResponse", "remoteIPAddress")

						// merge request headers
						requestHeaders, _ := q.Object("redirectResponse", "requestHeaders")
						output.usedCookies = parseHeaders(true, lastHeaders, requestHeaders)

						// add this output
						if len(output.usedCookies) > 0 || len(output.setCookies) > 0 {
							outputs = append(outputs, output)
						}

						output = new(outputRecord)
					}
				}

				output.url, _ = q.String("request", "url")
				output.reqSeq = r.reqSeq
				headers, _ := q.Object("request", "headers")
				output.usedCookies = parseHeaders(true, headers)
				output.initiator, _ = q.String("initiator", "type")
				output.source, _ = q.String("initiator", "url")
				output.lineNo, _ = q.Int("initiator", "lineNumber")
				lastHeaders = headers
			} else {
				output.statusCode, _ = q.Int("response", "status")
				headers, _ := q.Object("response", "headers")
				output.setCookies = parseHeaders(false, headers)

				// set cookie default domain using current document domain
				if len(output.setCookies) > 0 {
					if reqURLObj, _ := url.Parse(output.url); reqURLObj != nil {
						for _, c := range output.setCookies {
							if c.Domain == "" {
								c.Domain = reqURLObj.Host
							}
						}
					}
				}

				output.mimeType, _ = q.String("response", "mimeType")
				output.remoteAddr, _ = q.String("response", "remoteIPAddress")

				// parse request headers
				requestHeaders, _ := q.Object("response", "requestHeaders")
				output.usedCookies = parseHeaders(true, lastHeaders, requestHeaders)

				// add this output
				if len(output.usedCookies) > 0 || len(output.setCookies) > 0 {
					outputs = append(outputs, output)
				}

				output = new(outputRecord)
				lastHeaders = nil
			}

			lastRecord = r
		}
	}

	var (
		cookieUsedCount = map[string]int{}
		cookieSeqMap    = map[string]int{}
		reportRecords   = map[int]*reportRecord{}
	)

	for idx, output := range outputs {
		for _, c := range output.usedCookies {
			cookieUsedCount[c.Name]++
		}
		for _, c := range output.setCookies {
			if i, ok := cookieSeqMap[c.Name]; !ok || outputs[i].reqSeq > output.reqSeq {
				cookieSeqMap[c.Name] = idx
			}
		}
	}

	cookieCount = len(cookieSeqMap)

	for c, idx := range cookieSeqMap {
		for _, cookie := range outputs[idx].setCookies {
			if cookie.Name == c {
				var (
					r  *reportRecord
					ok bool
				)
				if r, ok = reportRecords[idx]; !ok {
					reportRecords[idx] = &reportRecord{
						URL:        outputs[idx].url,
						RemoteAddr: outputs[idx].remoteAddr,
						Status:     outputs[idx].statusCode,
						MimeType:   outputs[idx].mimeType,
						Initiator:  outputs[idx].initiator,
						Source:     outputs[idx].source,
						LineNo:     outputs[idx].lineNo,
					}

					r = reportRecords[idx]
				}

				r.Cookies = append(r.Cookies, &reportCookieRecord{
					Name:    cookie.Name,
					Path:    cookie.Path,
					Domain:  cookie.Domain,
					Expires: cookie.Expires,
					Expiry: func(expiry time.Time, maxAge int) string {
						if maxAge > 0 {
							return estimatedDuration(time.Second * time.Duration(maxAge))
						}

						return estimatedDuration(expiry.Sub(startTime))
					}(cookie.Expires, cookie.MaxAge),
					MaxAge:       cookie.MaxAge,
					Secure:       cookie.Secure,
					HttpOnly:     cookie.HttpOnly,
					UsedRequests: cookieUsedCount[c],
				})

				break
			}
		}
	}

	for _, r := range reportRecords {
		resultData = append(resultData, r)
	}

	return
}

func estimatedDuration(d time.Duration) string {
	if d >= 365*24*time.Hour {
		return fmt.Sprintf("%.1f year", math.Ceil(float64(d)/float64(365*24*time.Hour)))
	} else if d >= 30*24*time.Hour {
		return fmt.Sprintf("%.1f month", math.Ceil(float64(d)/float64(30*24*time.Hour)))
	} else if d >= 24*time.Hour {
		return fmt.Sprintf("%.1f day", math.Ceil(float64(d)/float64(24*time.Hour)))
	} else if d >= time.Hour {
		return fmt.Sprintf("%.1f hour", math.Ceil(float64(d)/float64(time.Hour)))
	} else if d >= time.Minute {
		return fmt.Sprintf("%.1f min", math.Ceil(float64(d)/float64(time.Minute)))
	} else if d >= time.Second {
		return fmt.Sprintf("%.1f sec", math.Ceil(float64(d)/float64(time.Second)))
	} else {
		return "Session"
	}
}
