// Minimal CalDAV client: REPORT calendar-query on a collection URL, plus
// PUT/DELETE for VEVENT resources. Basic auth only — works with Radicale,
// Nextcloud app passwords, Baikal, Fastmail. jarvis: no principal discovery;
// the user pastes the calendar collection URL directly.

package caldav

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Client struct {
	http     *http.Client
	url      string
	username string
	password string
}

// RemoteEvent is one object in the remote collection.
type RemoteEvent struct {
	Href string
	ETag string
	ICS  string
}

func New(collectionURL, username, password string) *Client {
	return &Client{
		http:     &http.Client{Timeout: 20 * time.Second},
		url:      strings.TrimSuffix(collectionURL, "/") + "/",
		username: username,
		password: password,
	}
}

func (c *Client) do(ctx context.Context, method, url, contentType string, body []byte, depth string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.username, c.password)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if depth != "" {
		req.Header.Set("Depth", depth)
	}
	return c.http.Do(req)
}

// TestConnection PROPFINDs the collection — verifies URL + credentials.
func (c *Client) TestConnection(ctx context.Context) error {
	body := `<?xml version="1.0" encoding="utf-8"?>
<D:propfind xmlns:D="DAV:"><D:prop><D:resourcetype/><D:displayname/></D:prop></D:propfind>`
	resp, err := c.do(ctx, "PROPFIND", c.url, "application/xml", []byte(body), "0")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("authentication failed")
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}
	return nil
}

type multistatus struct {
	Responses []struct {
		Href     string `xml:"href"`
		PropStat []struct {
			Prop struct {
				ETag         string `xml:"getetag"`
				CalendarData string `xml:"calendar-data"`
			} `xml:"prop"`
		} `xml:"propstat"`
	} `xml:"response"`
}

// ListEvents returns every VEVENT in the collection with href + etag + data.
func (c *Client) ListEvents(ctx context.Context) ([]RemoteEvent, error) {
	body := `<?xml version="1.0" encoding="utf-8"?>
<C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop><D:getetag/><C:calendar-data/></D:prop>
  <C:filter><C:comp-filter name="VCALENDAR"><C:comp-filter name="VEVENT"/></C:comp-filter></C:filter>
</C:calendar-query>`
	resp, err := c.do(ctx, "REPORT", c.url, "application/xml", []byte(body), "1")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("authentication failed")
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	var ms multistatus
	if err := xml.Unmarshal(data, &ms); err != nil {
		return nil, fmt.Errorf("parse multistatus: %w", err)
	}
	var events []RemoteEvent
	for _, r := range ms.Responses {
		for _, ps := range r.PropStat {
			ics := ps.Prop.CalendarData
			if ics == "" {
				continue
			}
			events = append(events, RemoteEvent{Href: r.Href, ETag: ps.Prop.ETag, ICS: ics})
			break
		}
	}
	return events, nil
}

// objectURL resolves an href against the collection URL. Root-relative hrefs
// keep scheme+host; bare filenames join onto the collection path.
func (c *Client) objectURL(href string) string {
	if strings.HasPrefix(href, "http") {
		return href
	}
	if strings.HasPrefix(href, "/") {
		scheme := strings.Index(c.url, "://") + 3
		if i := strings.Index(c.url[scheme:], "/"); i >= 0 {
			return c.url[:scheme+i] + href
		}
	}
	return c.url + href
}

// PutEvent upserts a VEVENT body. Returns the new ETag when the server sends one.
func (c *Client) PutEvent(ctx context.Context, href, ics, etag string) (string, error) {
	url := c.objectURL(href)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, strings.NewReader(ics))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-Type", "text/calendar; charset=utf-8")
	if etag != "" {
		req.Header.Set("If-Match", etag)
	} else {
		req.Header.Set("If-None-Match", "*")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("put failed: %d", resp.StatusCode)
	}
	return resp.Header.Get("ETag"), nil
}

func (c *Client) DeleteEvent(ctx context.Context, href string) error {
	resp, err := c.do(ctx, "DELETE", c.objectURL(href), "", nil, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("delete failed: %d", resp.StatusCode)
	}
	return nil
}

// Collection is one calendar collection discovered on the server.
type Collection struct {
	Href string `json:"href"`
	Name string `json:"name"`
}

// Discover walks principal → calendar-home-set → child collections so the UI
// can offer a picker instead of requiring a raw collection URL.
func Discover(ctx context.Context, baseURL, username, password string) ([]Collection, error) {
	c := New(baseURL, username, password)

	// 1. current-user-principal on the server root.
	principal, err := c.propValue(ctx, c.url, `<D:propfind xmlns:D="DAV:"><D:prop><D:current-user-principal/></D:prop></D:propfind>`, "current-user-principal")
	if err != nil {
		return nil, err
	}
	if principal == "" {
		// Some servers accept PROPFIND on the collection directly — try base as home-set.
		principal = c.url
	}
	principalURL := c.objectURL(principal)

	// 2. calendar-home-set on the principal.
	home, err := c.propValue(ctx, principalURL, `<D:propfind xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav"><D:prop><C:calendar-home-set/></D:prop></D:propfind>`, "calendar-home-set")
	if err != nil || home == "" {
		return nil, fmt.Errorf("no calendar-home-set (status ok but empty)")
	}
	homeURL := c.objectURL(home)

	// 3. Depth:1 on the home-set → child collections.
	body := `<?xml version="1.0" encoding="utf-8"?>
<D:propfind xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop><D:resourcetype/><D:displayname/></D:prop>
</D:propfind>`
	resp, err := c.do(ctx, "PROPFIND", homeURL, "application/xml", []byte(body), "1")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	type resp2 struct {
		Href     string `xml:"href"`
		PropStat []struct {
			Prop struct {
				Name string `xml:"displayname"`
				RT   struct {
					Cal []struct{} `xml:"calendar"`
				} `xml:"resourcetype"`
			} `xml:"prop"`
		} `xml:"propstat"`
	}
	var ms struct {
		Responses []resp2 `xml:"response"`
	}
	if err := xml.Unmarshal(data, &ms); err != nil {
		return nil, fmt.Errorf("parse home-set: %w", err)
	}
	var out []Collection
	for _, r := range ms.Responses {
		for _, ps := range r.PropStat {
			if len(ps.Prop.RT.Cal) == 0 {
				continue // not a calendar collection
			}
			name := ps.Prop.Name
			if name == "" {
				name = r.Href
			}
			out = append(out, Collection{Href: c.objectURL(r.Href), Name: name})
			break
		}
	}
	return out, nil
}

// propValue extracts the href inside a single-prop PROPFIND response.
func (c *Client) propValue(ctx context.Context, url, body, tag string) (string, error) {
	resp, err := c.do(ctx, "PROPFIND", url, "application/xml", []byte(body), "0")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return "", fmt.Errorf("authentication failed")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	// Tags may carry any namespace prefix (<C:calendar-home-set> or bare).
	// Extract the <href> inside the named prop.
	re := regexp.MustCompile(`(?is)<(?:[A-Za-z0-9]+:)?` + tag + `[^>]*>.*?<(?:[A-Za-z0-9]+:)?href[^>]*>(.*?)</(?:[A-Za-z0-9]+:)?href>`)
	if m := re.FindSubmatch(data); m != nil {
		return strings.TrimSpace(string(m[1])), nil
	}
	return "", nil
}
