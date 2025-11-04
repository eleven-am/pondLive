# LiveUI Client

TypeScript client for LiveUI server-driven reactive UI framework.

## Features

- **Server-driven UI**: UI state and rendering logic live on the server
- **Efficient patching**: Only dynamic slots are updated, not the entire DOM
- **WebSocket communication**: Real-time updates via PondSocket
- **Event delegation**: Automatic event handling and forwarding to server
- **Type-safe**: Written in TypeScript with full type definitions
- **Reconnection support**: Automatic reconnection and session resume

## Installation

```bash
npm install
```

## Development

### Build for development (with watch mode)

```bash
npm run dev
```

This watches for changes and rebuilds automatically, outputting to `../pkg/liveui/server/static/js/`.

### Build for production

```bash
npm run build:prod
```

This creates an optimized production build without source maps.

### Type checking

```bash
npm run typecheck
```

### Clean build artifacts

```bash
npm run clean
```

## Architecture

### Modules

- **index.ts**: Main LiveUI client class that manages connections and coordination
- **dom-index.ts**: Tracks dynamic slot anchors and keyed list containers
- **patcher.ts**: Applies diff operations to the DOM
- **events.ts**: Event delegation and forwarding to server
- **types.ts**: TypeScript type definitions for protocol messages

### Protocol

The client communicates with the server using JSON messages over WebSocket:

**Server → Client**:
- `init`: Initial session setup
- `frame`: DOM updates (patches, effects, navigation)
- `join`: Session join confirmation
- `resume`: Resume after reconnection
- `error`: Error messages

**Client → Server**:
- `evt`: User events (clicks, inputs, etc.)
- `ack`: Acknowledgement of received frames
- `nav`: Navigation requests
- `pop`: Popstate events

## Usage

### Basic Setup

```typescript
import LiveUI from '@liveui/client';

// Create and connect the client
const client = new LiveUI({
  endpoint: '/live',
  debug: true,
  autoConnect: true
});

await client.connect();
```

### Browser Usage

The client is also exposed on the global `window` object:

```html
<script src="/pondlive.js"></script>
<script>
  const client = new window.LiveUI({
    endpoint: '/live',
    debug: true
  });
  client.connect();
</script>
```

### SSR Hydration

The client includes DOM utilities for SSR hydration:

```javascript
// Register dynamic slots during hydration
document.querySelectorAll('[data-slot-index]').forEach(el => {
  const index = parseInt(el.getAttribute('data-slot-index'));
  window.LiveUI.dom.registerSlot(index, el);
});
```

## Development Notes

### Custom Patcher vs morphdom

This client uses a custom patcher optimized for LiveUI's structured rendering approach. Unlike general-purpose DOM diffing libraries like morphdom:

- The server pre-computes minimal diff operations
- Only dynamic slots are tracked and updated
- List operations (insert/delete/move) are handled efficiently
- Smaller bundle size (~200 lines total)

### Event Handling

Events are delegated at the document level and forwarded to the server only when a registered handler exists. Handler IDs are embedded in the DOM as `data-on{event}` attributes.

### Type Safety

All protocol messages are fully typed in `types.ts`, ensuring type safety across the client-server boundary.

## Build Output

The build process outputs:
- `../pkg/liveui/server/static/js/index.js` - Main bundle
- `../pkg/liveui/server/static/js/index.js.map` - Source maps (dev build only)

## License

ISC
