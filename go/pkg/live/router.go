package live

import (
	"net/url"

	"github.com/eleven-am/pondlive/go/internal/runtime"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type (
	Location   = runtime.Location
	RouteProps = runtime.RouteProps
	LinkProps  = runtime.LinkProps
	Match      = runtime.Match
	NavMsg     = runtime.NavMsg
	PopMsg     = runtime.PopMsg
)

var (
	Parse            = runtime.Parse
	NormalizePattern = runtime.NormalizePattern
	Prefer           = runtime.Prefer
	BestMatch        = runtime.BestMatch
	BuildHref        = runtime.BuildHref
	SetSearch        = runtime.SetSearch
	AddSearch        = runtime.AddSearch
	DelSearch        = runtime.DelSearch
	MergeSearch      = runtime.MergeSearch
	ClearSearch      = runtime.ClearSearch
	ParseHref        = runtime.ParseHref
	ErrMissingRouter = runtime.ErrMissingRouter
)

func Router(ctx Ctx, children ...Node) Node {
	return runtime.Router(ctx, children...)
}

func Routes(ctx Ctx, children ...Node) Node {
	return runtime.Routes(ctx, children...)
}

func Route(ctx Ctx, props RouteProps, children ...Node) Node {
	return runtime.Route(ctx, props, children...)
}

func Outlet(ctx Ctx) Node {
	return runtime.Outlet(ctx)
}

func Link(ctx Ctx, props LinkProps, children ...h.Item) Node {
	return runtime.RouterLink(ctx, props, children...)
}

func Navigate(ctx Ctx, href string) {
	runtime.RouterNavigate(ctx, href)
}

func Replace(ctx Ctx, href string) {
	runtime.RouterReplace(ctx, href)
}

func NavigateWithSearch(ctx Ctx, patch func(url.Values) url.Values) {
	runtime.RouterNavigateWithSearch(ctx, patch)
}

func ReplaceWithSearch(ctx Ctx, patch func(url.Values) url.Values) {
	runtime.RouterReplaceWithSearch(ctx, patch)
}

func Redirect(ctx Ctx, to string) Node {
	return runtime.RouterRedirect(ctx, to)
}

// UseLocation returns the current router location including pathname, search params, and hash.
//
// Example:
//
//	func CurrentPage(ctx live.Ctx) h.Node {
//	    loc := live.UseLocation(ctx)
//
//	    return h.Div(
//	        h.Text(fmt.Sprintf("Current path: %s", loc.Pathname)),
//	        h.Text(fmt.Sprintf("Search: %s", loc.Search)),
//	    )
//	}
func UseLocation(ctx Ctx) Location {
	return runtime.UseLocation(ctx)
}

// UseParams returns all route parameters extracted from the current URL pattern.
//
// Example:
//
//	// Route pattern: "/users/:userID/posts/:postID"
//	// Current URL: "/users/123/posts/456"
//
//	func PostDetail(ctx live.Ctx) h.Node {
//	    params := live.UseParams(ctx)
//	    userID := params["userID"]  // "123"
//	    postID := params["postID"]  // "456"
//
//	    return h.Div(
//	        h.Text(fmt.Sprintf("User: %s, Post: %s", userID, postID)),
//	    )
//	}
func UseParams(ctx Ctx) map[string]string {
	return runtime.UseParams(ctx)
}

// UseParam returns a single route parameter by key. Returns empty string if not found.
//
// Example:
//
//	// Route pattern: "/users/:userID"
//	// Current URL: "/users/123"
//
//	func UserProfile(ctx live.Ctx) h.Node {
//	    userID := live.UseParam(ctx, "userID")  // "123"
//
//	    user, _ := live.UseState(ctx, User{})
//
//	    live.UseEffect(ctx, func() live.Cleanup {
//	        u, err := fetchUser(userID)
//	        if err == nil {
//	            user(u)
//	        }
//	        return nil
//	    }, userID)
//
//	    return h.Div(h.Text(user().Name))
//	}
func UseParam(ctx Ctx, key string) string {
	return runtime.UseParam(ctx, key)
}

// UseSearch returns the current URL search/query parameters as url.Values.
//
// Example:
//
//	// Current URL: "/products?category=electronics&sort=price&page=2"
//
//	func ProductList(ctx live.Ctx) h.Node {
//	    search := live.UseSearch(ctx)
//	    category := search.Get("category")  // "electronics"
//	    sort := search.Get("sort")          // "price"
//	    page := search.Get("page")          // "2"
//
//	    // Get multi-value params
//	    tags := search["tag"]  // []string{"new", "sale"}
//
//	    return h.Div(
//	        h.Text(fmt.Sprintf("Category: %s, Sort: %s, Page: %s", category, sort, page)),
//	    )
//	}
func UseSearch(ctx Ctx) url.Values {
	return runtime.UseSearch(ctx)
}

// UseSearchParam returns reactive getter/setter for a specific search parameter.
// Setting the value updates the URL and triggers a render.
//
// Example - Single value parameter:
//
//	func SearchBox(ctx live.Ctx) h.Node {
//	    query, setQuery := live.UseSearchParam(ctx, "q")
//
//	    return h.Input(
//	        h.Type("search"),
//	        h.Value(strings.Join(query(), "")),
//	        h.OnInput(func(evt h.InputEvent) h.Updates {
//	            setQuery([]string{evt.Value})  // Updates URL to ?q=newvalue
//	            return nil
//	        }),
//	    )
//	}
//
// Example - Multi-value parameter:
//
//	func FilterTags(ctx live.Ctx) h.Node {
//	    tags, setTags := live.UseSearchParam(ctx, "tag")
//	    currentTags := tags()  // ["electronics", "sale"]
//
//	    return h.Div(
//	        h.Button(
//	            h.OnClick(func() h.Updates {
//	                // Add a new tag
//	                newTags := append(currentTags, "featured")
//	                setTags(newTags)  // Updates URL to ?tag=electronics&tag=sale&tag=featured
//	                return nil
//	            }),
//	            h.Text("Add Featured Tag"),
//	        ),
//	    )
//	}
func UseSearchParam(ctx Ctx, key string) (func() []string, func([]string)) {
	return runtime.UseSearchParam(ctx, key)
}

func LocEqual(a, b Location) bool {
	return runtime.LocEqual(a, b)
}

// UseMetadata sets document metadata (title, meta tags, etc.) for the current page.
// Useful for SEO and dynamic page titles in single-page applications.
//
// Example - Set page title:
//
//	func BlogPost(ctx live.Ctx) h.Node {
//	    post, _ := live.UseState(ctx, Post{})
//	    postID := live.UseParam(ctx, "postID")
//
//	    live.UseEffect(ctx, func() live.Cleanup {
//	        p, err := fetchPost(postID)
//	        if err == nil {
//	            post(p)
//	        }
//	        return nil
//	    }, postID)
//
//	    live.UseMetadata(ctx, &live.Meta{
//	        Title: post().Title + " - My Blog",
//	    })
//
//	    return h.Article(
//	        h.H1(h.Text(post().Title)),
//	        h.P(h.Text(post().Content)),
//	    )
//	}
//
// Example - Full metadata with Open Graph tags:
//
//	func ProductPage(ctx live.Ctx) h.Node {
//	    product, _ := live.UseState(ctx, Product{})
//
//	    live.UseMetadata(ctx, &live.Meta{
//	        Title:       product().Name,
//	        Description: product().Description,
//	        Tags: []live.MetaTag{
//	            {Property: "og:title", Content: product().Name},
//	            {Property: "og:description", Content: product().Description},
//	            {Property: "og:image", Content: product().ImageURL},
//	            {Property: "og:type", Content: "product"},
//	        },
//	    })
//
//	    return h.Div(/* render product */)
//	}
func UseMetadata(ctx Ctx, meta *Meta) {
	runtime.UseMetadata(ctx, meta)
}
