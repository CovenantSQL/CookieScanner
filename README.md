# Cookie Tester

Cookie Tester is a simple utility to analyze website cookie status and generate reports for GDPR-compliance.

### Installation

Requires MacOS/Linux system.

First, install [`Google Chrome`](<https://www.google.com/chrome/>) in your operating system.

Then, install the `CookieTester` using `go get`.

```shell
$ go get github.com/CovenantSQL/CookieTester
```

### Usage

CookieTester is capable of geneating reports in `json/html/pdf` format.

```shell
$ CookieTester -help
Usage of ./CookieTester:
  -cmd string
    	command to execute to start the browser (default "\"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome\" --headless --remote-debugging-port=9222 --no-default-browser-check --no-first-run --hide-scrollbars --bwsi --disable-gpu --user-data-dir=/var/folders/dz/qvdhk7t94mg68jsmd8ccv9x40000gn/T/gdpr_cookie228503997 about:blank")
  -headless
    	headless mode
  -html string
    	output as html (save with specified file name)
  -json
    	output as json
  -pdf string
    	output as pdf (save with specified file name)
  -port string
    	Chrome remote debugger port (default "localhost:9222")
  -timeout duration
    	timeout for cookie scan (default 1m0s)
  -verbose
    	verbose logging
  -version
    	display remote devtools version
  -wait duration
    	wait duration after page load (capturing ajax/deferred requests)
```

### Examples

Generate HTML report for `www.google.com`.

```shell
$ CookieTester -headless -html google.html www.google.com
```

Just wait for a while, you will found `google.html` showing results containing cookie descriptions like `NID`.
