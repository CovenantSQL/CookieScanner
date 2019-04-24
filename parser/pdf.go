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
	}).Parse(`
<!DOCTYPE html>
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
            <img class="image w-25"
                 src="data:image/svg+xml;base64,PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0iVVRGLTgiPz4KPHN2ZyB3aWR0aD0iNDA2cHgiIGhlaWdodD0iNjZweCIgdmlld0JveD0iMCAwIDQwNiA2NiIgdmVyc2lvbj0iMS4xIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHhtbG5zOnhsaW5rPSJodHRwOi8vd3d3LnczLm9yZy8xOTk5L3hsaW5rIj4KICAgIDwhLS0gR2VuZXJhdG9yOiBTa2V0Y2ggNTMuMiAoNzI2NDMpIC0gaHR0cHM6Ly9za2V0Y2hhcHAuY29tIC0tPgogICAgPHRpdGxlPkdEUFJFeHBlcnQ8L3RpdGxlPgogICAgPGRlc2M+Q3JlYXRlZCB3aXRoIFNrZXRjaC48L2Rlc2M+CiAgICA8ZyBpZD0iUGFnZS0xIiBzdHJva2U9Im5vbmUiIHN0cm9rZS13aWR0aD0iMSIgZmlsbD0ibm9uZSIgZmlsbC1ydWxlPSJldmVub2RkIj4KICAgICAgICA8ZyBpZD0iR0RQUkV4cGVydCIgdHJhbnNmb3JtPSJ0cmFuc2xhdGUoMC43NjkwMDAsIDAuNTk2MDAwKSIgZmlsbC1ydWxlPSJub256ZXJvIj4KICAgICAgICAgICAgPHBhdGggZD0iTTQ4LjU2NCw1MS40MDQgTDQwLjYxMiw1MS40MDQgTDM5Ljk3Myw0NS42NTMgQzM4LjY5NDk5MzYsNDcuNTkzNjc2NCAzNi43ODk4NDYsNDkuMjE0ODI2OCAzNC4yNTc1LDUwLjUxNjUgQzMxLjcyNTE1NCw1MS44MTgxNzMyIDI4LjY4NDAxNzgsNTIuNDY5IDI1LjEzNCw1Mi40NjkgQzE4LjEyODYzMTYsNTIuNDY5IDEyLjE4ODM1NzcsNTAuMDc4NjkwNiA3LjMxMyw0NS4yOTggQzIuNDM3NjQyMjksNDAuNTE3MzA5NCA0LjU0NzQ3MzUxZS0xMywzNC4xNzQ3MDYyIDQuNTQ3NDczNTFlLTEzLDI2LjI3IEM0LjU0NzQ3MzUxZS0xMywxOC40MTI2Mjc0IDIuNTIwNDc0OCwxMi4wNzAwMjQxIDcuNTYxNSw3LjI0MiBDMTIuNjAyNTI1MiwyLjQxMzk3NTg2IDE4LjY3Mjk2NDUsMCAyNS43NzMsMCBDMzEuOTI2MzY0MSwwIDM2Ljk1NTQ4MDUsMS40OTA5ODUwOSA0MC44NjA1LDQuNDczIEM0NC43NjU1MTk1LDcuNDU1MDE0OTEgNDcuNDA0MzI2NSwxMS4xMjMzMTE2IDQ4Ljc3NywxNS40NzggTDM5LjQwNSwxOC44MTUgQzM4LjY0NzY2MjksMTYuMjExNjUzNyAzNy4xMjExNzgxLDEzLjk1MTUwOTYgMzQuODI1NSwxMi4wMzQ1IEMzMi41Mjk4MjE5LDEwLjExNzQ5MDQgMjkuNTEyMzUyLDkuMTU5IDI1Ljc3Myw5LjE1OSBDMjEuNTEyOTc4Nyw5LjE1OSAxNy44MDkxODI0LDEwLjYyNjMxODcgMTQuNjYxNSwxMy41NjEgQzExLjUxMzgxNzYsMTYuNDk1NjgxMyA5Ljk0LDIwLjczMTk3MjMgOS45NCwyNi4yNyBDOS45NCwzMS43MTMzNjA2IDExLjQ2NjQ4NDcsMzUuOTQ5NjUxNSAxNC41MTk1LDM4Ljk3OSBDMTcuNTcyNTE1Myw0Mi4wMDgzNDg1IDIxLjM0NzMxMDksNDMuNTIzIDI1Ljg0NCw0My41MjMgQzMwLjAwOTM1NDIsNDMuNTIzIDMzLjIwNDMyMjIsNDIuNDY5ODQzOSAzNS40MjksNDAuMzYzNSBDMzcuNjUzNjc3OCwzOC4yNTcxNTYxIDM4Ljk1NTMzMTQsMzYuMDQ0MzQ0OSAzOS4zMzQsMzMuNzI1IEwyMy40MywzMy43MjUgTDIzLjQzLDI1LjIwNSBMNDguNTY0LDI1LjIwNSBMNDguNTY0LDUxLjQwNCBaIE02OC4yMzEsNDIuMzE2IEw3Ni4wNDEsNDIuMzE2IEM4MC40NDMwMjIsNDIuMzE2IDg0LjA2Mzk4NTgsNDAuOTQzMzQ3MSA4Ni45MDQsMzguMTk4IEM4OS43NDQwMTQyLDM1LjQ1MjY1MjkgOTEuMTY0LDMxLjQ3NjY5MjcgOTEuMTY0LDI2LjI3IEM5MS4xNjQsMjEuMDYzMzA3MyA4OS43NDQwMTQyLDE3LjA3NTUxMzggODYuOTA0LDE0LjMwNjUgQzg0LjA2Mzk4NTgsMTEuNTM3NDg2MiA4MC40NjY2ODg0LDEwLjE1MyA3Ni4xMTIsMTAuMTUzIEw2OC4yMzEsMTAuMTUzIEw2OC4yMzEsNDIuMzE2IFogTTc2LjM5Niw1MS40MDQgTDU4LjQzMyw1MS40MDQgTDU4LjQzMywxLjA2NSBMNzYuNDY3LDEuMDY1IEM4My44MDM3MDMzLDEuMDY1IDg5Ljc3OTQ3NjksMy4zMTMzMTA4NSA5NC4zOTQ1LDcuODEgQzk5LjAwOTUyMzEsMTIuMzA2Njg5MiAxMDEuMzE3LDE4LjQ1OTk2MSAxMDEuMzE3LDI2LjI3IEMxMDEuMzE3LDM0LjAzMjcwNTUgOTguOTk3Njg5OSw0MC4xNjIzMTA5IDk0LjM1OSw0NC42NTkgQzg5LjcyMDMxMDEsNDkuMTU1Njg5MSA4My43MzI3MDM0LDUxLjQwNCA3Ni4zOTYsNTEuNDA0IFogTTExOS43MDYsMjMuOTk4IEwxMjcuNDQ1LDIzLjk5OCBDMTI5Ljg1OTAxMiwyMy45OTggMTMxLjc3NTk5MywyMy4zNDcxNzMyIDEzMy4xOTYsMjIuMDQ1NSBDMTM0LjYxNjAwNywyMC43NDM4MjY4IDEzNS4zMjYsMTkuMDA0MzQ0MiAxMzUuMzI2LDE2LjgyNyBDMTM1LjMyNiwxNC42MDIzMjIyIDEzNC42Mjc4NCwxMi44MzkxNzMyIDEzMy4yMzE1LDExLjUzNzUgQzEzMS44MzUxNiwxMC4yMzU4MjY4IDEyOS45MDYzNDYsOS41ODUgMTI3LjQ0NSw5LjU4NSBMMTE5LjcwNiw5LjU4NSBMMTE5LjcwNiwyMy45OTggWiBNMTI4LjY1MiwzMi40NDcgTDExOS42MzUsMzIuNDQ3IEwxMTkuNjM1LDUxLjQwNCBMMTA5LjgzNyw1MS40MDQgTDEwOS44MzcsMS4wNjUgTDEyOC42NTIsMS4wNjUgQzEzMy41NzQ2OTEsMS4wNjUgMTM3LjU2MjQ4NSwyLjUzMjMxODY2IDE0MC42MTU1LDUuNDY3IEMxNDMuNjY4NTE1LDguNDAxNjgxMzQgMTQ1LjE5NSwxMi4xNjQ2NDM3IDE0NS4xOTUsMTYuNzU2IEMxNDUuMTk1LDIxLjM0NzM1NjMgMTQzLjY2ODUxNSwyNS4xMTAzMTg3IDE0MC42MTU1LDI4LjA0NSBDMTM3LjU2MjQ4NSwzMC45Nzk2ODEzIDEzMy41NzQ2OTEsMzIuNDQ3IDEyOC42NTIsMzIuNDQ3IFogTTE3Ny43MTMsNTEuNDA0IEwxNjcuODQ0LDMyLjAyMSBMMTYyLjczMiwzMi4wMjEgTDE2Mi43MzIsNTEuNDA0IEwxNTIuODYzLDUxLjQwNCBMMTUyLjg2MywxLjA2NSBMMTcyLjUzLDEuMDY1IEMxNzcuMzU4MDI0LDEuMDY1IDE4MS4yMzkzMTksMi41MzIzMTg2NiAxODQuMTc0LDUuNDY3IEMxODcuMTA4NjgxLDguNDAxNjgxMzQgMTg4LjU3NiwxMi4wOTM2NDQ0IDE4OC41NzYsMTYuNTQzIEMxODguNTc2LDIwLjA0NTY4NDIgMTg3LjYxNzUxLDIzLjA2MzE1NCAxODUuNzAwNSwyNS41OTU1IEMxODMuNzgzNDksMjguMTI3ODQ2IDE4MS4xNjgzNSwyOS44NjczMjg2IDE3Ny44NTUsMzAuODE0IEwxODguNjQ3LDUxLjQwNCBMMTc3LjcxMyw1MS40MDQgWiBNMTYyLjczMiwyMy41NzIgTDE3MC42ODQsMjMuNTcyIEMxNzMuMTQ1MzQ2LDIzLjU3MiAxNzUuMDc0MTYsMjIuOTMzMDA2NCAxNzYuNDcwNSwyMS42NTUgQzE3Ny44NjY4NCwyMC4zNzY5OTM2IDE3OC41NjUsMTguNjk2Njc3MSAxNzguNTY1LDE2LjYxNCBDMTc4LjU2NSwxNC40ODM5ODk0IDE3Ny44NjY4NCwxMi43ODAwMDY0IDE3Ni40NzA1LDExLjUwMiBDMTc1LjA3NDE2LDEwLjIyMzk5MzYgMTczLjE0NTM0Niw5LjU4NSAxNzAuNjg0LDkuNTg1IEwxNjIuNzMyLDkuNTg1IEwxNjIuNzMyLDIzLjU3MiBaIiBpZD0iR0RQUiIgZmlsbD0iIzNGM0YzRiI+PC9wYXRoPgogICAgICAgICAgICA8cGF0aCBkPSJNMjI4Ljc2Miw1MS40MDQgTDE5Ny4yMzgsNTEuNDA0IEwxOTcuMjM4LDEuMDY1IEwyMjguNzYyLDEuMDY1IEwyMjguNzYyLDEwLjI5NSBMMjA3LjAzNiwxMC4yOTUgTDIwNy4wMzYsMjEuNzk3IEwyMjYuNzAzLDIxLjc5NyBMMjI2LjcwMywzMC41MyBMMjA3LjAzNiwzMC41MyBMMjA3LjAzNiw0Mi4xNzQgTDIyOC43NjIsNDIuMTc0IEwyMjguNzYyLDUxLjQwNCBaIE0yMzMuNjYxLDUxLjQwNCBMMjQ2LjA4NiwzMy44NjcgTDIzMy42NjEsMTYuNDcyIEwyNDQuODc5LDE2LjQ3MiBDMjQ1LjMwNTAwMiwxNy4xMzQ2NyAyNDYuNDc2NDksMTguODc0MTUyNiAyNDguMzkzNSwyMS42OTA1IEMyNTAuMzEwNTEsMjQuNTA2ODQ3NCAyNTEuNDU4MzMxLDI2LjIyMjY2MzYgMjUxLjgzNywyNi44MzggTDI1OC43MjQsMTYuNDcyIEwyNjkuNDQ1LDE2LjQ3MiBMMjU3LjIzMywzMy41ODMgTDI2OS44NzEsNTEuNDA0IEwyNTguNzk1LDUxLjQwNCBMMjUxLjQ4Miw0MC42ODMgQzI1MS4yNDUzMzIsNDEuMDYxNjY4NiAyNTAuNTcwODM5LDQyLjA3OTMyNSAyNDkuNDU4NSw0My43MzYgQzI0OC4zNDYxNjEsNDUuMzkyNjc0OSAyNDcuMzA0ODM4LDQ2Ljk0MjgyNjEgMjQ2LjMzNDUsNDguMzg2NSBDMjQ1LjM2NDE2Miw0OS44MzAxNzM5IDI0NC42ODk2NjksNTAuODM1OTk3MiAyNDQuMzExLDUxLjQwNCBMMjMzLjY2MSw1MS40MDQgWiBNMjg1LjA2NSw2NC44OTQgTDI3NS42MjIsNjQuODk0IEwyNzUuNjIyLDE2LjQ3MiBMMjg0Ljc4MSwxNi40NzIgTDI4NC43ODEsMjAuNzMyIEMyODUuNjMzMDA0LDE5LjMxMTk5MjkgMjg3LjAwNTY1NywxOC4xMTY4MzgyIDI4OC44OTksMTcuMTQ2NSBDMjkwLjc5MjM0MywxNi4xNzYxNjE4IDI5Mi45OTMzMjEsMTUuNjkxIDI5NS41MDIsMTUuNjkxIEMzMDAuNDcyMDI1LDE1LjY5MSAzMDQuNDAwNjUyLDE3LjM4MzE0OTcgMzA3LjI4OCwyMC43Njc1IEMzMTAuMTc1MzQ4LDI0LjE1MTg1MDMgMzExLjYxOSwyOC41MTgzMDY2IDMxMS42MTksMzMuODY3IEMzMTEuNjE5LDM5LjIxNTY5MzQgMzEwLjA5MjUxNSw0My42MTc2NDk0IDMwNy4wMzk1LDQ3LjA3MyBDMzAzLjk4NjQ4NSw1MC41MjgzNTA2IDMwMC4wMjIzNTgsNTIuMjU2IDI5NS4xNDcsNTIuMjU2IEMyOTAuNTA4MzEsNTIuMjU2IDI4Ny4xNDc2NzcsNTAuODM2MDE0MiAyODUuMDY1LDQ3Ljk5NiBMMjg1LjA2NSw2NC44OTQgWiBNMjk5LjgzMyw0MS4xOCBDMzAxLjQ4OTY3NSwzOS4zODEzMjQzIDMwMi4zMTgsMzYuOTY3MzQ4NSAzMDIuMzE4LDMzLjkzOCBDMzAyLjMxOCwzMC45MDg2NTE1IDMwMS41MDE1MDgsMjguNTE4MzQyMSAyOTkuODY4NSwyNi43NjcgQzI5OC4yMzU0OTIsMjUuMDE1NjU3OSAyOTYuMTY0Njc5LDI0LjE0IDI5My42NTYsMjQuMTQgQzI5MS4xNDczMjEsMjQuMTQgMjg5LjA2NDY3NSwyNS4wMjc0OTExIDI4Ny40MDgsMjYuODAyNSBDMjg1Ljc1MTMyNSwyOC41Nzc1MDg5IDI4NC45MjMsMzAuOTU1OTg1MSAyODQuOTIzLDMzLjkzOCBDMjg0LjkyMywzNi45MjAwMTQ5IDI4NS43NTEzMjUsMzkuMzEwMzI0MyAyODcuNDA4LDQxLjEwOSBDMjg5LjA2NDY3NSw0Mi45MDc2NzU3IDI5MS4xNDczMjEsNDMuODA3IDI5My42NTYsNDMuODA3IEMyOTYuMTY0Njc5LDQzLjgwNyAyOTguMjIzNjU5LDQyLjkzMTM0MjEgMjk5LjgzMyw0MS4xOCBaIE0zMjUuOTYxLDI5Ljk2MiBMMzQxLjU4MSwyOS45NjIgQzM0MS40ODYzMzMsMjguMDIxMzIzNiAzNDAuNzg4MTczLDI2LjM4ODM0IDMzOS40ODY1LDI1LjA2MyBDMzM4LjE4NDgyNywyMy43Mzc2NiAzMzYuMjc5Njc5LDIzLjA3NSAzMzMuNzcxLDIzLjA3NSBDMzMxLjQ5ODk4OSwyMy4wNzUgMzI5LjY1MzAwNywyMy43ODQ5OTI5IDMyOC4yMzMsMjUuMjA1IEMzMjYuODEyOTkzLDI2LjYyNTAwNzEgMzI2LjA1NTY2NywyOC4yMTA2NTc5IDMyNS45NjEsMjkuOTYyIFogTTM0Mi41MDQsMzkuMDUgTDM1MC4zODUsNDEuMzkzIEMzNDkuNDM4MzI5LDQ0LjYxMTY4MjggMzQ3LjU4MDUxNCw0Ny4yNjIzMjI5IDM0NC44MTE1LDQ5LjM0NSBDMzQyLjA0MjQ4Niw1MS40Mjc2NzcxIDMzOC41OTkwMjEsNTIuNDY5IDMzNC40ODEsNTIuNDY5IEMzMjkuNDYzNjQyLDUyLjQ2OSAzMjUuMjAzNjg0LDUwLjc3Njg1MDMgMzIxLjcwMSw0Ny4zOTI1IEMzMTguMTk4MzE2LDQ0LjAwODE0OTcgMzE2LjQ0NywzOS40NzYwMjg0IDMxNi40NDcsMzMuNzk2IEMzMTYuNDQ3LDI4LjM5OTk3MyAzMTguMTUwOTgzLDIzLjk4NjE4MzggMzIxLjU1OSwyMC41NTQ1IEMzMjQuOTY3MDE3LDE3LjEyMjgxNjIgMzI4Ljk5MDMxLDE1LjQwNyAzMzMuNjI5LDE1LjQwNyBDMzM5LjAyNTAyNywxNS40MDcgMzQzLjI0OTQ4NSwxNy4wMTYzMTcyIDM0Ni4zMDI1LDIwLjIzNSBDMzQ5LjM1NTUxNSwyMy40NTM2ODI4IDM1MC44ODIsMjcuODc5MzA1MiAzNTAuODgyLDMzLjUxMiBDMzUwLjg4MiwzMy44OTA2Njg2IDM1MC44NzAxNjcsMzQuMzE2NjY0MyAzNTAuODQ2NSwzNC43OSBDMzUwLjgyMjgzMywzNS4yNjMzMzU3IDM1MC44MTEsMzUuNjQxOTk4NiAzNTAuODExLDM1LjkyNiBMMzUwLjc0LDM2LjQyMyBMMzI1Ljc0OCwzNi40MjMgQzMyNS44NDI2NjcsMzguNjk1MDExNCAzMjYuNzQxOTkxLDQwLjU4ODMyNTggMzI4LjQ0Niw0Mi4xMDMgQzMzMC4xNTAwMDksNDMuNjE3Njc0MiAzMzIuMTg1MzIxLDQ0LjM3NSAzMzQuNTUyLDQ0LjM3NSBDMzM4LjU3NTM1Myw0NC4zNzUgMzQxLjIyNTk5NCw0Mi42MDAwMTc3IDM0Mi41MDQsMzkuMDUgWiBNMzc5LjYzNywxNi4zMyBMMzc5LjYzNywyNS44NDQgQzM3OC42OTAzMjksMjUuNjU0NjY1NyAzNzcuNzQzNjcxLDI1LjU2IDM3Ni43OTcsMjUuNTYgQzM3NC4wOTg5ODcsMjUuNTYgMzcxLjkyMTY3NSwyNi4zMjkxNTkgMzcwLjI2NSwyNy44Njc1IEMzNjguNjA4MzI1LDI5LjQwNTg0MSAzNjcuNzgsMzEuOTI2MzE1OCAzNjcuNzgsMzUuNDI5IEwzNjcuNzgsNTEuNDA0IEwzNTguMzM3LDUxLjQwNCBMMzU4LjMzNywxNi40NzIgTDM2Ny40OTYsMTYuNDcyIEwzNjcuNDk2LDIxLjY1NSBDMzY5LjIwMDAwOSwxOC4wMTAzMTUxIDM3Mi41MTMzMDksMTYuMTg4IDM3Ny40MzYsMTYuMTg4IEMzNzcuOTU2NjY5LDE2LjE4OCAzNzguNjkwMzI5LDE2LjIzNTMzMjkgMzc5LjYzNywxNi4zMyBaIE0zOTcuNzQyLDYuMDM1IEwzOTcuNzQyLDE2LjQ3MiBMNDA0Ljc3MSwxNi40NzIgTDQwNC43NzEsMjQuODUgTDM5Ny43NDIsMjQuODUgTDM5Ny43NDIsMzkuNDc2IEMzOTcuNzQyLDQwLjk0MzM0MDcgMzk4LjA3MzMzLDQxLjk4NDY2MzYgMzk4LjczNiw0Mi42IEMzOTkuMzk4NjcsNDMuMjE1MzM2NCA0MDAuNDM5OTkzLDQzLjUyMyA0MDEuODYsNDMuNTIzIEM0MDMuMTM4MDA2LDQzLjUyMyA0MDQuMTA4MzMsNDMuNDI4MzM0MyA0MDQuNzcxLDQzLjIzOSBMNDA0Ljc3MSw1MS4wNDkgQzQwMy4zOTgzMjYsNTEuNjE3MDAyOCA0MDEuNjIzMzQ0LDUxLjkwMSAzOTkuNDQ2LDUxLjkwMSBDMzk2LjAzNzk4Myw1MS45MDEgMzkzLjM0MDAxLDUwLjk0MjUwOTYgMzkxLjM1Miw0OS4wMjU1IEMzODkuMzYzOTksNDcuMTA4NDkwNCAzODguMzcsNDQuNDY5NjgzNSAzODguMzcsNDEuMTA5IEwzODguMzcsMjQuODUgTDM4Mi4wNTEsMjQuODUgTDM4Mi4wNTEsMTYuNDcyIEwzODMuODI2LDE2LjQ3MiBDMzg1LjU3NzM0MiwxNi40NzIgMzg2LjkxNDQ5NSwxNS45NjMxNzE4IDM4Ny44Mzc1LDE0Ljk0NTUgQzM4OC43NjA1MDUsMTMuOTI3ODI4MiAzODkuMjIyLDEyLjU5MDY3NDkgMzg5LjIyMiwxMC45MzQgTDM4OS4yMjIsNi4wMzUgTDM5Ny43NDIsNi4wMzUgWiIgaWQ9IkV4cGVydCIgZmlsbD0iIzAwNkVFNSI+PC9wYXRoPgogICAgICAgIDwvZz4KICAgIDwvZz4KPC9zdmc+"/>
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
