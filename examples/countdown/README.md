# Countdown Timer

A ten second countdown driven by a client-side timer. It shows:

- `UseScript`: the browser runs the interval and sends `done` back to Go
- `UseSignal` and `UseComputed` for the running and done state
- `pkg.Disabled(computed)` and a signal-valued `Class` part for the status colour

The timer text is owned by the script, so the body renders it once as static text.

## Run

```bash
cd examples/countdown
go run .
```

Open http://localhost:8080.

The script is built with `new Function` in the browser, so a Content-Security-Policy for this page must allow `unsafe-eval`.

## Test

```bash
go test ./...
```

`main_test.go` renders the page and checks the ready state: the title, the status text and that only Stop is disabled.
