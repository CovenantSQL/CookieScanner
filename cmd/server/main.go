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

package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/CovenantSQL/CookieTester/cmd"
	"github.com/CovenantSQL/CookieTester/parser"
	"github.com/CovenantSQL/CookieTester/utils"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/raff/godet"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

const (
	argSite   = "site"
	argType   = "type"
	argPretty = "pretty"

	typeJSON = "json"
	typeHTML = "html"
	typePDF  = "pdf"

	contentTypeJSON = "application/json"
	contentTypeHTML = "text/html"
	contentTypePDF  = "application/pdf"
)

var (
	listenAddr  string
	disablePDF  bool
	disableHTML bool
	disableJSON bool
	versionOnce sync.Once
	versionLock sync.RWMutex
	versionInfo *godet.Version

	maxInflightScan int
	inflightSem     *semaphore.Weighted
)

func jsonContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// test if request is post
		if r.Method == http.MethodPost &&
			r.Header.Get("Content-Type") == "application/json" &&
			r.Body != nil {
			// parse json and set to form in request
			var d map[string]interface{}

			if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
				// decode failed
				logrus.WithError(err).Warning("decode request failed")
			} else {
				// fill data to new form
				r.Form = make(url.Values)

				for k, v := range d {
					r.Form.Set(k, fmt.Sprintf("%v", v))
				}

				r.PostForm = r.Form
			}
		}

		next.ServeHTTP(rw, r)
	})
}

func sendResponse(code int, success bool, msg interface{}, data interface{}, rw http.ResponseWriter) {
	msgStr := "ok"
	if msg != nil {
		msgStr = fmt.Sprint(msg)
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(code)
	_ = json.NewEncoder(rw).Encode(map[string]interface{}{
		"status":  msgStr,
		"success": success,
		"data":    data,
	})
}

func RegisterCommand(app *kingpin.Application, opts *cmd.CommonOptions) {
	c := app.Command("server", "start a report generation server")
	c.Flag("listen", "rpc server listen addr").Default(":9223").StringVar(&listenAddr)
	c.Flag("max", "max inflight scan instance").IntVar(&maxInflightScan)
	c.Flag("disable-json", "disable json output support").BoolVar(&disableJSON)
	c.Flag("disable-html", "disable html output support").BoolVar(&disableHTML)
	c.Flag("disable-pdf", "disable pdf output support").BoolVar(&disablePDF)
	c.Action(func(context *kingpin.ParseContext) error {
		return handler(opts)
	})
}

func getVersionFunc(opts *cmd.CommonOptions) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		versionOnce.Do(func() {
			port, err := utils.GetRandomPort()
			if err != nil {
				sendResponse(http.StatusInternalServerError, false, err, nil, rw)
				return
			}

			t := parser.NewTask(&parser.TaskConfig{
				Timeout:           opts.Timeout,
				WaitAfterPageLoad: opts.WaitAfterPageLoad,
				Verbose:           opts.Verbose,
				ChromeApp:         opts.ChromeApp,
				DebuggerPort:      port,
				Headless:          true,
			})

			if err = t.Start(); err != nil {
				sendResponse(http.StatusInternalServerError, false, err, nil, rw)
				return
			}

			defer t.Cleanup()

			versionLock.Lock()
			defer versionLock.Unlock()
			versionInfo, err = t.Version()

			if err != nil {
				sendResponse(http.StatusInternalServerError, false, err, nil, rw)
				return
			}
		})

		versionLock.RLock()
		defer versionLock.RUnlock()

		if versionInfo != nil {
			sendResponse(http.StatusOK, true, nil, versionInfo, rw)
		} else {
			sendResponse(http.StatusInternalServerError, false, "could not get server version", nil, rw)
		}
	}
}

