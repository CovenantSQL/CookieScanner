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
	"context"
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
	gomail "gopkg.in/gomail.v2"
)

const (
	argSite   = "site"
	argType   = "type"
	argPretty = "pretty"
	argTo     = "to"
	argAsync  = "async"

	typeJSON  = "json"
	typeHTML  = "html"
	typePDF   = "pdf"
	typeEmail = "email"

	contentTypeJSON = "application/json"
	contentTypeHTML = "text/html"
	contentTypePDF  = "application/pdf"

	mailSubjectPrefix   = `CookieScan report for `
	mailContentTemplate = `Your report for site %s is generated, please see the PDF attachment.`
)

var (
	listenAddr string

	disablePDF   bool
	disableHTML  bool
	disableJSON  bool
	disableEmail bool

	versionOnce sync.Once
	versionLock sync.RWMutex
	versionInfo *godet.Version
	versionErr  error

	maxInflightScan int
	inflightSem     *semaphore.Weighted

	mailServer   string
	mailPort     int
	mailUser     string
	mailPassword string
	mailFrom     string
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
	c.Flag("disable-email", "disable htm output support").BoolVar(&disableEmail)
	c.Flag("mail-server", "mail server hostname").Envar("MAIL_SERVER").StringVar(&mailServer)
	c.Flag("mail-port", "mail server port").Envar("MAIL_PORT").IntVar(&mailPort)
	c.Flag("mail-user", "mail login user").Envar("MAIL_USER").StringVar(&mailUser)
	c.Flag("mail-password", "mail login password").Envar("MAIL_PASSWORD").StringVar(&mailPassword)
	c.Flag("mail-from", "mail sender from address").Envar("MAIL_FROM").StringVar(&mailFrom)
	c.Action(func(context *kingpin.ParseContext) error {
		return handler(opts)
	})
}

func getVersionFunc(opts *cmd.CommonOptions) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		versionOnce.Do(func() {
			var err error
			defer func() {
				versionErr = err
			}()
			port, err := utils.GetRandomPort()
			if err != nil {
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
				return
			}

			defer t.Cleanup()

			versionLock.Lock()
			defer versionLock.Unlock()
			versionInfo, err = t.Version()

			if err != nil {
				return
			}
		})

		versionLock.RLock()
		defer versionLock.RUnlock()

		if versionInfo != nil {
			sendResponse(http.StatusOK, true, nil, versionInfo, rw)
		} else if versionErr != nil {
			sendResponse(http.StatusInternalServerError, false, versionErr, nil, rw)
		} else {
			sendResponse(http.StatusInternalServerError, false, "could not get server version", nil, rw)
		}
	}
}

