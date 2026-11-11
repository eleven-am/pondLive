# Scoped Styles

Two components define the same class names (`.card`, `.title`, `.content`) with different CSS. `UseStyles` scopes each stylesheet, and `style.Class("card")` returns the scoped name.

## Run

```bash
cd examples/scoped-styles
go run .
```

Open http://localhost:8080.

## Test

```bash
go test ./...
```
