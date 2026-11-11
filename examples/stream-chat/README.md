# Stream Chat

A chat list built with `For`. It shows:

- `For(Items(messages, key), row)`: a retained list where each insert, prepend and clear is one exact edit
- `Show` for the empty notice
- a two-way bound input: `pkg.Bind(draft)` keeps the `draft` signal current as the user types, and `draft.Set("")` clears the field
- `lastID` and `time.Now()` used only inside handlers, so a rebuild renders the same tree

To keep the list out of memory, replace `pkg.Items(messages, messageKey)` with `pkg.NewFeed(messageKey, load)` and push with `feed.Append`. The row function is unchanged.

## Run

```bash
cd examples/stream-chat
go run .
```

Open http://localhost:8080.

## Test

```bash
go test ./...
```
