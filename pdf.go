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
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/raff/godet"
)

var (
	reportTemplate = template.New("report_template")
)

func init() {
	template.Must(reportTemplate.Parse(`
<meta charset="UTF-8">
<html>
<head>
<title>Cookie scan report</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap@3.3.7/dist/css/bootstrap.min.css">
<script src="https://cdn.jsdelivr.net/npm/bootstrap@3.3.7/dist/js/bootstrap.min.js"></script>
</head>
<body>
<div class="container">
	<h1>Cookie scan report</h1>
	<h2>Summary</h2>
	<hr>
	<dl class="row">
		<dt class="col-sm-2">Scan date</dt>
		<dd class="col-sm-10">{{.ScanTime}}</d>
		<dt class="col-sm-2">Scan URL</dt>
		<dd class="col-sm-10">{{.ScanURL}}</dd>
		<dt class="col-sm-2">Cookies (in total)</dt>
		<dd class="col-sm-10">{{.CookieCount}}</dd>
	</dl>
	<h2>Cookies</h2>
	<hr>
	{{range $r := .Records}}
	{{range $c := .Cookies}}
	<h3>{{$c.Name}}</h3>
	<dl class="row">
		<dt class="col-sm-2">Domain</dt>
		<dd class="col-sm-10">{{$c.Domain}}</dd>
		<dt class="col-sm-2">HttpOnly</dt>
		<dd class="col-sm-10">{{$c.HttpOnly}}</dd>
		<dt class="col-sm-2">Expiry</dt>
		<dd class="col-sm-10">{{$c.Expiry}}</dd>
		<dt class="col-sm-2">First found</dt>
		<dd class="col-sm-10">{{$r.URL}}</dd>
		<dt class="col-sm-2">Initiator</dt>
		<dd class="col-sm-10">{{$r.Initiator}}</dd>
		<dt class="col-sm-2">Source</dt>
		<dd class="col-sm-10">{{if ne $r.Source "" }}{{$r.Source}}{{if gt $r.LineNo 0}}: {{$r.LineNo}}{{end}}{{else}}-{{end}}</dd>
		<dt class="col-sm-2">Server Address</dt>
		<dd class="col-sm-10">{{$r.RemoteAddr}}</dd>
		<dt class="col-sm-2">Mime Type</dt>
		<dd class="col-sm-10">{{$r.MimeType}}</dd>
		<dt class="col-sm-2">Used Requests</dt>
		<dd class="col-sm-10">{{$c.UsedRequests}}</dd>
	</dl>
	{{else}}
	<div class="row">
		<p>This website does not use any cookies.</p>
	</div>
	{{end}}
	{{end}}
</div>
</body>
</html>
`))
}

func outputAsHTML(data *reportData, htmlFile string) (err error) {
	f, err := os.OpenFile(htmlFile, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		err = errors.Wrap(err, "write html report failed")
		return
	}
	defer func() {
		_ = f.Close()
	}()
	err = reportTemplate.Execute(f, data)
	return
}

func outputAsPDF(remote *godet.RemoteDebugger, htmlFile string, pdfFile string) (err error) {
	var tab *godet.Tab

	htmlFile, _ = filepath.Abs(htmlFile)
	fileLink := "file://" + htmlFile

	if tab, err = remote.NewTab(fileLink); err != nil {
		return
	}
	if err = remote.ActivateTab(tab); err != nil {
		return
	}
	time.Sleep(5 * time.Second)
	err = remote.SavePDF(pdfFile, 0644, godet.PortraitMode())

	return
}
