# Tailwind Counter

A counter styled with TailwindCSS. It shows:

- `UseSignal` for state and `Text(count)` bound to it
- `UseComputed` feeding a signal-valued `UseHead` title
- a click handler through the typed `pkg.OnClick` and one through the generic `pkg.On`

A component body runs once. A click updates the signal, and the server sends two small edits: the count text and the title. Nothing else is rebuilt.

## Run

```bash
cd examples/counter
go run .
```

Open http://localhost:8080. The compiled stylesheet in `public/assets/tailwind.css` is embedded and served from `/assets/`.

Regenerate it after changing classes in `main.go`:

```bash
npm install
npx tailwindcss -i styles/input.css -o public/assets/tailwind.css --minify
```

## Test

```bash
go test ./...
```

`main_test.go` renders the page through `app.Handler()` and checks the SSR output.
