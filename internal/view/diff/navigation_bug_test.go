package diff

import (
	"fmt"
	"testing"

	"github.com/eleven-am/pondlive/internal/view"
)

func TestCoverLettersToCSVNavigationBug(t *testing.T) {
	makeCard := func(href, company, title, date, preview string) *view.Element {
		return withAttr(
			withAttr(
				withChildren(elementNode("a"),
					withChildren(elementNode("div"),
						withChildren(elementNode("span"), textNode(company)),
						textNode("·"),
						withChildren(elementNode("span"), textNode(title)),
					),
					withChildren(elementNode("div"), textNode(date)),
					withChildren(elementNode("div"), textNode(preview)),
				),
				"href", href,
			),
			"class", "block",
		)
	}

	coverLetters := withChildren(
		withAttr(elementNode("div"), "class", "space-y-3"),
		makeCard("/applications/google-senior-engineer", "Google", "Senior Software Engineer", "Dec 14, 2025", "Dear Hiring Manager at Google..."),
		makeCard("/applications/stripe-backend-engineer", "Stripe", "Backend Engineer", "Dec 10, 2025", "Dear Stripe Recruiting Team..."),
		makeCard("/applications/netflix-platform-engineer", "Netflix", "Platform Engineer", "Dec 13, 2025", "Dear Netflix Engineering Team..."),
		makeCard("/applications/apple-ios-engineer", "Apple", "iOS Engineer", "Dec 11, 2025", "Dear Apple Hiring Team..."),
		makeCard("/applications/microsoft-cloud-engineer", "Microsoft", "Cloud Solutions Engineer", "Dec 12, 2025", "Dear Microsoft Azure Team..."),
		makeCard("/cover-letters/cl-generic-backend", "Standalone", "Cover Letter", "Dec 4, 2025", "Dear Hiring Manager..."),
		makeCard("/cover-letters/cl-generic-platform", "Standalone", "Cover Letter", "Oct 30, 2025", "Dear Hiring Team..."),
	)

	cvs := withChildren(
		withAttr(elementNode("div"), "class", "space-y-3"),
		makeCard("/cvs/cv-789", "Master CV", "Senior Software Engineer | Backend & Infrastructure", "Master", "You are..."),
		makeCard("/applications/google-senior-engineer", "Google", "Senior Software Engineer - Google Focused", "Dec 14, 2025", "Skills: Go, Python..."),
		makeCard("/applications/stripe-backend-engineer", "Stripe", "Backend Engineer - Payments Focus", "Dec 10, 2025", "Skills: Java, Kotlin..."),
		makeCard("/applications/netflix-platform-engineer", "Netflix", "Platform Engineer - Streaming Infrastructure", "Dec 9, 2025", "Skills: Kubernetes..."),
	)

	t.Log("Cover Letters page (7 items):")
	for i, child := range coverLetters.Children {
		if a, ok := child.(*view.Element); ok {
			href := a.Attrs["href"]
			t.Logf("  [%d] href=%s", i, href)
		}
	}

	t.Log("\nCVs page (4 items):")
	for i, child := range cvs.Children {
		if a, ok := child.(*view.Element); ok {
			href := a.Attrs["href"]
			t.Logf("  [%d] href=%s", i, href)
		}
	}

	hasCross := hasCrossPositionSignatureMatch(coverLetters.Children, cvs.Children)
	t.Logf("\nhasCrossPositionSignatureMatch: %v", hasCross)

	t.Log("\nSignatures:")
	for i, child := range coverLetters.Children {
		sig := nodeSignature(child)
		t.Logf("  Old[%d] sig=%s", i, sig)
	}
	for i, child := range cvs.Children {
		sig := nodeSignature(child)
		t.Logf("  New[%d] sig=%s", i, sig)
	}

	patches := Diff(coverLetters, cvs)

	t.Logf("\nGenerated %d patches:", len(patches))
	for _, p := range patches {
		switch p.Op {
		case OpSetAttr:
			attrs := p.Value.(map[string][]string)
			t.Logf("  [seq=%d] setAttr path=%v attrs=%v", p.Seq, p.Path, attrs)
		case OpSetText:
			t.Logf("  [seq=%d] setText path=%v value=%q", p.Seq, p.Path, p.Value)
		case OpDelChild:
			t.Logf("  [seq=%d] delChild path=%v idx=%d", p.Seq, p.Path, *p.Index)
		case OpAddChild:
			if elem, ok := p.Value.(*view.Element); ok {
				href := elem.Attrs["href"]
				t.Logf("  [seq=%d] addChild path=%v idx=%d (href=%v)", p.Seq, p.Path, *p.Index, href)
			} else {
				t.Logf("  [seq=%d] addChild path=%v idx=%d", p.Seq, p.Path, *p.Index)
			}
		case OpMoveChild:
			t.Logf("  [seq=%d] moveChild path=%v value=%v", p.Seq, p.Path, p.Value)
		case OpReplaceNode:
			t.Logf("  [seq=%d] replaceNode path=%v", p.Seq, p.Path)
		default:
			t.Logf("  [seq=%d] %s path=%v", p.Seq, p.Op, p.Path)
		}
	}

	expectedFinalState := map[int]string{
		0: "/cvs/cv-789",
		1: "/applications/google-senior-engineer",
		2: "/applications/stripe-backend-engineer",
		3: "/applications/netflix-platform-engineer",
	}

	t.Log("\nVerifying final state after applying patches:")
	t.Log("Starting with Cover Letters (7 items), target is CVs (4 items)")

	for pos, expectedHref := range expectedFinalState {
		t.Logf("  Position %d should have href=%s", pos, expectedHref)
	}

	hrefAtPos0AfterPatches := ""
	for _, p := range patches {
		if p.Op == OpAddChild && len(p.Path) == 0 && *p.Index == 0 {
			if elem, ok := p.Value.(*view.Element); ok {
				hrefAtPos0AfterPatches = elem.Attrs["href"][0]
			}
		}
	}

	if hrefAtPos0AfterPatches == "" {
		t.Log("\nWARNING: No addChild patch at index 0 found!")
		t.Log("This means the diff expects position 0 to stay as /applications/google-senior-engineer")
		t.Log("But CVs page should have /cvs/cv-789 at position 0!")

		for _, p := range patches {
			if p.Op == OpMoveChild {
				moveInfo := p.Value.(map[string]interface{})
				t.Logf("  Found move: %v", moveInfo)
			}
		}
	} else {
		t.Logf("\nPosition 0 will be set to href=%s", hrefAtPos0AfterPatches)
		if hrefAtPos0AfterPatches != "/cvs/cv-789" {
			t.Errorf("Expected /cvs/cv-789 at position 0, got %s", hrefAtPos0AfterPatches)
		}
	}
}

