# Auth

Cookie login and logout with routing. It shows:

- `Routes` and `Route`, with a layout route (`"/*"`) that renders its own relative `Routes`, plus `Link` and `Redirect`
- `UseRequest(ctx).Cookie("session")` as a signal, and `SetCookie` in a handler
- `Show` with `Else` for the logged-in and logged-out header
- `UseRouter(ctx).Navigate` from a handler

Opening `/dashboard` without the cookie answers with an HTTP redirect to `/login`, and opening `/login` with the cookie redirects to `/dashboard`.

## Run

```bash
cd examples/auth
go run .
```

Open http://localhost:8080 and sign in with any username.

## Test

```bash
go test ./...
```
