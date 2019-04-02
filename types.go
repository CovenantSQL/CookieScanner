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
	"net/http"
	"sort"
	"sync"

	"github.com/raff/godet"
)

type record struct {
	isRequest bool
	reqID     string
	reqSeq    float64
	params    godet.Params
}

type recordCollector struct {
	l       sync.Mutex
	records map[string][]*record
}

func newRecordCollector() *recordCollector {
	return &recordCollector{
		records: map[string][]*record{},
	}
}

func (rc *recordCollector) addRecord(r *record) {
	if r.reqID == "" {
		return
	}

	rc.l.Lock()
	defer rc.l.Unlock()

	rc.records[r.reqID] = append(rc.records[r.reqID], r)

	// sort
	sort.SliceStable(rc.records[r.reqID], func(i, j int) bool {
		return rc.records[r.reqID][i].reqSeq < rc.records[r.reqID][j].reqSeq
	})
}

func (rc *recordCollector) get() (r map[string][]*record) {
	rc.l.Lock()
	defer rc.l.Unlock()

	r = make(map[string][]*record, len(rc.records))

	for k, v := range rc.records {
		r[k] = append([]*record(nil), v...)
	}

	return
}

func (rc *recordCollector) addRequest(p godet.Params) {
	rc.addRecord(&record{
		isRequest: true,
		params:    p,
		reqID:     p.String("requestId"),
		reqSeq:    p["timestamp"].(float64),
	})
}

func (rc *recordCollector) addResponse(p godet.Params) {
	rc.addRecord(&record{
		isRequest: false,
		params:    p,
		reqID:     p.String("requestId"),
		reqSeq:    p["timestamp"].(float64),
	})
}

type outputRecord struct {
	url         string
	reqSeq      float64
	statusCode  int
	usedCookies []*http.Cookie
	setCookies  []*http.Cookie
	mimeType    string
	remoteAddr  string
	initiator   string
	source      string
	lineNo      int
}
