package httpapi

// Link unfurling: server-side fetch of <title>, favicon and og:image for a
// URL. SSRF-guarded: http(s) only, private/loopback addresses refused after
// DNS resolution and after each redirect.

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var unfurlClient = &http.Client{
	Timeout: 8 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return errors.New("too many redirects")
		}
		return checkURL(req.URL)
	},
}

// checkURL refuses non-http schemes and hosts that resolve to private,
// loopback, link-local or unspecified addresses.
func checkURL(u *url.URL) error {
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("scheme not allowed")
	}
	host := u.Hostname()
	if host == "" {
		return errors.New("missing host")
	}
	ips, err := net.DefaultResolver.LookupIPAddr(context.Background(), host)
	if err != nil || len(ips) == 0 {
		return errors.New("host unresolvable")
	}
	for _, ipa := range ips {
		ip := ipa.IP
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() || ip.IsMulticast() {
			return errors.New("private address")
		}
	}
	return nil
}

var (
	reTitle = regexp.MustCompile(`(?is)<title[^>]*>\s*(.*?)\s*</title>`)
	reOG    = regexp.MustCompile(`(?is)<meta[^>]+(?:property|name)=["'](?:og:title|twitter:title)["'][^>]+content=["']([^"']+)["']`)
	reOGRev = regexp.MustCompile(`(?is)<meta[^>]+content=["']([^"']+)["'][^>]+(?:property|name)=["'](?:og:title|twitter:title)["']`)
	reIcon  = regexp.MustCompile(`(?is)<link[^>]+rel=["'](?:icon|shortcut icon|apple-touch-icon)["'][^>]+href=["']([^"']+)["']`)
	reDesc  = regexp.MustCompile(`(?is)<meta[^>]+name=["']description["'][^>]+content=["']([^"']+)["']`)
)

func (s *Server) unfurl(c *gin.Context) {
	raw := strings.TrimSpace(c.Query("url"))
	u, err := url.Parse(raw)
	if err != nil || checkURL(u) != nil {
		c.String(http.StatusBadRequest, "invalid url")
		return
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, raw, nil)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid url")
		return
	}
	req.Header.Set("User-Agent", "Cal-LinkPreview/1.0")
	req.Header.Set("Accept", "text/html")
	resp, err := unfurlClient.Do(req)
	if err != nil {
		c.String(http.StatusBadGateway, "fetch failed")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		c.String(http.StatusBadGateway, "fetch failed")
		return
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		c.String(http.StatusBadGateway, "read failed")
		return
	}
	html := string(body)
	title := first(reOG.FindStringSubmatch(html), reOGRev.FindStringSubmatch(html), reTitle.FindStringSubmatch(html))
	icon := ""
	if m := reIcon.FindStringSubmatch(html); m != nil {
		icon = resolveURL(u, m[1])
	}
	if icon == "" {
		icon = u.Scheme + "://" + u.Host + "/favicon.ico"
	}
	desc := ""
	if m := reDesc.FindStringSubmatch(html); m != nil {
		desc = decodeEntities(strings.TrimSpace(m[1]))
	}
	c.JSON(http.StatusOK, gin.H{
		"title":       decodeEntities(strings.TrimSpace(title)),
		"favicon":     icon,
		"description": desc,
	})
}

func first(ms ...[]string) string {
	for _, m := range ms {
		if len(m) > 1 && strings.TrimSpace(m[1]) != "" {
			return m[1]
		}
	}
	return ""
}

func resolveURL(base *url.URL, href string) string {
	r, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return base.ResolveReference(r).String()
}

var reEntity = regexp.MustCompile(`&(?:#(\d+)|amp|quot|lt|gt|apos|nbsp);`)

func decodeEntities(s string) string {
	return reEntity.ReplaceAllStringFunc(s, func(ent string) string {
		switch ent {
		case "&amp;":
			return "&"
		case "&quot;":
			return `"`
		case "&lt;":
			return "<"
		case "&gt;":
			return ">"
		case "&apos;":
			return "'"
		case "&nbsp;":
			return " "
		}
		if m := reEntity.FindStringSubmatch(ent); m != nil && m[1] != "" {
			if code, err := strconv.Atoi(m[1]); err == nil && code > 0 && code < 0x10FFFF {
				return string(rune(code))
			}
		}
		return ent
	})
}