func TestSimulateClientPatchApplication(t *testing.T) {
	makeCard := func(href, content string) *view.Element {
		return withAttr(
			withChildren(elementNode("a"),
				withChildren(elementNode("span"), textNode(content)),
			),
			"href", href,
		)
	}

	coverLetters := withChildren(
		withAttr(elementNode("div"), "class", "space-y-3"),
		makeCard("/applications/google", "Google Cover Letter"),
		makeCard("/applications/stripe", "Stripe Cover Letter"),
		makeCard("/applications/netflix", "Netflix Cover Letter"),
		makeCard("/applications/apple", "Apple Cover Letter"),
	)

	cvs := withChildren(
		withAttr(elementNode("div"), "class", "space-y-3"),
		makeCard("/cvs/master", "Master CV"),
		makeCard("/applications/google", "Google CV"),
		makeCard("/applications/stripe", "Stripe CV"),
	)

	patches := Diff(coverLetters, cvs)

	t.Log("Simulating client-side DOM state:")
	type domNode struct {
		tag      string
		href     string
		children []domNode
		text     string
	}

	clientDOM := []struct {
		href    string
		content string
	}{
		{"/applications/google", "Google Cover Letter"},
		{"/applications/stripe", "Stripe Cover Letter"},
		{"/applications/netflix", "Netflix Cover Letter"},
		{"/applications/apple", "Apple Cover Letter"},
	}

	t.Log("Initial client DOM:")
	for i, node := range clientDOM {
		t.Logf("  [%d] href=%s content=%s", i, node.href, node.content)
	}

	t.Logf("\nApplying %d patches:", len(patches))
	for _, p := range patches {
		switch p.Op {
		case OpDelChild:
			idx := *p.Index
			t.Logf("  delChild at index %d (removing %s)", idx, clientDOM[idx].href)
			clientDOM = append(clientDOM[:idx], clientDOM[idx+1:]...)
		case OpAddChild:
			idx := *p.Index
			elem := p.Value.(*view.Element)
			href := elem.Attrs["href"][0]
			content := "new content"
			if len(elem.Children) > 0 {
				if span, ok := elem.Children[0].(*view.Element); ok && len(span.Children) > 0 {
					if txt, ok := span.Children[0].(*view.Text); ok {
						content = txt.Text
					}
				}
			}
			t.Logf("  addChild at index %d (adding href=%s content=%s)", idx, href, content)
			newNode := struct {
				href    string
				content string
			}{href, content}
			if idx >= len(clientDOM) {
				clientDOM = append(clientDOM, newNode)
			} else {
				clientDOM = append(clientDOM[:idx+1], clientDOM[idx:]...)
				clientDOM[idx] = newNode
			}
		case OpSetAttr:
			if len(p.Path) == 1 {
				idx := p.Path[0]
				attrs := p.Value.(map[string][]string)
				if href, ok := attrs["href"]; ok {
					t.Logf("  setAttr at index %d: href=%s->%s", idx, clientDOM[idx].href, href[0])
					clientDOM[idx].href = href[0]
				}
			}
		case OpSetText:
			pathStr := fmt.Sprintf("%v", p.Path)
			t.Logf("  setText path=%s value=%q", pathStr, p.Value)
		case OpMoveChild:
			moveInfo := p.Value.(map[string]interface{})
			t.Logf("  moveChild: %v", moveInfo)
		}
	}

	t.Log("\nFinal client DOM after patches:")
	for i, node := range clientDOM {
		t.Logf("  [%d] href=%s content=%s", i, node.href, node.content)
	}

	expectedHrefs := []string{"/cvs/master", "/applications/google", "/applications/stripe"}

	if len(clientDOM) != len(expectedHrefs) {
		t.Errorf("Expected %d items, got %d", len(expectedHrefs), len(clientDOM))
	}

	for i, expected := range expectedHrefs {
		if i >= len(clientDOM) {
			t.Errorf("Missing item at position %d", i)
			continue
		}
		if clientDOM[i].href != expected {
			t.Errorf("Position %d: expected href=%s, got href=%s", i, expected, clientDOM[i].href)
		}
	}
}
