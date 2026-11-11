package livetest_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/pkg"
	"github.com/eleven-am/pondlive/pkg/livetest"
)

var form = pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
	name := pkg.UseSignal(ctx, "")
	agree := pkg.UseSignal(ctx, false)
	colors := pkg.UseSignal(ctx, []string{"red"})
	submitted := pkg.UseSignal(ctx, "")
	summary := pkg.UseComputed(ctx, func() string {
		return "name=" + name.Get() + " agree=" + map[bool]string{true: "yes", false: "no"}[agree.Get()] + " colors=" + strings.Join(colors.Get(), "+")
	})
	submit := func(pkg.ClickEvent) {
		submitted.Set(name.Peek())
		name.Set("")
	}
	return pkg.Div(
		pkg.Input(pkg.ID("name"), pkg.Type("text"), pkg.Bind(name)),
		pkg.Input(pkg.ID("agree"), pkg.Type("checkbox"), pkg.Bind(agree)),
		pkg.Select(pkg.ID("colors"), pkg.Multiple(true), pkg.Bind(colors),
			pkg.Option(pkg.Attr("value", "red"), pkg.Text("Red")), pkg.Option(pkg.Attr("value", "blue"), pkg.Text("Blue"))),
		pkg.P(pkg.ID("summary"), pkg.Text(summary)),
		pkg.P(pkg.ID("sent"), pkg.Text(submitted)),
		pkg.Button(pkg.ID("go"), pkg.OnClick(submit), pkg.Text("go")),
	)
})

func TestATwoWayFormRoundTrip(t *testing.T) {
	page := livetest.Mount(t, form)

	if echoed := page.ByID("name").Type("Ada"); echoed {
		t.Fatal("typing must not be echoed back to the field that sent it")
	}
	page.ByID("agree").Check(true)
	page.ByID("colors").Choose("red", "blue")
	if got := page.HTML(); !strings.Contains(got, "name=Ada agree=yes colors=red+blue") {
		t.Fatalf("html = %s", got)
	}

	page.ByID("go").Click()
	got := page.HTML()
	if !strings.Contains(got, `<p id="sent">Ada</p>`) || !strings.Contains(got, "name= agree=yes") {
		t.Fatalf("submit must read the typed value and clear the signal: %s", got)
	}
}

func TestClearingWithSetReachesTheFieldAndTypingAgainStillWorks(t *testing.T) {
	var draft *pkg.Signal[string]
	root := pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
		draft = pkg.UseSignal(ctx, "")
		return pkg.Div(pkg.Input(pkg.ID("f"), pkg.Bind(draft)), pkg.P(pkg.ID("echo"), pkg.Text(draft)))
	})
	page := livetest.Mount(t, root)
	field := page.ByID("f")
	field.Type("hello")
	page.Do(func() { draft.Set("") })
	if strings.Contains(page.HTML(), "hello") {
		t.Fatalf("draft.Set(\"\") must clear the field: %s", page.HTML())
	}
	field.Type("again")
	if !strings.Contains(page.HTML(), `<p id="echo">again</p>`) {
		t.Fatalf("html = %s", page.HTML())
	}
}

func TestPushedMediaAndElementValuesFeedTheSignals(t *testing.T) {
	root := pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
		video := pkg.UseRef[pkg.MediaRef](ctx)
		box := pkg.UseRef[pkg.ElementRef](ctx)
		time := video.CurrentTime()
		rect := box.Rect()
		label := pkg.UseComputed(ctx, func() string {
			return strings.Join([]string{"t=" + ftoa(time.Get()), "w=" + ftoa(rect.Get().Width)}, " ")
		})
		return pkg.Div(pkg.Video(pkg.ID("v"), pkg.Attach(video)), pkg.Div(pkg.ID("box"), pkg.Attach(box)), pkg.P(pkg.Text(label)))
	})
	page := livetest.Mount(t, root)
	page.ByID("v").PushCurrentTime(12.5)
	page.ByID("box").PushRect(pkg.DOMRect{Width: 320, Height: 40})
	if got := page.HTML(); !strings.Contains(got, "t=12.5 w=320") {
		t.Fatalf("html = %s", got)
	}
}