func analyzeFunc(opts *cmd.CommonOptions) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// parse requests
		site := r.FormValue(argSite)
		reportType := r.FormValue(argType)

		if site == "" {
			sendResponse(http.StatusBadRequest, false, "invalid website url", nil, rw)
			return
		}

		switch strings.ToLower(reportType) {
		case "", typeJSON:
			if disableJSON {
				sendResponse(http.StatusBadRequest, false, "json report is disabled", nil, rw)
				return
			}
		case typeHTML:
			if disableHTML {
				sendResponse(http.StatusBadRequest, false, "html report is disabled", nil, rw)
				return
			}
		case typePDF:
			if disablePDF {
				sendResponse(http.StatusBadRequest, false, "pdf report is disabled", nil, rw)
				return
			}
		default:
			sendResponse(http.StatusBadRequest, false, "invalid report type", nil, rw)
			return
		}

		if maxInflightScan > 0 {
			if err := inflightSem.Acquire(r.Context(), 1); err != nil {
				sendResponse(http.StatusTooManyRequests, false, "service is busy", nil, rw)
				return
			}

			defer inflightSem.Release(1)
		}

		port, err := utils.GetRandomPort()
		if err != nil {
			sendResponse(http.StatusInternalServerError, false, err, nil, rw)
			return
		}

		t := parser.NewTask(&parser.TaskConfig{
			Timeout:           opts.Timeout,
			WaitAfterPageLoad: opts.WaitAfterPageLoad,
			Verbose:           opts.Verbose,
			ChromeApp:         opts.ChromeApp,
			DebuggerPort:      port,
			Headless:          true,
		})

		if err = t.Start(); err != nil {
			sendResponse(http.StatusInternalServerError, false, err, nil, rw)
			return
		}

		defer t.Cleanup()

		if err = t.Parse(site); err != nil {
			sendResponse(http.StatusInternalServerError, false, err, nil, rw)
			return
		}

		switch strings.ToLower(reportType) {
		case "", typeJSON:
			var prettyResult bool
			if r.FormValue(argPretty) != "" {
				prettyResult = true
			}

			jsonData, err := t.OutputJSON(prettyResult)
			if err != nil {
				sendResponse(http.StatusInternalServerError, false, err, nil, rw)
				return
			}

			rw.Header().Set("Content-Type", contentTypeJSON)
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte(jsonData))
		case typeHTML:
			htmlData, err := t.OutputHTML()
			if err != nil {
				sendResponse(http.StatusInternalServerError, false, err, nil, rw)
				return
			}

			rw.Header().Set("Content-Type", contentTypeHTML)
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte(htmlData))
		case typePDF:
			var f *os.File
			if f, err = ioutil.TempFile("", "gdpr_cookie*.pdf"); err != nil {
				return
			}

			tempPDF := f.Name()
			_ = f.Close()

			defer func() {
				_ = os.Remove(tempPDF)
			}()

			if err = t.OutputPDF(tempPDF); err != nil {
				sendResponse(http.StatusInternalServerError, false, err, nil, rw)
				return
			}

			pdfBytes, err := ioutil.ReadFile(tempPDF)
			if err != nil {
				sendResponse(http.StatusInternalServerError, false, err, nil, rw)
				return
			}

			rw.Header().Set("Content-Type", contentTypePDF)
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write(pdfBytes)
		}
	}
}

func handler(opts *cmd.CommonOptions) (err error) {
	if disableJSON && disablePDF && disableHTML {
		disableJSON = false
	}

	router := mux.NewRouter()
	router.Use(jsonContentType)
	router.HandleFunc("/", getVersionFunc(opts))
	router.HandleFunc("/api/v1/analyze", analyzeFunc(opts)).Methods(http.MethodGet, http.MethodPost)

	if maxInflightScan > 0 {
		inflightSem = semaphore.NewWeighted(int64(maxInflightScan))
	}

	s := &http.Server{
		Addr:         listenAddr,
		WriteTimeout: opts.Timeout * 2,
		ReadTimeout:  opts.Timeout * 2,
		IdleTimeout:  opts.Timeout * 2,
		Handler: handlers.CORS(
			handlers.AllowedHeaders([]string{"Content-Type"}),
		)(router),
	}

	logrus.Infof("server starting at %s, press Ctrl + C to stop", listenAddr)

	if err = s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		err = errors.Wrapf(err, "server unexpected stopped")
		return
	} else {
		err = nil
	}

	return
}
