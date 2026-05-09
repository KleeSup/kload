# kload
Lightweight CLI load-testing-tool for HTTPS-Endpoints written in Go.

## Usage
By invoking the command ```kload``` the following flags can be used:

| Flag                      | Parameter type | Required | Description                                                                 | Usage example                           |
|:--------------------------|:--------------:|:--------:|:----------------------------------------------------------------------------|-----------------------------------------|
| ``-u``, ``--url``         |    ``URL``     |   Yes    | Target URL to load test                                                     | ``-u https://example.com``              |
| ``-m``, ``--method``      |    ``Text``    |    No    | HTTP method (GET, POST, PUT, PATCH, DELETE)                                 | ``-m GET``                              |
| ``-H``, ``--header``      |    ``Text``    |    No    | Request headers, repeatable                                                 | ``-H "Content-Type: application/json"`` |
| ``-b``, ``--body``        |    ``Text``    |    No    | Request body as raw string (used with POST/PUT)                             | ``-b '{"user":"test"}'``                |
| ``--body-file``           |    ``Path``    |    No    | Read request body from file instead of inline string                        | ``--body-file body.json``               |
| ``-n``, ``--requests``    |  ``Integer``   |    No    | Total number of requests to send                                            | ``-n 100``                              |
| ``-c``, ``--concurrency`` |  ``Integer``   |    No    | Total number of concurrent workers sending requests                         | ``-c 10``                               |
| ``-d``, ``--duration``    |  ``Duration``  |    No    | Run for a fixed duration instead of a fixed request count                   | ``-d 30ms``                             |
| ``-rps``                  |  ``Integer``   |    No    | Cap requests per second (useful for rate-limit testing, 0 = unlimited)      | ``-rps 30``                             |
| ``--warmup``              |  ``Integer``   |    No    | Number of warmup requests to send before measuring (results excluded)       | ``--warmup 5``                          |
| ``-t``, ``--timeout``     |  ``Duration``  |    No    | Per-request timeout                                                         | ``-t 500ms``                            |
| ``--retries``             |  ``Integer``   |    No    | Number of retries on timeout before marking as failed                       | ``--retries 3``                         |
| ``--no-redirect``         |  ``Boolean``   |    No    | Disable following HTTP redirects                                            | ``--no-redirect``                       |
| ``-o``, ``--output``      |    ``Path``    |    No    | Write results to file. Format inferred from extension (``.json``, ``.csv``) | ``-o report.json``                      |
| ``--format``              |    ``Text``    |    No    | Terminal output style (``table, json, csv, silent``)                        | ``--format table``                      |
| ``--no-progress``         |  ``Boolean``   |    No    | Hide the live progress bar                                                  | ``--no-progress``                       |
| ``-v``, ``--verbose``     |  ``Boolean``   |    No    | Print each request result individually as it completes                      | ``-v``                                  |
| ``--insecure``            |  ``Boolean``   |    No    | Skip TLS certificate verification                                           | ``--insecure``                          |
| ``--http2``               |  ``Boolean``   |    No    | Force HTTP/2 (default negotiates)                                           | ``--http2``                             |
| ``--keep-alive``          |  ``Boolean``   |    No    | Reuse TCP connections between requests                                      | ``--keep-alive``                        |

Execution examples
```
# Basic GET with 500 requests and 20 workers
kload -u https://api.example.com/health -n 500 -c 20

# POST with body, 30s run, capped at 100 rps
kload -u https://api.example.com/login \
      -m POST \
      -H "Content-Type: application/json" \
      -b '{"user":"test","pass":"test"}' \
      -d 30s --rps 100

# Export results as JSON and no progress bar (CI mode)
kload -u https://api.example.com -n 1000 -c 50 \
      --no-progress -o results.json
```

## Dependencies
- [cobra](https://github.com/spf13/cobra) (v1.10.2) by spf13
- [progressbar](https://github.com/schollz/progressbar) (v1.0.0) by schollz
- [fatih/color](https://github.com/fatih/color) (v1.19.0) by fatih