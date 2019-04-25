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
	"database/sql"
	"net/url"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/client"
)

type Classifier struct {
	db *sql.DB
}

func NewClassifier(dsn string) (c *Classifier, err error) {
	u, err := url.Parse(dsn)
	if err != nil {
		err = errors.Wrap(err, "init classifier failed")
		return
	}

	c = &Classifier{}

	switch strings.ToLower(u.Scheme) {
	case "covenantsql", "cql":
		queries := u.Query()
		cfg := queries.Get("config")
		passwd := queries.Get("password")
		if cfg != "" {
			if err = client.Init(cfg, []byte(passwd)); err != nil {
				return
			}
		}
		c.db, err = sql.Open("covenantsql", dsn)
	case "sqlite3", "sqlite":
		c.db, err = sql.Open("sqlite3", "file:"+strings.TrimPrefix(dsn, u.Scheme+"://"))
	default:
		err = errors.New("invalid classifier database dsn")
	}

	return
}

func (c *Classifier) GetCookieDetail(name string) (cookieType string, cookieDesc string, err error) {
	err = c.db.QueryRow("SELECT cookie_type, cookie_desc FROM cookies WHERE cookie_name = ? LIMIT 1", name).
		Scan(&cookieType, &cookieDesc)
	if err == sql.ErrNoRows {
		err = nil
	}
	return
}
