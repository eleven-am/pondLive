# Tailwind Counter Example

This example boots a complete LiveUI application with TailwindCSS styling. It
shows how to:

- render a component that uses Tailwind utility classes
- serve the SSR boot payload with `internal/server/http.Manager`
- attach a PondSocket endpoint so websocket diffs reach the browser
- compile Tailwind locally so no CDN requests are required at runtime

## Running the example

```bash
cd examples/counter
go run .
```

The server listens on [http://localhost:8080](http://localhost:8080). The
precompiled Tailwind subset from `public/tailwind.css` is embedded and served so
styling works without reaching out to a CDN.

The checked-in `public/tailwind.css` is generated, not handwritten. It is the
exact output of the Tailwind CLI scanning the Go components referenced in this
example. Regenerate it whenever you tweak the Go markup or Tailwind config:

```bash
npm install
npx tailwindcss -i styles/input.css -o public/tailwind.css --minify
```

The bundled `tailwind.config.js` is configured to scan `*.go` files so Tailwind
detects the class names embedded in component definitions.

## Smoke-testing the HTTP boot endpoint

You can verify the SSR response with a simple HTTP request:

```bash
curl -s http://localhost:8080/ | head
```

The output contains the stylesheet link and the LiveUI boot payload.

Automated coverage lives in `main_test.go`, which uses the HTTP manager directly
to assert on the rendered HTML.