func asyncEmailReport(opts *cmd.CommonOptions, site string, mailTo string) {
	if maxInflightScan > 0 {
		if err := inflightSem.Acquire(context.Background(), 1); err != nil {
			logrus.WithFields(logrus.Fields{
				"site": site,
				"to":   mailTo,
			}).WithError(err).Error("wait for email report")
			return
		}

		defer inflightSem.Release(1)
	}

	port, err := utils.GetRandomPort()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"site": site,
			"to":   mailTo,
		}).WithError(err).Error("start chrome task with random debugger port failed")
		return
	}

	t := parser.NewTask(&parser.TaskConfig{
		Timeout:           opts.Timeout,
		WaitAfterPageLoad: opts.WaitAfterPageLoad,
		Verbose:           opts.Verbose,
		ChromeApp:         opts.ChromeApp,
		DebuggerPort:      port,
		Headless:          true,
		Classifier:        opts.ClassifierHandler,
	})

	if err = t.Start(); err != nil {
		logrus.WithFields(logrus.Fields{
			"site": site,
			"to":   mailTo,
		}).WithError(err).Error("start chrome task failed")
		return
	}

	defer t.Cleanup()

	if err = t.Parse(site); err != nil {
		logrus.WithFields(logrus.Fields{
			"site": site,
			"to":   mailTo,
		}).WithError(err).Error("load and parse website failed")
		return
	}

	var f *os.File
	if f, err = ioutil.TempFile("", "cookie_scan_*.pdf"); err != nil {
		logrus.WithFields(logrus.Fields{
			"site": site,
			"to":   mailTo,
		}).WithError(err).Error("create temp pdf scan file failed")
		return
	}

	tempPDF := f.Name()
	_ = f.Close()

	defer func() {
		_ = os.Remove(tempPDF)
	}()

	if err = t.OutputPDFToFile(tempPDF); err != nil {
		logrus.WithFields(logrus.Fields{
			"site": site,
			"to":   mailTo,
		}).WithError(err).Error("generate pdf report failed")
		return
	}

	d := gomail.NewPlainDialer(mailServer, mailPort, mailUser, mailPassword)
	m := gomail.NewMessage()
	m.SetHeader("From", mailFrom)
	m.SetAddressHeader("To", mailTo, mailTo)
	m.SetHeader("Subject", mailSubjectPrefix+site)
	m.SetBody("text/html", fmt.Sprintf(mailContentTemplate, site))
	m.Attach(tempPDF, gomail.SetHeader(map[string][]string{"Content-Type": {contentTypePDF}}))

	defer m.Reset()

	if err = d.DialAndSend(m); err != nil {
		logrus.WithFields(logrus.Fields{
			"site": site,
			"to":   mailTo,
		}).WithError(err).Error("send email failed")
		return
	}

	logrus.WithFields(logrus.Fields{
		"site": site,
		"to":   mailTo,
	}).Info("generate report complete")
}

func analyzeFunc(opts *cmd.CommonOptions) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// parse requests
		site := r.FormValue(argSite)
		reportType := r.FormValue(argType)
		asyncReport := r.FormValue(argAsync)

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
		case typeEmail:
			if disableEmail {
				sendResponse(http.StatusBadGateway, false, "email report is disabled", nil, rw)
				return
			}

			mailTo := r.FormValue(argTo)

			if mailServer == "" || mailPort == 0 || mailFrom == "" {
				sendResponse(http.StatusInternalServerError, false, "email setting not provided", nil, rw)
				return
			}

			if mailTo == "" {
				sendResponse(http.StatusBadRequest, false, "mail to address not provided", nil, rw)
				return
			}

			if asyncReport != "" {
				// issue async report
				go asyncEmailReport(opts, site, mailTo)
				sendResponse(http.StatusOK, true, nil, nil, rw)
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
			Classifier:        opts.ClassifierHandler,
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
			pdfBytes, err := t.OutputPDF()
			if err != nil {
				sendResponse(http.StatusInternalServerError, false, err, nil, rw)
				return
			}

			rw.Header().Set("Content-Type", contentTypePDF)
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write(pdfBytes)
		case typeEmail:
			// send email
			mailTo := r.FormValue(argTo)

			var f *os.File
			if f, err = ioutil.TempFile("", "cookie_scan_*.pdf"); err != nil {
				sendResponse(http.StatusInternalServerError, false, err, nil, rw)
				return
			}

			tempPDF := f.Name()
			_ = f.Close()

			defer func() {
				_ = os.Remove(tempPDF)
			}()

			if err = t.OutputPDFToFile(tempPDF); err != nil {
				sendResponse(http.StatusInternalServerError, false, err, nil, rw)
				return
			}

			d := gomail.NewPlainDialer(mailServer, mailPort, mailUser, mailPassword)
			m := gomail.NewMessage()
			m.SetHeader("From", mailFrom)
			m.SetAddressHeader("To", mailTo, mailTo)
			m.SetHeader("Subject", mailSubjectPrefix+site)
			m.SetBody("text/html", fmt.Sprintf(mailContentTemplate, site))
			m.Attach(tempPDF, gomail.SetHeader(map[string][]string{"Content-Type": {contentTypePDF}}))

			defer m.Reset()

			if err = d.DialAndSend(m); err != nil {
				sendResponse(http.StatusInternalServerError, false, err, nil, rw)
				return
			}

			sendResponse(http.StatusOK, true, nil, nil, rw)
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
