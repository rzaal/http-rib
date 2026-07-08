# ribnip

A terminal HTTP client, like Bruno, but living in a git repo. Requests are
plain YAML files, environments are plain YAML files, and every send just
shells out to real `curl`.

## Requirements

- Go 1.25+
- `curl` on your `PATH`

## Install / build

```sh
git clone <this-repo>
cd ribnip-http
go build -o ribnip .
```

## Usage

Run it inside a git repo (or any directory you want as your collection root):

```sh
./ribnip
```

- If the directory already has a `ribnip.yaml`, it opens that collection.
- If not, ribnip scaffolds a starter collection for you (manifest, a sample
  request, and `dev`/`acceptance`/`production` envs) and opens it.
- On startup (if the collection has more than one environment) you get an
  environment picker — choose once with `enter` before browsing requests.
  The active environment is always shown next to the selected request, and
  `e` reopens the picker any time to switch.

### Keys

| Key       | Action                                          |
|-----------|--------------------------------------------------|
| `↑`/`k`   | move up (request list, or env picker)             |
| `↓`/`j`   | move down (request list, or env picker)           |
| `enter`   | send selected request — or confirm env in picker  |
| `e`       | reopen the environment picker                     |
| `c`       | show the equivalent curl command                  |
| `r`       | reload collection from disk                       |
| `q`       | quit                             |

## Collection layout

```
ribnip.yaml            # collection manifest (name, version)
requests/
  httpbin/
    get.yaml
    post-json.yaml
    headers.yaml
env/
  dev.yaml
  acceptance.yaml
  production.yaml
```

Requests are organized into folders however you like under `requests/` —
folder structure becomes the sidebar tree.

### Request file

```yaml
name: Get
method: GET
url: "{{baseUrl}}/get"
headers:
  Accept: application/json
query:
  source: ribnip
body: ""
```

### Env file

```yaml
baseUrl: "https://httpbin.org"
token: "dev-secret"
```

Any `{{varName}}` in a request's `url`, `headers`, `query`, or `body` gets
substituted with the value from the active environment. Unresolved variables
are left as `{{varName}}` in the output so you can spot typos.

## Example collection

This repo ships with a working example under `requests/httpbin/`, pointed at
`https://httpbin.org` on the `dev` environment — build and run, pick a
request, hit `enter`, and you'll get a real response back.

## Project layout

```
main.go                    entry point: open or scaffold collection, start TUI
internal/collection/       manifest, request, and env loading + scaffolding
internal/render/           {{var}} substitution
internal/curl/             builds curl argv, execs curl, parses the response
internal/tui/               Bubble Tea UI (sidebar, viewport, status bar)
```

## Tests

```sh
go test ./...
```
