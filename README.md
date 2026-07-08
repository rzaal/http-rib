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
- If not, ribnip scaffolds a minimal starter for you: manifest, a `dev` env
  (see [Env file + secrets](#env-file--secrets)), and a `.gitignore` entry so
  secret values never get committed. `requests/` starts otherwise empty —
  add your own request files and, if needed, more envs (`acceptance.yaml`,
  `production.yaml`, ...) under `requests/env/`.
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

Everything lives under one `requests/` folder — request files and
environments alike. The `env/` subfolder is reserved: it's loaded as
environments rather than shown in the request sidebar. Scaffolding only
creates `dev`; the layout below shows a collection that's grown to add
`acceptance`/`production` envs and a couple of request files too.

```
ribnip.yaml                        # collection manifest (name, version)
requests/
  env/
    dev.yaml                       # committed, non-secret vars
    dev.secrets.yaml                # gitignored — real secret values
    dev.secrets.yaml.example        # committed template, shape only
    acceptance.yaml
    acceptance.secrets.yaml
    acceptance.secrets.yaml.example
    production.yaml
    production.secrets.yaml
    production.secrets.yaml.example
  httpbin/
    get.yaml
    post-json.yaml
    headers.yaml
```

Requests are organized into folders however you like under `requests/`
(other than the reserved `env/` name) — folder structure becomes the
sidebar tree.

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

### Env file + secrets

Each environment is a plain YAML file (`requests/env/dev.yaml`) for
non-secret values:

```yaml
baseUrl: "https://httpbin.org"
```

Real secrets (API keys, tokens) go in a sibling `dev.secrets.yaml` — same
name, `.secrets.yaml` suffix — which is merged on top at load time:

```yaml
token: "sk-real-key-here"
```

`requests/env/*.secrets.yaml` is gitignored by default, so the real values
never get committed. A `dev.secrets.yaml.example` with placeholder values
*is* committed, so teammates know which keys to fill in — copy it to
`dev.secrets.yaml` and drop in real values locally.

Any `{{varName}}` in a request's `url`, `headers`, `query`, or `body` gets
substituted with the merged value (base env + secrets) from the active
environment. Unresolved variables are left as `{{varName}}` in the output
so you can spot typos.

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
