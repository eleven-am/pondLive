# client/js

Browser helpers that hydrate structured renders and apply patches.

- `patcher.js` owns the patch protocol entry point (`LiveUI.apply`) and delegates
  DOM bookkeeping to `dom-index.js`.
- `dom-index.js` keeps track of dynamic slot anchors and keyed list containers so
  list operations can insert, remove, and reorder DOM regions without full
  re-renders.
- `events.js` (planned) will attach delegated listeners and forward event
  payloads back to the server; the thin slice currently focuses on diff/patch
  support.

The client assumes SSR markup registers slot anchors during bootstrapping and
that keyed rows expose `data-row-key` attributes produced during finalization.
