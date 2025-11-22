package live

import "github.com/eleven-am/pondlive/go/internal/runtime"

type (
	HandlerFunc   = runtime.HandlerFunc
	HandlerHandle = runtime.HandlerHandle
)

// UseHandler registers an ephemeral HTTP handler scoped to the current component and session.
// The handler is accessible at the URL returned by handle.URL() and persists until the
// component unmounts or the session ends.
//
// The method parameter specifies the HTTP verb (GET, POST, PUT, DELETE, etc.) that this
// handler accepts. Requests with other methods receive a 405 Method Not Allowed response.
//
// The chain parameter accepts one or more HandlerFunc functions that execute in order.
// Each function receives the http.ResponseWriter and *http.Request, and returns an error.
// If any function returns an error and hasn't written a response, a 500 status is sent
// with the error message.
//
// Example - Simple POST handler:
//
//	func ApiComponent(ctx live.Ctx) h.Node {
//	    h := live.UseHandler(ctx, "POST", func(w http.ResponseWriter, r *http.Request) error {
//	        var data MyData
//	        if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
//	            return err
//	        }
//	        result := processData(data)
//	        w.Header().Set("Content-Type", "application/json")
//	        return json.NewEncoder(w).Encode(result)
//	    })
//
//	    // Use h.URL() with UseScript
//	    script := live.UseScript(ctx, fmt.Sprintf(`
//	        (element, transport) => {
//	            element.addEventListener('click', async () => {
//	                const res = await fetch('%s', {
//	                    method: 'POST',
//	                    body: JSON.stringify({foo: 'bar'})
//	                });
//	                const data = await res.json();
//	                transport.send('result', data);
//	            });
//	        }
//	    `, h.URL()))
//
//	    return h.Button(h.Attach(script), h.Text("Call API"))
//	}
//
// Example - Handler with middleware:
//
//	func SecureApi(ctx live.Ctx) h.Node {
//	    authMiddleware := func(w http.ResponseWriter, r *http.Request) error {
//	        token := r.Header.Get("Authorization")
//	        if token == "" {
//	            http.Error(w, "Unauthorized", http.StatusUnauthorized)
//	            return nil // Response written, stop chain
//	        }
//	        return nil // Continue to next handler
//	    }
//
//	    handler := func(w http.ResponseWriter, r *http.Request) error {
//	        w.Write([]byte("Authenticated!"))
//	        return nil
//	    }
//
//	    h := live.UseHandler(ctx, "GET", authMiddleware, handler)
//	    return h.Div(h.Text(fmt.Sprintf("Endpoint: %s", h.URL())))
//	}
func UseHandler(ctx Ctx, method string, chain ...HandlerFunc) HandlerHandle {
	return runtime.UseHandler(ctx, method, chain...)
}
