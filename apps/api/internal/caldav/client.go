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
