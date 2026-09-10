// Package carddav pulls contacts and extracts birthdays (BDAY) into yearly
// recurring entries. Read-only — CardDAV writes are out of scope.
package carddav

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Client struct {
	Username string
	Password string
	URL      string // addressbook collection URL
	http     *http.Client
}

func New(url, username, password string) *Client {
	return &Client{URL: url, Username: username, Password: password, http: &http.Client{Timeout: 15 * time.Second}}
}

// Contact is the sliver of a VCARD we care about.
type Contact struct {
	Name     string
	Birthday string // MM-DD or YYYY-MM-DD normalized to --MM-DD if yearless
}

// Birthdays fetches all vCards and extracts FN + BDAY pairs.
func (c *Client) Birthdays(ctx context.Context) ([]Contact, error) {
	body := `<?xml version="1.0" encoding="utf-8"?>
<C:addressbook-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:carddav">
  <D:prop><D:getetag/><C:address-data/></D:prop>
</C:addressbook-query>`
	req, err := http.NewRequestWithContext(ctx, "REPORT", c.URL, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Depth", "1")
	req.SetBasicAuth(c.Username, c.Password)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 207 {
		return nil, fmt.Errorf("report: %d", resp.StatusCode)
	}
	return parseMultistatus(string(data)), nil
}

var reAddressData = regexp.MustCompile(`(?is)<(?:[A-Za-z0-9]+:)?address-data[^>]*>(.*?)</(?:[A-Za-z0-9]+:)?address-data>`)

func parseMultistatus(xml string) []Contact {
	var out []Contact
	for _, m := range reAddressData.FindAllStringSubmatch(xml, -1) {
		if c := parseVCard(m[1]); c != nil {
			out = append(out, *c)
		}
	}
	return out
}

var (
	reFN   = regexp.MustCompile(`(?im)^FN[^:\r\n]*:(.*)$`)
	reBDAY = regexp.MustCompile(`(?im)^BDAY[^:\r\n]*:(.*)$`)
)

// parseVCard unfolds continuation lines and extracts FN + BDAY.
func parseVCard(raw string) *Contact {
	unfolded := regexp.MustCompile("\r?\n[ \t]").ReplaceAllString(raw, "")
	var name, bday string
	for _, line := range strings.Split(unfolded, "\n") {
		line = strings.TrimSpace(line)
		if m := reFN.FindStringSubmatch(line); m != nil && name == "" {
			name = strings.TrimSpace(m[1])
		}
		if m := reBDAY.FindStringSubmatch(line); m != nil && bday == "" {
			bday = normalizeBday(strings.TrimSpace(m[1]))
		}
	}
	if name == "" || bday == "" {
		return nil
	}
	return &Contact{Name: name, Birthday: bday}
}

// normalizeBday accepts YYYYMMDD, YYYY-MM-DD, --MM-DD, --MMDD → "MM-DD" (we
// only need the day for a yearly recurring entry).
func normalizeBday(v string) string {
	digits := regexp.MustCompile(`\D`).ReplaceAllString(v, "")
	if len(digits) == 8 { // YYYYMMDD
		return digits[4:6] + "-" + digits[6:8]
	}
	if len(digits) == 4 { // MMDD from --MMDD
		return digits[:2] + "-" + digits[2:]
	}
	if len(digits) == 6 { // YYYYMMDD misstripped (leading -- removed)
		return digits[2:4] + "-" + digits[4:6]
	}
	return ""
}
