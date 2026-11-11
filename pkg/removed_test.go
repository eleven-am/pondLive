package pkg

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func removedNames() (map[string]bool, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi fs.FileInfo) bool { return !strings.HasSuffix(fi.Name(), "_test.go") }, 0)
	if err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for _, p := range pkgs {
		for _, f := range p.Files {
			for _, d := range f.Decls {
				switch x := d.(type) {
				case *ast.FuncDecl:
					if x.Recv == nil {
						names[x.Name.Name] = true
					}
				case *ast.GenDecl:
					for _, s := range x.Specs {
						switch sp := s.(type) {
						case *ast.TypeSpec:
							names[sp.Name.Name] = true
						case *ast.ValueSpec:
							for _, n := range sp.Names {
								names[n.Name] = true
							}
						}
					}
				}
			}
		}
	}
	return names, nil
}

func exportedHooks(src map[string]bool) []string {
	var hooks []string
	for name := range src {
		if strings.HasPrefix(name, "Use") && len(name) > 3 && name[3] >= 'A' && name[3] <= 'Z' {
			hooks = append(hooks, name)
		}
	}
	sort.Strings(hooks)
	return hooks
}

func TestTheHookSetIsExactlyTheSixteenUseHooks(t *testing.T) {
	src, err := removedNames()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"UseBroadcast", "UseComputed", "UseContext", "UseEffect", "UseErrorBoundary", "UseHandler", "UseHead", "UseRef",
		"UseRenderEffect", "UseRequest", "UseRouter", "UseScript", "UseShared", "UseSignal", "UseStyles", "UseUpload",
	}
	got := exportedHooks(src)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("hooks = %v\nwant    %v", got, want)
	}
}

func TestOnlyTheSevenCapabilityRefTypesExist(t *testing.T) {
	src, err := removedNames()
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{"ElementRef": true, "InputRef": true, "FormRef": true, "MediaRef": true, "DialogRef": true, "CanvasRef": true, "DetailsRef": true}
	for name := range src {
		if strings.HasSuffix(name, "Ref") && name[0] >= 'A' && name[0] <= 'Z' && name != "UseRef" && !allowed[name] {
			t.Errorf("pkg declares the ref type %s", name)
		}
	}
}

func TestRemovedTopLevelNamesStayRemoved(t *testing.T) {
	src, err := removedNames()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"If", "IfFn", "Ternary", "TernaryFn", "Map", "MapIdx", "UseConnected", "UseElement", "UseWatch", "UseState",
		"UseDocument", "UseMetadata", "UseMetaTags", "UseHeaders", "UseCookie", "ErrUnsafeURL", "ErrQueryTimeout",
		"ResizeEvent", "HashChangeEvent", "PopStateEvent", "Modifiers", "ClassIf", "CN", "ClassPart", "InlineStyles",
		"Checked", "Selected", "Value", "Open", "OnKeyPress", "OnFileChange", "OnInput", "OnWithProps",
		"Outlet", "RoutesProps", "RouteProps", "Fallback",
	} {
		if src[name] {
			t.Errorf("pkg still declares %s", name)
		}
	}
}

func TestRemovedMembersStayRemoved(t *testing.T) {
	refs := []reflect.Type{
		reflect.TypeFor[*ElementRef](), reflect.TypeFor[*InputRef](), reflect.TypeFor[*FormRef](), reflect.TypeFor[*MediaRef](),
		reflect.TypeFor[*DialogRef](), reflect.TypeFor[*CanvasRef](), reflect.TypeFor[*DetailsRef](),
	}
	for _, typ := range refs {
		for _, name := range []string{
			"Call", "Set", "Query", "AsyncCall", "AddHandler", "SetValue", "SetChecked", "SetSelectedIndex",
			"Value", "Checked", "SelectedIndex", "Disabled", "GetValue", "GetChecked", "GetBoundingClientRect",
		} {
			if _, ok := typ.MethodByName(name); ok {
				t.Errorf("%s still has the method %s", typ, name)
			}
		}
		for i := 0; i < typ.NumMethod(); i++ {
			if name := typ.Method(i).Name; strings.HasPrefix(name, "On") || strings.HasPrefix(name, "Get") {
				t.Errorf("%s still has the method %s", typ, name)
			}
		}
	}
	for _, c := range []struct {
		typ     reflect.Type
		methods []string
		fields  []string
	}{
		{reflect.TypeFor[*Ctx](), []string{"Owner", "Session", "RunOnActor", "RequestClose"}, nil},
		{reflect.TypeFor[Script](), []string{"AttachTo"}, nil},
		{reflect.TypeFor[*UploadHandle](), []string{"AttachTo"}, nil},
		{reflect.TypeFor[*ElementRef](), []string{"AttachTo"}, nil},
		{reflect.TypeFor[*MediaRef](), []string{"AttachTo"}, nil},
		{reflect.TypeFor[*Request](), []string{"Redirect"}, nil},
		{reflect.TypeFor[*Router](), []string{"Redirect"}, nil},
		{reflect.TypeFor[EventOptions](), nil, []string{"Props"}},
		{reflect.TypeFor[*App](), []string{"Registry", "Session", "Sessions"}, nil},
		{reflect.TypeFor[Node](), []string{"ApplyTo", "ApplyBind"}, nil},
		{reflect.TypeFor[Item](), []string{"ApplyTo"}, nil},
		{reflect.TypeFor[BindOpt](), []string{"ApplyBind"}, nil},
	} {
		for _, name := range c.methods {
			if _, ok := c.typ.MethodByName(name); ok {
				t.Errorf("%s still has the method %s", c.typ, name)
			}
		}
		base := c.typ
		if base.Kind() == reflect.Pointer {
			base = base.Elem()
		}
		for _, name := range c.fields {
			if _, ok := base.FieldByName(name); ok {
				t.Errorf("%s still has the field %s", c.typ, name)
			}
		}
	}
}

func TestReportCannotCarryAFrameworkOnlySeverity(t *testing.T) {
	m, ok := reflect.TypeFor[*Ctx]().MethodByName("Report")
	if !ok {
		t.Fatal("Ctx.Report is missing")
	}
	if got := m.Type.NumIn(); got != 4 || m.Type.In(1) != reflect.TypeFor[ReportLevel]() || m.Type.In(2).Kind() != reflect.String || m.Type.In(3).Kind() != reflect.String {
		t.Fatalf("Ctx.Report has the signature %s", m.Type)
	}
	level := reflect.TypeFor[ReportLevel]()
	if level.Kind() != reflect.Struct {
		t.Fatalf("ReportLevel is a %s, so any number converts to it", level.Kind())
	}
	for i := 0; i < level.NumField(); i++ {
		if level.Field(i).IsExported() {
			t.Errorf("ReportLevel exposes the field %s", level.Field(i).Name)
		}
	}
}
