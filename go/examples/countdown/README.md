# LiveUI Countdown Timer Example

A simple countdown timer application demonstrating LiveUI's state management and event handling.

## Features

- **Countdown Timer**: Start from 10 seconds and manually count down
- **Interactive Controls**: -1, Reset, and +5s buttons
- **State Management**: Demonstrates `UseState` for tracking countdown value
- **Conditional Rendering**: Disabling buttons based on timer state
- **TailwindCSS Styling**: Modern, responsive UI with visual feedback

## Key Concepts

This example demonstrates:

1. **State Management**: Using `UseState` to manage countdown value
2. **Event Handling**: Click handlers for buttons
3. **Conditional Rendering**: Using `h.If()` to disable buttons conditionally
4. **Dynamic Content**: Updating UI based on state changes

## Running the Example

```bash
cd examples/countdown
go run .
```

Then open http://localhost:8081 in your browser.

## How It Works

The countdown timer uses basic LiveUI primitives:

1. **State**: `seconds` state variable tracks the current countdown value
2. **Decrement**: Clicking "-1" decreases seconds by 1 (disabled at 0)
3. **Reset**: Clicking "Reset" returns to 10 seconds
4. **Add Time**: Clicking "+5s" adds 5 seconds to the current value
5. **Live Updates**: UI automatically updates when state changes via websocket

This is a simple example showing the basics of LiveUI. For automatic countdown functionality, you would need to use client-side JavaScript intervals (which could be added via the `UseScript` hook once it's exported in the public API).

## Code Structure

```go
// State management
seconds, setSeconds := ui.UseState(ctx, 10)

// Event handlers
decrement := func(h.Event) h.Updates {
    if seconds() > 0 {
        setSeconds(seconds() - 1)
    }
    return nil
}

// Conditional rendering
h.If(seconds() == 0, h.Attr("disabled", ""))
```
