package effectguard

import (
	"errors"
	"sync"
	"testing"
)

func TestRefuseOutsideAScopeIsFalseAndFree(t *testing.T) {
	if Refuse("x") {
		t.Fatal("refused with no scope")
	}
	if Active() {
		t.Fatal("active with no scope")
	}
}

func TestDevScopePanicsWithTheComponentAndTheCall(t *testing.T) {
	exit := Enter(&Scope{Component: "Page", Dev: true})
	defer exit()
	defer func() {
		var fc *ForbiddenCallError
		r, _ := recover().(error)
		if !errors.As(r, &fc) || fc.Op != "Shared.Set" || fc.Component != "Page" {
			t.Fatalf("recovered %v", r)
		}
	}()
	Refuse("Shared.Set")
	t.Fatal("did not panic")
}

func TestProdScopeDropsAndReports(t *testing.T) {
	var got string
	exit := Enter(&Scope{Component: "Page", Dropped: func(op, c string) { got = op + "@" + c }})
	if !Refuse("req.Push") || got != "req.Push@Page" {
		t.Fatalf("refused=%v got=%q", got != "", got)
	}
	exit()
	if Refuse("req.Push") || Active() {
		t.Fatal("scope leaked after exit")
	}
}

func TestOtherGoroutinesAreNotInTheScope(t *testing.T) {
	exit := Enter(&Scope{Component: "Page", Dev: true})
	defer exit()
	var wg sync.WaitGroup
	wg.Add(1)
	refused := true
	go func() {
		defer wg.Done()
		refused = Refuse("Shared.Set")
	}()
	wg.Wait()
	if refused {
		t.Fatal("a goroutine started by the effect was treated as inside it")
	}
}

func TestNestedScopesRestoreTheOuterOne(t *testing.T) {
	outer := Enter(&Scope{Component: "A", Dropped: func(string, string) {}})
	inner := Enter(&Scope{Component: "B", Dropped: func(string, string) {}})
	inner()
	if sc := Current(); sc == nil || sc.Component != "A" {
		t.Fatalf("current = %v", sc)
	}
	outer()
	if Current() != nil {
		t.Fatal("scope left behind")
	}
}
