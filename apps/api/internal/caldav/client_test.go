package caldav

import "testing"

func TestObjectURL(t *testing.T) {
	c := New("https://cloud.example/remote.php/dav/calendars/u/personal/", "u", "p")

	got := c.objectURL("uid-1.ics")
	want := "https://cloud.example/remote.php/dav/calendars/u/personal/uid-1.ics"
	if got != want {
		t.Errorf("bare filename: got %q want %q", got, want)
	}

	got = c.objectURL("/remote.php/dav/calendars/u/personal/uid-2.ics")
	want = "https://cloud.example/remote.php/dav/calendars/u/personal/uid-2.ics"
	if got != want {
		t.Errorf("root-relative: got %q want %q", got, want)
	}

	abs := "https://other.example/c/x.ics"
	if got := c.objectURL(abs); got != abs {
		t.Errorf("absolute: got %q want %q", got, abs)
	}
}

func TestUidOf(t *testing.T) {
	ics := "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nUID:abc-123\r\nSUMMARY:x\r\nEND:VEVENT\r\nEND:VCALENDAR"
	if got := uidOf(ics); got != "abc-123" {
		t.Errorf("got %q", got)
	}
	if got := uidOf("BEGIN:VEVENT\nSUMMARY:x"); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}