func ftoa(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

func TestOnChangeReceivesTheTypedValueAndCheckedAndBindIsAlreadyUpdated(t *testing.T) {
	var log []string
	root := pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
		name := pkg.UseSignal(ctx, "")
		agree := pkg.UseSignal(ctx, false)
		return pkg.Div(
			pkg.Input(pkg.ID("name"), pkg.Bind(name), pkg.OnChange(func(e pkg.ChangeEvent) {
				log = append(log, "name:"+e.Value+"|"+name.Peek())
			})),
			pkg.Input(pkg.ID("agree"), pkg.Type("checkbox"), pkg.Bind(agree), pkg.OnChange(func(e pkg.ChangeEvent) {
				log = append(log, "agree:"+map[bool]string{true: "on", false: "off"}[e.Checked])
			})),
		)
	})
	page := livetest.Mount(t, root)
	page.ByID("name").Type("Ada")
	page.ByID("agree").Check(true)
	page.ByID("agree").Check(false)
	if strings.Join(log, ",") != "name:Ada|Ada,agree:on,agree:off" {
		t.Fatalf("log = %v", log)
	}
}

func TestBindOpenFollowsTheUserAndTheServer(t *testing.T) {
	var open *pkg.Signal[bool]
	root := pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
		open = pkg.UseSignal(ctx, false)
		return pkg.Div(pkg.Details(pkg.ID("d"), pkg.Bind(open), pkg.Summary(pkg.Text("more"))), pkg.P(pkg.ID("state"), pkg.Text(open)))
	})
	page := livetest.Mount(t, root)
	if echoed := page.ByID("d").Toggle(true); echoed {
		t.Fatal("the toggling element must not get its own state back")
	}
	if !strings.Contains(page.HTML(), `<p id="state">true</p>`) {
		t.Fatalf("html = %s", page.HTML())
	}
	page.Do(func() { open.Set(false) })
	if strings.Contains(page.HTML(), "<details") && strings.Contains(page.HTML(), " open") {
		t.Fatalf("a server write must close it: %s", page.HTML())
	}
}

func TestTypingIsReportedAsEchoedWhenTheServerRewritesTheValue(t *testing.T) {
	root := pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
		name := pkg.UseSignal(ctx, "")
		return pkg.Input(pkg.ID("name"), pkg.Bind(name), pkg.OnChange(func(e pkg.ChangeEvent) {
			name.Set(strings.ToUpper(e.Value))
		}))
	})
	page := livetest.Mount(t, root)

	if !page.ByID("name").Type("ada") {
		t.Fatalf("a value the server rewrote must be echoed back: %s", page.HTML())
	}
}

var userPage = pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
	router := pkg.UseRouter(ctx)
	req := pkg.UseRequest(ctx)
	label := pkg.UseComputed(ctx, func() string { return "user=" + router.Params.Get()["id"] + " at=" + req.URL().Get().Path })
	return pkg.Div(
		pkg.P(pkg.ID("who"), pkg.Text(label)),
		pkg.Button(pkg.ID("next"), pkg.OnClick(func(pkg.ClickEvent) { router.Navigate("/users/9") }), pkg.Text("next")),
	)
})

var homePage = pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
	return pkg.P(pkg.ID("home"), pkg.Text("home"))
})

var routedApp = pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
	return pkg.Routes(pkg.Route("/", homePage), pkg.Route("/users/:id", userPage))
})

func TestAnAppUsingRoutesAndUseRequestCanBeMountedAtAStartURL(t *testing.T) {
	page := livetest.Mount(t, routedApp, livetest.WithURL("/users/7"))
	if got := page.HTML(); !strings.Contains(got, "user=7 at=/users/7") {
		t.Fatalf("html = %s", got)
	}
	page.ByID("next").Click()
	if got := page.HTML(); !strings.Contains(got, "user=9 at=/users/9") {
		t.Fatalf("a live navigation did not reach the route: %s", got)
	}
}

func TestMountedRoutesStartAtTheRootByDefaultAndInProd(t *testing.T) {
	if got := livetest.MountProd(t, routedApp).HTML(); !strings.Contains(got, `<p id="home">home</p>`) {
		t.Fatalf("html = %s", got)
	}
}

func TestDiagnosticsAreReturnedAsAPublicType(t *testing.T) {
	var diags []pkg.Diagnostic = livetest.Mount(t, homePage).Diagnostics("nothing")
	if len(diags) != 0 {
		t.Fatalf("diagnostics = %v", diags)
	}
}
