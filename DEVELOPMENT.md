# Development

## Prerequisites 

- Working [Go](https://golang.org/) 1.15+ environment
- Working [Node.js](https://nodejs.org/en/) v12+ environment

## Getting started

Wombat is built with [Wails](https://wails.app/); to install the CLI:

```zsh
$ go get github.com/wailsapp/wails/cmd/wails
```

Wails has a "Bridge Mode" that allows you to debug with your browser of choice; to enable:

```zsh
$ wails serve
```

In a separate terminal you will need to start the frontend:

```zsh
$ cd frontend
$ npm run dev
```

The frontend will automatically re-build and reload on changes.

## Built-in gRPC server

During development it's common to want to test proto files and gRPC APIs. When Wombat lanuches in "Bridge/Debug Mode" it
automatically starts a gRPC server running on `localhost:5001`. You can also make changes to the
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

The frontend is built with [Svelte](https://svelte.dev/); and should be fairly straight forward if you have done any
type of web-based frontend development before.

The frontend is bundled with [Vite](https://vitejs.dev/). `wails dev` starts the Vite dev server and proxies it, so
changes to the frontend are hot reloaded; `wails build` runs `npm run build` and embeds `frontend/dist`.

During development, you can use any browser (and it's dev-tools) by opening http://localhost:34115 while `wails dev` is
running, but it's important to remember that Wombat does not bundle a browser (like Chromium), but rather uses the
system's built-in web view component: WKWebView on macOS, WebKitGTK on Linux and WebView2 (Chromium) on Windows. All of
them are evergreen, so modern JS and CSS (including CSS variables) can be used without transpiling or polyfills.

The theme colours are CSS variables declared in `frontend/src/views/App.svelte`. The UI zoom uses CSS `zoom`; see
`frontend/src/zoom.js` for the two things to keep in mind when writing CSS.

After getting your UI correct; it's always good to validate in a real build:

```zsh
$ wails build -d
$ ./build/wombat
```

The `-d` flag will build Wombat in debug mode so you will still get all the debug output in the terminal, but also runs
the development gRPC server too.

In debug mode you can still "Right/Option Click" -> "Inspect Element" and bring up the development tools (mac and linux
only). This is disabled in production builds.

## Changelog

We maintain a changelog for all versions of Wombat. If you create a PR please add the change to `CHANGELOG.md`. Don't
forget to add your GitHub handle to make sure you get the credit!
