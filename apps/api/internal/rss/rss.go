// Package rss parses RSS 2.0 and Atom feeds into ical events so the feed
// pipeline (cache → expand → render) works unchanged — each item becomes an
// all-day event on its publication date.
package rss

import (
	"bytes"
	"encoding/xml"
	"strings"
	"time"

	"cal/apps/api/internal/ical"
)

// Item is one feed entry normalised across RSS and Atom.
type Item struct {
	Title   string
	Link    string
	Summary string
	Date    time.Time
	GUID    string
}

// LooksLikeFeed reports whether a body sniffs as RSS/Atom rather than ICS.
func LooksLikeFeed(body string) bool {
	head := strings.ToUpper(body[:min(512, len(body))])
	return strings.Contains(head, "<RSS") || strings.Contains(head, "<FEED") || strings.Contains(head, "<RDF:RDF")
}

// ToICS parses the body and serialises up to 200 newest items as a VCALENDAR.
func ToICS(body string) (string, error) {
	items, err := parse(body)
	if err != nil {
		return "", err
	}
	if len(items) > 200 {
		items = items[:200]
	}
	events := make([]ical.Event, 0, len(items))
	for _, it := range items {
		if it.Date.IsZero() {
			continue
		}
		events = append(events, ical.Event{
			UID:         it.GUID,
			Summary:     it.Title,
			Description: it.Summary,
			URL:         it.Link,
			Start:       it.Date,
			End:         it.Date.AddDate(0, 0, 1),
			AllDay:      true,
		})
	}
	return ical.EncodeCalendar(events), nil
}

// --- parsing ---

type rssDoc struct {
	Items []struct {
		Title   string `xml:"title"`
		Link    string `xml:"link"`
		GUID    string `xml:"guid"`
		PubDate string `xml:"pubDate"`
		DCDate  string `xml:"date"`
		Desc    string `xml:"description"`
	} `xml:"channel>item"`
}

type atomDoc struct {
	XMLName xml.Name `xml:"feed"`
	Entries []struct {
		Title   string `xml:"title"`
		Links   []struct {
			Rel  string `xml:"rel,attr"`
			Href string `xml:"href,attr"`
		} `xml:"link"`
		ID        string `xml:"id"`
		Updated   string `xml:"updated"`
		Published string `xml:"published"`
		Summary   string `xml:"summary"`
		Content   string `xml:"content"`
	} `xml:"entry"`
}

func parse(body string) ([]Item, error) {
	data := []byte(body)
	var out []Item

	var rss rssDoc
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&rss); err == nil && len(rss.Items) > 0 {
		for _, i := range rss.Items {
			out = append(out, Item{
				Title:   strings.TrimSpace(i.Title),
				Link:    strings.TrimSpace(i.Link),
				Summary: stripTags(i.Desc),
				Date:    parseDate(i.PubDate, i.DCDate),
				GUID:    firstNonEmpty(i.GUID, i.Link, i.Title),
			})
		}
		return out, nil
	}

	var atom atomDoc
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&atom); err != nil {
		return nil, err
	}
	for _, e := range atom.Entries {
		link := ""
		for _, l := range e.Links {
			if l.Rel == "" || l.Rel == "alternate" {
				link = l.Href
				break
			}
		}
		out = append(out, Item{
			Title:   strings.TrimSpace(e.Title),
			Link:    link,
			Summary: stripTags(firstNonEmpty(e.Summary, e.Content)),
			Date:    parseDate(e.Published, e.Updated),
			GUID:    firstNonEmpty(e.ID, link, e.Title),
		})
	}
	return out, nil
}

var dateFormats = []string{
	time.RFC1123Z, time.RFC1123, time.RFC822Z, time.RFC822,
	time.RFC3339, "2006-01-02", "2006-01-02T15:04:05Z0700",
	"Mon, 02 Jan 2006 15:04:05 MST", "02 Jan 2006 15:04:05 -0700",
}

func parseDate(values ...string) time.Time {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		for _, f := range dateFormats {
			if t, err := time.Parse(f, v); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

func stripTags(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		switch {
		case r == '<':
			in = true
		case r == '>':
			in = false
		case !in:
			b.WriteRune(r)
		}
	}
	out := strings.Join(strings.Fields(b.String()), " ")
	if len(out) > 300 {
		out = out[:300] + "…"
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
