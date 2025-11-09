package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

/************ Test helpers: fake RoundTripper ************/

type fakeRT struct {
	mu      sync.Mutex
	records []*http.Request
	// handler decides response based on request
	handler func(req *http.Request) *http.Response
}

func (f *fakeRT) RoundTrip(req *http.Request) (*http.Response, error) {
	f.mu.Lock()
	f.records = append(f.records, req.Clone(req.Context()))
	f.mu.Unlock()
	return f.handler(req), nil
}

func newResp(status int, body any, headers map[string]string, req *http.Request) *http.Response {
	var rc io.ReadCloser
	if body != nil {
		b, _ := json.Marshal(body)
		rc = io.NopCloser(bytes.NewReader(b))
	} else {
		rc = io.NopCloser(strings.NewReader(""))
	}
	h := make(http.Header, len(headers))
	for k, v := range headers {
		h.Set(k, v)
	}
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       rc,
		Header:     h,
		Request:    req,
	}
}

/************ Tests ************/

func TestWorkspaces_PaginationAndHeaders(t *testing.T) {
	// Simulate two pages:
	// page1 -> next_page.offset="abc"
	// page2 -> no next_page
	rt := &fakeRT{}
	rt.handler = func(req *http.Request) *http.Response {
		if got := req.Header.Get("Authorization"); !strings.HasPrefix(got, "Bearer ") {
			t.Fatalf("missing/invalid Authorization header: %q", got)
		}
		if ua := req.Header.Get("User-Agent"); ua == "" {
			t.Fatalf("missing User-Agent header")
		}

		if req.URL.Path != "/api/1.0/workspaces" {
			return newResp(404, map[string]string{"error": "wrong path"}, nil, req)
		}

		q := req.URL.Query()
		offset := q.Get("offset")
		if offset == "" {
			// first page
			return newResp(200, ListResponse[Workspace]{
				Data: []Workspace{
					{Gid: "w1", Name: "Acme One"},
				},
				NextPage: &struct {
					Offset string `json:"offset"`
					Path   string `json:"path"`
					URI    string `json:"uri"`
				}{
					Offset: "abc",
					Path:   "/workspaces?offset=abc",
					URI:    "https://app.asana.com/api/1.0/workspaces?offset=abc",
				},
			}, nil, req)
		}
		if offset == "abc" {
			// second (final) page
			return newResp(200, ListResponse[Workspace]{
				Data: []Workspace{
					{Gid: "w2", Name: "Beta Two"},
				},
				NextPage: nil,
			}, nil, req)
		}
		return newResp(400, map[string]string{"error": "unexpected offset"}, nil, req)
	}

	httpc := &http.Client{Transport: rt, Timeout: 5 * time.Second}
	cl := &Client{http: httpc, pat: "test_pat"}

	ctx := context.Background()
	got, err := cl.Workspaces(ctx)
	if err != nil {
		t.Fatalf("Workspaces error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 workspaces, got %d: %+v", len(got), got)
	}
	if got[0].Gid != "w1" || got[1].Gid != "w2" {
		t.Fatalf("unexpected workspaces order/content: %+v", got)
	}

	// ensure we hit two requests with offset progression
	if len(rt.records) != 2 {
		t.Fatalf("want 2 HTTP calls, got %d", len(rt.records))
	}
	checkHasQuery(t, rt.records[0].URL, "offset", "")
	checkHasQuery(t, rt.records[1].URL, "offset", "abc")
}

func TestUsers_HandlesRetryAfter429ThenSuccess(t *testing.T) {
	var mu sync.Mutex
	attempts := 0

	rt := &fakeRT{}
	rt.handler = func(req *http.Request) *http.Response {
		if req.URL.Path != "/api/1.0/users" {
			return newResp(404, map[string]string{"error": "wrong path"}, nil, req)
		}
		q := req.URL.Query()
		if q.Get("workspace") != "w1" {
			return newResp(400, map[string]string{"error": "missing workspace"}, nil, req)
		}

		mu.Lock()
		defer mu.Unlock()
		attempts++
		switch attempts {
		case 1:
			// first call rate limited; small Retry-After to keep test fast
			return newResp(429, nil, map[string]string{"Retry-After": "1"}, req)
		case 2:
			// on retry, return one-page response
			return newResp(200, ListResponse[User]{
				Data: []User{{Gid: "u1", Name: "Alice", Email: strPtr("a@example.com")}},
			}, nil, req)
		default:
			return newResp(500, nil, nil, req) // should not reach
		}
	}

	httpc := &http.Client{Transport: rt, Timeout: 10 * time.Second}
	cl := &Client{http: httpc, pat: "x"}

	start := time.Now()
	users, err := cl.Users(context.Background(), "w1")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Users error: %v", err)
	}
	if len(users) != 1 || users[0].Gid != "u1" {
		t.Fatalf("unexpected users: %+v", users)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts (429 then 200), got %d", attempts)
	}

	// Sanity: elapsed should be >= Retry-After (≈1s). Give slack for scheduler.
	if elapsed < 900*time.Millisecond {
		t.Fatalf("429 retry did not wait; elapsed=%v", elapsed)
	}
}

/************ tiny helpers ************/

func strPtr(s string) *string { return &s }

func checkHasQuery(t *testing.T, u *url.URL, key, want string) {
	t.Helper()
	got := u.Query().Get(key)
	if got != want {
		t.Fatalf("URL %s query %s: want %q, got %q", u.String(), key, want, got)
	}
}
