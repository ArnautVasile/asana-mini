package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	baseURL    = "https://app.asana.com/api/1.0"
	userAgent  = "asana-mini/1.0 (+github.com/ArnautVasile/asana-mini)"
	maxRetries = 6
	pageLimit  = 100
)

type Client struct {
	http *http.Client
	pat  string
}

func New(pat string, timeout time.Duration) *Client {
	return &Client{
		http: &http.Client{Timeout: timeout},
		pat:  pat,
	}
}

func (c *Client) Workspaces(ctx context.Context) ([]Workspace, error) {
	var all []Workspace
	offset := ""
	for {
		u, _ := url.Parse(baseURL + "/workspaces")
		q := u.Query()
		q.Set("limit", strconv.Itoa(pageLimit))
		if offset != "" {
			q.Set("offset", offset)
		}
		u.RawQuery = q.Encode()

		var page ListResponse[Workspace]
		if err := c.getJSON(ctx, u.String(), &page); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if page.NextPage == nil || page.NextPage.Offset == "" {
			break
		}
		offset = page.NextPage.Offset
	}
	return all, nil
}

func (c *Client) Projects(ctx context.Context, workspaceGID string) ([]Project, error) {
	var all []Project
	offset := ""
	for {
		u, _ := url.Parse(baseURL + "/projects")
		q := u.Query()
		q.Set("limit", strconv.Itoa(pageLimit))
		q.Set("workspace", workspaceGID)
		// q.Set("archived", "false")
		if offset != "" {
			q.Set("offset", offset)
		}
		u.RawQuery = q.Encode()

		var page ListResponse[Project]
		if err := c.getJSON(ctx, u.String(), &page); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if page.NextPage == nil || page.NextPage.Offset == "" {
			break
		}
		offset = page.NextPage.Offset
	}
	return all, nil
}

func (c *Client) Users(ctx context.Context, workspaceGID string) ([]User, error) {
	var all []User
	offset := ""
	for {
		u, _ := url.Parse(baseURL + "/users")
		q := u.Query()
		q.Set("limit", strconv.Itoa(pageLimit))
		q.Set("workspace", workspaceGID)
		if offset != "" {
			q.Set("offset", offset)
		}
		u.RawQuery = q.Encode()

		var page ListResponse[User]
		if err := c.getJSON(ctx, u.String(), &page); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if page.NextPage == nil || page.NextPage.Offset == "" {
			break
		}
		offset = page.NextPage.Offset
	}
	return all, nil
}

func (c *Client) getJSON(ctx context.Context, urlStr string, out any) error {
	var lastErr error
	backoff := 1 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.pat)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", userAgent)

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(backoff)
			backoff = nextBackoff(backoff)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		switch {
		case resp.StatusCode == http.StatusOK:
			return json.Unmarshal(body, out)

		case resp.StatusCode == http.StatusTooManyRequests:
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
			if retryAfter <= 0 {
				retryAfter = backoff
			}
			time.Sleep(retryAfter)
			backoff = nextBackoff(backoff)

		case resp.StatusCode >= 500 && resp.StatusCode <= 599:
			time.Sleep(backoff)
			backoff = nextBackoff(backoff)

		default:
			return fmt.Errorf("GET %s: %s: %s", urlStr, resp.Status, string(body))
		}
	}

	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("exhausted retries for %s", urlStr)
}

func parseRetryAfter(h string) time.Duration {
	if h == "" {
		return 0
	}
	if secs, err := strconv.Atoi(h); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(h); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

func nextBackoff(cur time.Duration) time.Duration {
	n := cur * 2
	if n > 32*time.Second {
		n = 32 * time.Second
	}
	return n
}
