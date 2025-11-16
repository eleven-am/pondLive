package router

import (
	"testing"

	h "github.com/eleven-am/pondlive/go/internal/html"
	runtime "github.com/eleven-am/pondlive/go/internal/runtime"
)

type metaProps struct{}

func metadataPage(ctx runtime.Ctx, match Match) h.Node {
	UseMetadata(ctx, &Meta{
		Title: "User " + match.Params["id"],
		Meta: []h.MetaTag{{
			Name:    "description",
			Content: "Profile for user " + match.Params["id"],
		}},
	})
	return runtime.WithMetadata(
		testDiv(h.Text("metadata")),
		&Meta{Links: []h.LinkTag{{Rel: "canonical", Href: "/users/" + match.Params["id"]}}},
	)
}

func metadataApp(ctx runtime.Ctx, _ metaProps) h.Node {
	return Router(ctx,
		Routes(ctx,
			Route(ctx, RouteProps{Path: "/users/:id", Component: metadataPage}),
		),
	)
}

func TestUseMetadataMergesRouteAndResult(t *testing.T) {
	sess := runtime.NewSession(metadataApp, metaProps{})
	InternalSeedSessionLocation(sess, ParseHref("/users/42"))
	sess.InitialStructured()
	meta := sess.Metadata()
	if meta == nil {
		t.Fatal("expected metadata to be captured")
	}
	if meta.Title != "User 42" {
		t.Fatalf("unexpected title %q", meta.Title)
	}
	if len(meta.Meta) != 1 || meta.Meta[0].Name != "description" {
		t.Fatalf("expected description meta tag, got %+v", meta.Meta)
	}
	if len(meta.Links) != 1 || meta.Links[0].Href != "/users/42" {
		t.Fatalf("expected canonical link, got %+v", meta.Links)
	}
}
