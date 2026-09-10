package rss

import (
	"strings"
	"testing"
)

func TestRSS2(t *testing.T) {
	body := `<?xml version="1.0"?><rss version="2.0"><channel>
		<item><title>Post One</title><link>https://blog.example/one</link>
		<pubDate>Mon, 01 Sep 2025 10:00:00 +0200</pubDate>
		<description>&lt;p&gt;Hello &lt;b&gt;world&lt;/b&gt;&lt;/p&gt;</description>
		<guid>one-guid</guid></item>
		<item><title>Post Two</title><link>https://blog.example/two</link>
		<pubDate>2025-09-02</pubDate></item>
	</channel></rss>`
	ics, err := ToICS(body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ics, "SUMMARY:Post One") || !strings.Contains(ics, "URL:https://blog.example/one") {
		t.Fatalf("missing item fields:\n%s", ics)
	}
	if !strings.Contains(ics, "Hello world") {
		t.Fatalf("description not stripped: %s", ics)
	}
	if !strings.Contains(ics, "DTSTART;VALUE=DATE:20250901") {
		t.Fatalf("pubDate not mapped: %s", ics)
	}
}

func TestAtom(t *testing.T) {
	body := `<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom">
		<entry><title>Atom Post</title><link rel="alternate" href="https://a.example/p"/>
		<id>urn:x:1</id><published>2025-08-20T09:30:00Z</published>
		<summary>Atom summary</summary></entry>
	</feed>`
	ics, err := ToICS(body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ics, "SUMMARY:Atom Post") || !strings.Contains(ics, "DTSTART;VALUE=DATE:20250820") {
		t.Fatalf("atom item wrong: %s", ics)
	}
}

func TestParseDateFormats(t *testing.T) {
	for _, v := range []string{"Mon, 01 Sep 2025 10:00:00 +0200", "2025-09-01T10:00:00Z", "2025-09-01"} {
		if parseDate(v).IsZero() {
			t.Fatalf("unparsed: %q", v)
		}
	}
}

func TestLooksLike(t *testing.T) {
	if !LooksLikeFeed("<?xml version='1.0'?><rss version='2.0'>") {
		t.Fatal("rss not detected")
	}
	if LooksLikeFeed("BEGIN:VCALENDAR") {
		t.Fatal("ics misdetected as feed")
	}
}
