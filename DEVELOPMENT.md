# Development

## Prerequisites

- [Go](https://go.dev/) 1.27.4 (the version in `go.mod`)
- [Node.js](https://nodejs.org/) 20+
- The [Wails v2](https://wails.io/) CLI, at the version used in `go.mod`:

  ```zsh
  $ go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.1
  ```

- On Linux: GTK 3 and WebKitGTK 4.1 development packages (e.g. `libgtk-3-dev libwebkit2gtk-4.1-dev` on
  Debian/Ubuntu, `gtk3 webkit2gtk-4.1` on Arch). `wails doctor` checks everything is in place.

## Getting started

```zsh
$ wails dev
```

This builds and runs Wombat in development mode: the frontend is served by the Vite dev server with hot reload, and the
Go side is rebuilt and restarted when a `.go` file changes. Open http://localhost:34115 to use the app in your browser
(with its dev-tools) while still calling the Go backend.

`wails.json` sets the `webkit2_41` build tag, so `wails dev` and `wails build` link against WebKitGTK 4.1 on Linux.
When running Go tooling directly, pass it yourself:

```zsh
$ (cd frontend && npm install && npm run build)   # main.go embeds frontend/dist
$ go test -race -tags webkit2_41 ./...
$ golangci-lint run                                 # gofmt, vet, errcheck, staticcheck, ... (.golangci.yml)
```

CI runs [golangci-lint](https://golangci-lint.run/) v2.15.0; install the same version locally to get the same results.

## App data

Workspaces and their settings are stored in a Badger database:

| Mode | Location |
| --- | --- |
| `wails dev` | `.data/` in the repository |
| Linux | `~/.local/share/Wombat` |
| macOS | `~/Library/Application Support/Wombat` |
| Windows | `%AppData%\Wombat` |

Set `WOMBAT_DATA_DIR` to use a different directory. Unexpected panics are appended to `crash.log` in the same directory.

Only one instance of Wombat can run at a time (the database is locked); starting it again brings the running window to
the front. A `wails dev` instance and an installed release use different data directories, so they can run side by
side.

## Built-in gRPC server

During development it's common to want to test proto files and gRPC APIs. `wails dev` automatically starts a gRPC
server with reflection on `localhost:5001` (it is not part of release builds). You can also make changes to the
`/internal/server/foobar.proto` file to add test protos.

## Backend API

The Go backend has a singular API entry: `app.api{}`; All public methods defined on the `api` struct are exposed to the
frontend.

Wombat follows a (loose) [flux](https://facebook.github.io/flux/) like architecture. API calls are made to the backend
(actions) and the backend will then emit events (dispatcher) for any changes required by the frontend.

Although the backend APIs can responed with an object/error; the frontend mostly ignores these objects/errors (uncaught promise error), and
rather responds to events including error events.

All backend event constants are defined in `/internal/app/events.go`.

## Frontend

The frontend is built with [Svelte](https://svelte.dev/) 5 (the components still use the Svelte 4 syntax, which Svelte 5
supports); and should be fairly straight forward if you have done any
type of web-based frontend development before.

The frontend is bundled with [Vite](https://vitejs.dev/). `wails dev` starts the Vite dev server and proxies it, so
changes to the frontend are hot reloaded; `wails build` runs `npm run build` and embeds `frontend/dist`.

During development, you can use any browser (and it's dev-tools) by opening http://localhost:34115 while `wails dev` is
running, but it's important to remember that Wombat does not bundle a browser (like Chromium), but rather uses the
system's built-in web view component: WKWebView on macOS, WebKitGTK on Linux and WebView2 (Chromium) on Windows. All of
them are evergreen, so modern JS and CSS (including CSS variables) can be used without transpiling or polyfills.

The theme colours are CSS variables declared in `frontend/src/views/App.svelte`. The UI zoom uses CSS `zoom`; see
`frontend/src/zoom.js` for the two things to keep in mind when writing CSS.

### UI tests

`frontend/tests` holds [Playwright](https://playwright.dev/) tests that run the production build of the frontend
against a fake backend (`tests/fake-backend.js`), in WebKit (the engine of WebKitGTK and WKWebView) and Chromium (close
to WebView2). They cover every field type, sending, saved requests and metadata, method selection, grpcurl
import/export, the dialogs and zoom.

```zsh
$ cd frontend
$ npx playwright install webkit chromium   # once
$ npm test                                 # both engines
$ npm run test:chromium                    # Chromium only
$ npx playwright test --ui                 # interactive runner
$ npm run test:dev                         # against the dev server: also fails on Svelte runtime warnings
```

Playwright's WebKit build needs Ubuntu/Debian libraries. On other distributions run the WebKit tests in Playwright's
container instead (needs Docker): `npm run test:webkit:docker`.

The fake backend's schema, `tests/fixtures/schema.json`, is generated from the real Go backend so the two cannot drift
apart; `go test` fails when it is out of date. Regenerate it after changing the test protos or the method/field
descriptions:

```zsh
$ go test -tags webkit2_41 ./internal/app -run TestFrontendFixtures -update-fixtures
```

After getting your UI correct; it's always good to validate in a real build:

```zsh
$ wails build -debug
$ ./build/bin/wombat
```

`-debug` keeps debug logging and the web inspector ("Right/Option Click" -> "Inspect Element"), which are disabled in
production builds. Note that the built-in gRPC server only runs under `wails dev`.

## Releasing

1. Move the `[Unreleased]` entries in `CHANGELOG.md` into a new section, e.g. `## [v0.6.0] - 2026-11-13`.
2. Tag the commit and push the tag: `git tag -a v0.6.0 -m "Wombat v0.6.0" && git push origin v0.6.0`.

The tag runs `.github/workflows/package.yml`, which builds Linux and Windows packages and attaches
them to a **draft** GitHub release whose text is the tag's `CHANGELOG.md` section (the workflow fails if there is none).
Review and publish the draft on GitHub. Tags with a suffix (`v0.6.0-rc1`) become pre-releases.

The tag becomes the app version. The in-app update check compares it against the latest published, non-pre-release
release of the repository set in `internal/app/update_check.go`.

## Changelog

We maintain a changelog for all versions of Wombat. If you create a PR please add the change to `CHANGELOG.md`. Don't
forget to add your GitHub handle to make sure you get the credit!
