// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package guidance

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

const explainOut = `{"endpoint":"","findings":[{"id":"net.tls","title":"net.tls","status":"fail","means":"TLS matters.",
  "steps":[{"title":"Serve https","body":"Terminate TLS."}]},{"id":"other.x","status":"fail"}]}`

func fake(out string, err error, calls *int, got *[]byte) Runner {
	return func(_ context.Context, program string, args []string, stdin []byte) ([]byte, error) {
		*calls++
		if program == "" || strings.Join(args, " ") != "explain --output json" {
			return nil, errors.New("unexpected invocation")
		}
		*got = stdin
		return []byte(out), err
	}
}

func TestLookupFound(t *testing.T) {
	var calls int
	var stdin []byte
	p := &Passmcp{Run: fake(explainOut, nil, &calls, &stdin)}
	g, found, err := p.Lookup(context.Background(), "net.tls")
	if err != nil || !found || g.Means != "TLS matters." || len(g.Steps) != 1 {
		t.Fatalf("%+v %v %v", g, found, err)
	}
	checkReport(t, stdin, "net", "net.tls")
	if _, _, err := p.Lookup(context.Background(), "net.tls"); err != nil || calls != 1 {
		t.Fatalf("the second lookup must come from the cache: %d calls", calls)
	}
}

// checkReport asserts the report sent on stdin is one failing finding.
func checkReport(t *testing.T, stdin []byte, phase, id string) {
	t.Helper()
	var rep struct {
		Phases []struct {
			Name     string                        `json:"name"`
			Findings []struct{ ID, Status string } `json:"findings"`
		} `json:"phases"`
	}
	if err := json.Unmarshal(stdin, &rep); err != nil || len(rep.Phases) != 1 || len(rep.Phases[0].Findings) != 1 {
		t.Fatalf("report on stdin: %s", stdin)
	}
	p := rep.Phases[0]
	if p.Name != phase || p.Findings[0].ID != id || p.Findings[0].Status != "fail" {
		t.Fatalf("report on stdin: %s", stdin)
	}
}

func TestLookupNotFound(t *testing.T) {
	var calls int
	var stdin []byte
	for _, id := range []string{"other.x", "missing.y"} {
		p := &Passmcp{Program: "/opt/passmcp", Run: fake(explainOut, nil, &calls, &stdin)}
		if _, found, err := p.Lookup(context.Background(), id); err != nil || found {
			t.Fatalf("%s: found=%v err=%v", id, found, err)
		}
	}
}

func TestLookupErrors(t *testing.T) {
	var calls int
	var stdin []byte
	p := &Passmcp{Run: fake("", errors.New("boom"), &calls, &stdin)}
	if _, _, err := p.Lookup(context.Background(), "a.b"); err == nil {
		t.Fatal("a failed run is an error")
	}
	if _, _, err := p.Lookup(context.Background(), "a.b"); err == nil || calls != 2 {
		t.Fatal("errors are not cached")
	}
	p = &Passmcp{Run: fake("not json", nil, &calls, &stdin)}
	if _, _, err := p.Lookup(context.Background(), "a.b"); err == nil || !strings.Contains(err.Error(), "JSON document") {
		t.Fatalf("garbage output: %v", err)
	}
}

func TestTimeoutApplies(t *testing.T) {
	p := &Passmcp{Timeout: time.Millisecond, Run: func(ctx context.Context, _ string, _ []string, _ []byte) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	if _, _, err := p.Lookup(context.Background(), "a.b"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want a deadline, got %v", err)
	}
}

func TestProgramDefault(t *testing.T) {
	if (&Passmcp{}).program() != "passmcp" || (&Passmcp{Program: " "}).program() != "passmcp" || (&Passmcp{Program: "/x/p"}).program() != "/x/p" {
		t.Fatal("program")
	}
	if phaseOf("nodot") != "nodot" {
		t.Fatal("phaseOf")
	}
}

func TestReportEncodesTheID(t *testing.T) {
	b := report(`a.b","status":"pass`)
	var rep map[string]any
	if err := json.Unmarshal(b, &rep); err != nil || !strings.Contains(string(b), `a.b\",\"status\":\"pass`) {
		t.Fatalf("the id must stay a string: %s", b)
	}
}

func TestMarkdown(t *testing.T) {
	g := Guidance{ID: "net.tls_x", Means: "Means.", Steps: []Step{{"One", "Do one."}, {"Two", "Do two."}}}
	md := Markdown(g, "https://satellion.com/passmcp/docs/checks/#check-net-tls")
	for _, want := range []string{`**net.tls\_x**`, "Means.", "1. **One**: Do one.", "2. **Two**", "[Documentation](https://satellion.com/passmcp/docs/checks/#check-net-tls)", "passmcp's catalogue"} {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q in %s", want, md)
		}
	}
	for _, bad := range []string{"javascript:alert(1)", "https://", "%zz", ""} {
		if strings.Contains(Markdown(Guidance{ID: "a.b"}, bad), "[Documentation]") {
			t.Errorf("%q must not become a link", bad)
		}
	}
	// A bracket or parenthesis in the link is percent-encoded, so it cannot
	// end the link and start another.
	md = Markdown(Guidance{ID: "a.b"}, "https://a.example/x) [evil](https://b")
	if strings.Contains(md, "[evil]") || !strings.Contains(md, "x%29%20%5Bevil%5D%28https://b)") {
		t.Errorf("unescaped link text: %s", md)
	}
	if _, ok := safeLink("https://a.example/?q=(x)"); ok {
		t.Error("a raw query keeps its parentheses, so it is refused")
	}
	if u := Unavailable("a*b", "passmcp is not installed"); !strings.Contains(u, `a\*b`) || !strings.Contains(u, "_passmcp is not installed_") {
		t.Fatal(u)
	}
}
