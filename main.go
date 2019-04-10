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
	"os"
	"time"

	"github.com/CovenantSQL/CookieTester/cmd"
	"github.com/CovenantSQL/CookieTester/cmd/cli"
	"github.com/CovenantSQL/CookieTester/cmd/server"
	"github.com/CovenantSQL/CookieTester/cmd/version"
	"github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	app      = kingpin.New("CookieTester", "website cookie usage report generator")
	logLevel string
	options  cmd.CommonOptions
)

func init() {
	app.Flag("chrome", "chrome application to run as remote debugger").StringVar(&options.ChromeApp)
	app.Flag("verbose", "run debugger in verbose mode").BoolVar(&options.Verbose)
	app.Flag("timeout", "timeout for a single cookie scan").Default(time.Minute.String()).DurationVar(&options.Timeout)
	app.Flag("wait", "wait duration after page load in scan").DurationVar(&options.WaitAfterPageLoad)
	app.Flag("log-level", "set log level").PreAction(func(context *kingpin.ParseContext) (err error) {
		if logLevel != "" {
			var lvl logrus.Level
			lvl, err = logrus.ParseLevel(logLevel)
			if err == nil {
				logrus.SetLevel(lvl)
			}
		}

		return
	}).StringVar(&logLevel)

	cli.RegisterCommand(app, &options)
	version.RegisterCommand(app, &options)
	server.RegisterCommand(app, &options)
}

func main() {
	kingpin.MustParse(app.Parse(os.Args[1:]))
}
