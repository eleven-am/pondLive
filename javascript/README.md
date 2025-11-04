# PondLive

PondLive is a lightweight and user-friendly server-side web framework that utilizes the power of [PondSocket](https://github.com/Eleven-am/pondSocket) to create a simple yet powerful Single-Page Application (SPA) framework.

## Installation

PondLive is available as a Node.js module through the npm registry. To install it, use the following command:

```bash
npm install @eleven-am/pondlive
```

## Introduction

PondLive is a server-side framework that enables rendering HTML pages with real-time functionality directly to the browser, eliminating the need for a separate frontend framework. It leverages PondSocket to provide real-time features to your application.

## Usage

To get started with PondLive, you can use the following example code:

```javascript
import { useState, html, Router } from '@eleven-am/pondlive';

function Counter(ctx) {
  const [count, setCount] = useState(ctx, 0);

  return html`
    <div>
      <h1>Counter</h1>
      <p>Count: ${count}</p>
      <button pond-click=${setCount((count) => count + 1)}>Increment</button>
      <button pond-click=${setCount((count) => count - 1)}>Decrement</button>
    </div>
  `;
}

const router = new Router();

router.mount('/', Counter);

router.serve(3000);
```

In this example, we've created a simple counter component using PondLive. The `Counter` function defines the component logic and uses the `useState` hook to manage the component's state. The `html` template function is used to generate the HTML content.

The `Router` class is responsible for handling routing within the PondLive framework. In this example, we create a new router instance and mount the `Counter` component to the root path ('/') using the `mount` function. Finally, we start serving the components on port 3000 with the `serve` function.

## Enhanced Example

Here's an enhanced example that demonstrates additional features of PondLive:

```javascript
import { useState, html, useAction, useRouter } from '@eleven-am/pondlive';

function Counter(ctx) {
  const [count, action] = useAction(ctx, 0, {
    increment: (_, count) => count + 1,
    decrement: (_, count) => count - 1,
  });

  return html`
    <div>
      <h1>Counter</h1>
      <p>Count: ${count}</p>
      <button pond-click=${action.increment}>Increment</button>
      <button pond-click=${action.decrement}>Decrement</button>
    </div>
  `;
}

function Home(ctx) {
  const [name, setName] = useState(ctx, 'World');
  const router = useRouter([
      {
        path: '/counter',
        component: Counter,
      }
  ]);

  return html`
    <div>
      <h1>Home</h1>
      <p>Hello, ${name}!</p>
      <a href="/counter">Counter</a>
      <input type="text" pond-keyup=${setName((_, event) => event.data.value)} pond-value=${name} />
      ${router(ctx)}
    </div>
  `;
}
```

In this example, we've introduced two additional hooks: `useAction` and `useRouter`.

### useAction
The `useAction` hook is used to generate actions for a component when an event is triggered. It returns an action handler that can be invoked with a specific action name. In the example, `useAction` is used to create actions for incrementing and decrementing the count when the corresponding buttons are clicked. The actions are defined as an object mapping action names to

functions that update the count value.

### useRouter
The `useRouter` hook is used to create a router for your application. It returns the router object, which allows you to define routes and mount components. In the example, `useRouter` is used to create a router instance. The `path` property specifies the route path, and the `component` property specifies the component to render when the route is matched.

The `Home` component serves as the main entry point of the application and renders the `Counter` component as a child component when the path '/counter' is accessed. The `router` function is invoked within the template literal `${router(ctx)}` to render the appropriate component based on the current route.

### Styles
Here's a simplified example that demonstrates the usage of `makeStyles` with a dynamic style based on a `number` prop:

```javascript
import { makeStyles, useState, html } from '@eleven-am/pondlive';

const useStyles = makeStyles((number) => ({
  container: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    height: '100vh',
    backgroundColor: number % 2 === 0 ? '#f0f0f0' : '#ffffff',
  },
  title: {
    fontSize: '24px',
    fontWeight: 'bold',
    marginBottom: '16px',
    color: number % 2 === 0 ? '#000000' : '#ff0000',
  },
  button: {
    padding: '8px 16px',
    borderRadius: '4px',
    backgroundColor: number % 2 === 0 ? '#0088cc' : '#00cc88',
    color: '#ffffff',
    cursor: 'pointer',
  },
}));

function MyComponent(ctx) {
    const [number, setNumber] = useState(ctx, 0);
    const classes = useStyles(ctx, number);
    
    return html`
        <div class="${classes.container}">
          <h1 class="${classes.title}">Welcome to My Component</h1>
          <button class="${classes.button}">Click Me</button>
        </div>
  `;
}
```

In this example, the `useStyles` function takes a `number` prop as a parameter and returns an object with dynamically generated CSS classes. The `backgroundColor`, `color`, and `backgroundColor` styles of the `container`, `title`, and `button` classes respectively are conditionally set based on whether the `number` prop is even or odd.

When rendering the `MyComponent`, we pass the `number` prop to the `useStyles` function as an argument, and then use the generated classes inside the HTML template as before.

This example shows how you can create dynamic styles based on props, allowing you to customize the appearance of your components based on different conditions or inputs.

## API

### ServerInfo\<T\>
The `ServerInfo` class provides a mechanism for managing a shared state object that can be accessed by all users. It offers the following functions:
- `getState`: Retrieves the current state of the `ServerInfo` object.
- `setState`: Sets the state of the `ServerInfo` object to a new value.

### ServerContext\<T\>
The `ServerContext` class allows for the management of a state object specific to each user. It provides the following functions:
- `setState`: Sets the state of the `ServerContext` object for a given user.
- `getState`: Retrieves the current state of the `ServerContext` object for a specified user.

### Html
The `Html` class represents an abstraction for generating HTML content within PondLive. It does not have any specific functions but is used in conjunction with other components.

### Router
The `Router` class is responsible for handling routing within the PondLive framework. It offers the following functions:
- `addRoute`: Adds a route to the router, specifying the path and component to be rendered.
- `addStaticRoute`: Adds a directory that should be served statically.
- `serve`: Activates the router to start serving components. Can accept additional arguments.
- `serveWithExpress`: Activates the router to start serving components using Express. Requires specifying the entry point and Express app.

### createServerInfo
The `createServerInfo` function creates a `ServerInfo` object that holds a single state object accessible to all users. It takes the initial state as a parameter.

### createClientContext
The `createClientContext` function creates a `ServerContext` object that holds a state object specific to each user. It takes the initial state as a parameter.

### useServerInfo
The `useServerInfo` hook is used to retrieve the current state of the `ServerInfo` or `ServerContext` object and provides a function to update the state. It takes the `LiveContext` and the `ServerInfo` or `ServerContext` object as parameters.

### useState
The `useState` hook is used to manage the state of a component. It returns the current state value and a function to update the state. The `LiveContext` and the initial state value are required parameters.

### makeStyles
The `makeStyles` function is used to generate CSS styles for a component based on a provided CSS class object or generator. It returns a hook that provides the styles as an object. Accepts either a CSS class object or a CSS generator as a parameter.

### useAction
The `useAction` hook generates actions for a component when an event is triggered. It returns the result of the action. It requires the `LiveContext`, initial state, and a set of actions as parameters.

### useRouter
The `useRouter` hook is used to create a router for an application. It accepts an array of route objects and returns the appropriate route component to render based on the current URL path.

### html
The `html` function is

used to generate HTML content within PondLive. It takes a `TemplateStringsArray` and dynamic values as parameters and returns an `Html` object representing the HTML content.

## LiveContext

### Mount
When a user accesses a route, the component is mounted and the `mount` function is invoked. The `mount` function is responsible for rendering the component and returning the HTML content to the browser. Certain actions can be performed during the mount phase, such as setting the initial state of the component.

```javascript
import { useState, html } from '@eleven-am/pondlive';

function Greeter (ctx) {
    const [name, _, setOnServer] = useState(ctx, 'World');
    
    ctx.onMount((req, res) => {
        const name = req.url.searchParams.get('name') || 'World';
        setOnServer(req, name);
    });

    return html`
        <div>
            <h1>Hello, ${name}!</h1>
        </div>
    `;
}
```

The `onMount` function can be used for performing actions during the mount phase. In this example, the `onMount` function is used to set the initial state of the component based on the query string parameter `name`. It's assumed that during the mount phase, a call to some data store is made to retrieve the initial state of the component.

### Upgrades
When a user finally makes a websocket connection, the component is upgraded, and the `upgrade` function is invoked. The `upgrade` function is useful for making subscriptions to data sources and performing other actions that require a websocket connection.

```javascript
import { html, createServerInfo, useServerInfo } from '@eleven-am/pondlive';

const info = createServerInfo({ count: 0 });
// const info = createClientContext({ count: 0 });

// The createServerInfo function creates a ServerInfo object that holds a single state object accessible to all users.
// The createClientContext function creates a ServerContext object that holds a state object specific to each user.

function Greeter (ctx) {
    const [data, setter] = useServerInfo(ctx, info);
    
    ctx.onUpgrade((event) => {
        // now we have a websocket connection
        // any time any update is made to the state 
        // of the info object, the component will be
        // re-rendered
        info.setState(event, function (state) { 
            return { count: state.count + 1 };
        });
        
        // When inside a context callback the data value may be stale
        // because the component has not yet been re-rendered. To get
        // the latest value, use the getState function.
        
        // const count = info.getState(event).count;
        // setter(event, { count: count + 1 });
    });

    return html`
        <div>
            <h1>Hello, ${name}!</h1>
            <h2> A total of ${data.count} users have visited this page.</h2>
        </div>
    `;
}
```

The `onUpgrade` function can be used for performing actions during the upgrade phase. In this example, the `onUpgrade` function is used to subscribe to the `ServerInfo` object. The component will be re-rendered any time the state of the `ServerInfo` object changes.

### Unmount
When a user leaves a route, the component is unmounted, and the `unmount` function is invoked. The `unmount` function is useful for performing cleanup actions, such as unsubscribing from data sources.

```javascript
import { html, createServerInfo, useServerInfo } from '@eleven-am/pondlive';

const info = createServerInfo({ count: 0 });

function Greeter (ctx) {
    const [data, _, dataEffect] = useServerInfo(ctx, info);
    
    ctx.onUpgrade((event) => {
        // now we have a websocket connection
        // any time any update is made to the state 
        // of the info object, the component will be
        // re-rendered
        info.setState(event, function (state) { 
            return { count: state.count + 1 };
        });
    });
    
    dataEffect((data) => {
        // do something with the data
        
        // The dataEffect function is used to perform actions when the data changes.
        // cleanup actions can be performed by returning a function from the dataEffect
        
        // return () => { /* cleanup actions */ };
    });
    
    ctx.onUnmount((event) => {
        // we are leaving the route
        info.setState(event, function (state) { 
            return { count: state.count - 1 };
        });
    });

    return html`
        <div>
            <h1>Hello, ${name}!</h1>
            <h2> Active users: ${data.count}</h2>
        </div>
    `;
}
```

The `onUnmount` function can be used for performing actions during the unmount phase. In this example , the `onUnmount` function is used to decrement the count of active users when the component is unmounted. This ensures that the count is accurate even when users navigate away from the route.

### Uploading
When a user uploads a file, the `upload` function is invoked. The `upload` function is useful for performing actions when a file is uploaded.

```javascript
import { html, useAction } from '@eleven-am/pondlive';

function Upload (ctx) {
    const [_, action] = useAction(ctx, null, {
        upload: (event) => {
            // event.files contains that are to be uploaded
            // you can perform actions here such as checking if the mimetype is valid
            event.files?.accept('/path/to/save/file/'); // acceepts all files
            // event.files?.files[0].accept(/path/to/save/file/); accepts only the first file
        }
    });
    
    ctx.onUpload((event, files) => {
        // this function is invoked when a file is uploaded
        // you can perform actions here such as getting
        
        files[0].stream.on('data', (chunk) => {
            // do something with the chunk
        });
    });
    
    return html`
        <form pond-upload="${action.upload}">
            <input type="file" name="files" />
            <button type="submit">Upload</button>
        </form>
    `;
}
```
