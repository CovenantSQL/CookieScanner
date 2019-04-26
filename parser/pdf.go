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
	"bytes"
	"html/template"
	"path/filepath"
	"reflect"
	"time"

	"github.com/raff/godet"
)

var (
	reportTemplate = template.New("report_template")
)

func init() {
	template.Must(reportTemplate.Funcs(template.FuncMap{
		"isEven": func(v int) bool {
			return v%2 == 0
		},
		"len": func(v interface{}) int {
			rv := reflect.ValueOf(v)
			switch rv.Kind() {
			case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
				return rv.Len()
			default:
				return 0
			}
		},
	}).Parse(`<!DOCTYPE html>
<meta charset="UTF-8">
<html>
<head>
    <title>Cookie scan report</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap@4.3.1/dist/css/bootstrap.min.css"/>
</head>
<body>
<div class="container mt-5">
    <section>
        <a class="text-right d-block mb-3" href="https://gdprexpert.io">
            <img class="image w-25" src="https://cdn.jsdelivr.net/gh/CovenantLabs/assets/gdprexpert/logo_io.png"/>
        </a>
    </section>
    <section class="mb-5">
        <h2 class="mb-3">Cookie scan report</h2>
        <div class="row">
            <div class="col-6">
                <ul class="list-unstyled">
                    <li><span class="mr-1">Scan date:</span>{{.ScanTime}}</li>
                    <li><span class="mr-1">Scan URL:</span>{{.ScanURL}}</li>
                    <li><span class="mr-1">Cookies (in total):</span>{{.CookieCount}}</li>
                </ul>
            </div>
            <div class="col-6">
                {{if ne .ScreenShotImage ""}}
                    <img src="data:image/png;base64,{{.ScreenShotImage}}" class="img-fluid img-thumbnail"/>
                {{end}}
            </div>
        </div>
    </section>
    {{range $record := .Records}}
        <section>
            <h3>{{if ne $record.Category ""}}{{$record.Category}}{{else}}Unclassified{{end}}
                &nbsp;({{len $record.Cookies}})</h3>
            <p class="border-top pt-3">
                <!--
                {{if ne $record.Description ""}}
                    {{$record.Description}}
                {{else}}
                    We don’t have enough information about this cookie or the website hosting it to be able to assign it to a category at this time.
                {{end}}
                -->
            </p>
            <table class="table border-top-0">
                <thead>
                <tr class="text-uppercase">
                    <th scope="col" class="border-top-0">cookie name</th>
                    <th scope="col" class="border-top-0">provider</th>
                    <th scope="col" class="border-top-0">expiry</th>
                </tr>
                </thead>
                <tbody>
                {{range $index, $cookie := $record.Cookies}}
                    <tr class="{{if isEven $index}}bg-light{{end}}">
                        <td><strong>{{$cookie.Name}}</strong></td>
                        <td>{{$cookie.Domain}}</td>
                        <td>{{$cookie.Expiry}}</td>
                    </tr>
                    <tr class="{{if isEven $index}}bg-light{{end}}">
                        <td colspan="3" class="border-top-0 pt-0">
                            <ul class="list-unstyled">
                                <li>
                                    <small><strong class="mr-1">First found:</strong>{{$cookie.URL}}</small>
                                </li>
                                <li>
                                    <small><strong class="mr-1">Initiator:</strong>{{$cookie.Initiator}}</small>
                                </li>
                                <li>
                                    <small><strong class="mr-1">Source:</strong>
                                        {{if ne $cookie.Source "" }}{{$cookie.Source}}{{if gt $cookie.LineNo 0}}: {{$cookie.LineNo}}{{end}}{{else}}-{{end}}
                                    </small>
                                </li>
                                <li>
                                    <small><strong class="mr-1">Server&nbsp;Address:</strong>{{$cookie.RemoteAddr}}
                                    </small>
                                </li>
                                <li>
                                    <small>
                                        <strong class="mr-1">Mime&nbsp;Type:</strong>{{if ne $cookie.MimeType ""}}{{$cookie.MimeType}}{{else}}-{{end}}
                                    </small>
                                </li>
                                <li>
                                    <small>
                                        <strong class="mr-1">Used&nbsp;Requests:</strong>{{$cookie.UsedRequests}}
                                    </small>
                                </li>
                                <li>
                                    <small>
                                        <strong class="mr-1">HttpOnly:</strong>{{if $cookie.HttpOnly}}yes{{else}}no{{end}}
                                    </small>
                                </li>
                                <li>
                                    <small><strong class="mr-1">Description:</strong>{{$cookie.Description}}</small>
                                </li>
                            </ul>
                        </td>
                    </tr>
                {{end}}
                </tbody>
            </table>
        </section>
    {{end}}
</div>
</body>
</html>`))
}

func outputAsHTML(data *reportData) (str string, err error) {
	buf := new(bytes.Buffer)
	err = reportTemplate.Execute(buf, data)
	str = buf.String()
	return
}

func outputAsPDF(remote *godet.RemoteDebugger, htmlFile string) (pdfBytes []byte, err error) {
	var tab *godet.Tab

	htmlFile, _ = filepath.Abs(htmlFile)
	fileLink := "file://" + htmlFile

	if tab, err = remote.NewTab(fileLink); err != nil {
		return
	}
	if err = remote.ActivateTab(tab); err != nil {
		return
	}

	// wait for page to load
	time.Sleep(time.Second)

	return remote.PrintToPDF(godet.PortraitMode())
}
