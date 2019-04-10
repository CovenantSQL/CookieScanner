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

package version

import (
	"github.com/CovenantSQL/CookieTester/cmd"
	"github.com/CovenantSQL/CookieTester/parser"
	"github.com/gobs/pretty"
	"github.com/pkg/errors"
	"gopkg.in/alecthomas/kingpin.v2"
	"net"
)

func RegisterCommand(app *kingpin.Application, opts *cmd.CommonOptions) {
	app.Command("version", "get debugger version").Action(func(context *kingpin.ParseContext) (err error) {
		// random port
		l, err := net.Listen("tcp", ":0")
		if err != nil {
			err = errors.Wrapf(err, "could not listen new ports")
			return
		}
		port := l.Addr().(*net.TCPAddr).Port
		_ = l.Close()

		t := parser.NewTask(&parser.TaskConfig{
			Timeout:           opts.Timeout,
			WaitAfterPageLoad: opts.WaitAfterPageLoad,
			Verbose:           opts.Verbose,
			ChromeApp:         opts.ChromeApp,
			DebuggerPort:      port,
			Headless:          true,
		})

		if err = t.Start(); err != nil {
			err = errors.Wrapf(err, "start debugger failed")
			return
		}

		defer t.Cleanup()

		ver, err := t.Version()
		if err != nil {
			err = errors.Wrapf(err, "get version failed")
			return
		}

		pretty.PrettyPrint(ver)

		return
	})
}
