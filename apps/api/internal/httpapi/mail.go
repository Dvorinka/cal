package httpapi

// Mail module: a pragmatic IMAP/SMTP client (Edison/Briefkescht-shaped).
// Accounts store encrypted credentials (same AES-256-GCM path as CalDAV);
// every request opens a short-lived connection — no daemon, no cache.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"cal/apps/api/internal/store"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/gin-gonic/gin"
)

// imapDial connects + logs in; caller defers Logout.
func (s *Server) imapDial(ctx context.Context, userID, accountID string) (*imapclient.Client, store.MailAccount, error) {
	acct, password, err := s.store.MailCredentials(ctx, userID, accountID)
	if err != nil {
		return nil, acct, err
	}
	addr := fmt.Sprintf("%s:%d", acct.IMAPHost, acct.IMAPPort)
	opts := &imapclient.Options{Dialer: &net.Dialer{Timeout: 12 * time.Second}}
	var client *imapclient.Client
	if acct.IMAPPort == 143 {
		client, err = imapclient.DialStartTLS(addr, opts)
	} else {
		client, err = imapclient.DialTLS(addr, opts)
	}
	if err != nil {
		return nil, acct, fmt.Errorf("imap dial: %w", err)
	}
	if err := client.Login(acct.Username, password).Wait(); err != nil {
		client.Close()
		return nil, acct, fmt.Errorf("imap login: %w", err)
	}
	return client, acct, nil
}

// --- Account CRUD ---

func (s *Server) listMailAccounts(c *gin.Context) {
	items, err := s.store.ListMailAccounts(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) createMailAccount(c *gin.Context) {
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		IMAPHost string `json:"imapHost"`
		IMAPPort int    `json:"imapPort"`
		SMTPHost string `json:"smtpHost"`
		SMTPPort int    `json:"smtpPort"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !bind(c, &body) {
		c.String(http.StatusBadRequest, "invalid")
		return
	}
	body.Email = strings.TrimSpace(body.Email)
	if body.Username == "" {
		body.Username = body.Email
	}
	if body.IMAPPort == 0 {
		body.IMAPPort = 993
	}
	if body.SMTPPort == 0 {
		body.SMTPPort = 465
	}
	if !strings.Contains(body.Email, "@") || body.IMAPHost == "" || body.SMTPHost == "" || body.Password == "" {
		c.String(http.StatusBadRequest, "email, hosts and password required")
		return
	}
	acct, err := s.store.CreateMailAccount(c.Request.Context(), currentUser(c).ID, store.MailAccount{
		Name: body.Name, Email: body.Email,
		IMAPHost: body.IMAPHost, IMAPPort: body.IMAPPort,
		SMTPHost: body.SMTPHost, SMTPPort: body.SMTPPort,
		Username: body.Username,
	}, body.Password)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusCreated, acct)
}

func (s *Server) deleteMailAccount(c *gin.Context) {
	err := s.store.DeleteMailAccount(c.Request.Context(), currentUser(c).ID, c.Param("id"))
	if errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "not found")
		return
	} else if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.Status(http.StatusNoContent)
}

// testMailAccount verifies IMAP credentials by dialing + logging in.
func (s *Server) testMailAccount(c *gin.Context) {
	client, _, err := s.imapDial(c.Request.Context(), currentUser(c).ID, c.Param("id"))
	if err != nil {
		c.String(http.StatusBadGateway, err.Error())
		return
	}
	defer client.Close()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- Mailbox + message reads ---

func (s *Server) mailMailboxes(c *gin.Context) {
	client, _, err := s.imapDial(c.Request.Context(), currentUser(c).ID, c.Param("id"))
	if err != nil {
		c.String(http.StatusBadGateway, err.Error())
		return
	}
	defer client.Close()
	boxes, err := client.List("", "*", nil).Collect()
	if err != nil {
		c.String(http.StatusBadGateway, "list failed")
		return
	}
	out := []gin.H{}
	for _, b := range boxes {
		out = append(out, gin.H{"name": b.Mailbox, "delimiter": string(b.Delim)})
	}
	c.JSON(http.StatusOK, out)
}

type mailSummary struct {
	UID     uint32   `json:"uid"`
	From    string   `json:"from"`
	To      []string `json:"to,omitempty"`
	Subject string   `json:"subject"`
	Date    string   `json:"date"`
	Seen    bool     `json:"seen"`
	Size    int64    `json:"size"`
}

// mailMessages lists the newest page of envelopes in a mailbox.
func (s *Server) mailMessages(c *gin.Context) {
	mailbox := c.DefaultQuery("mailbox", "INBOX")
	page := 0
	fmt.Sscanf(c.DefaultQuery("page", "0"), "%d", &page)
	const pageSize = 40

	client, _, err := s.imapDial(c.Request.Context(), currentUser(c).ID, c.Param("id"))
	if err != nil {
		c.String(http.StatusBadGateway, err.Error())
		return
	}
	defer client.Close()

	selected, err := client.Select(mailbox, &imap.SelectOptions{ReadOnly: true}).Wait()
	if err != nil {
		c.String(http.StatusBadGateway, "select failed")
		return
	}
	total := selected.NumMessages
	if total == 0 {
		c.JSON(http.StatusOK, gin.H{"total": 0, "messages": []mailSummary{}})
		return
	}
	// Newest-first: sequence window from the end backwards.
	end := int(total) - page*pageSize
	start := end - pageSize + 1
	if start < 1 {
		start = 1
	}
	if end < 1 {
		c.JSON(http.StatusOK, gin.H{"total": total, "messages": []mailSummary{}})
		return
	}
	seq := imap.SeqSetNum(uint32(start), uint32(end))
	bufs, err := client.Fetch(seq, &imap.FetchOptions{
		Envelope:     true,
		Flags:        true,
		InternalDate: true,
		RFC822Size:   true,
	}).Collect()
	if err != nil {
		c.String(http.StatusBadGateway, "fetch failed")
		return
	}
	out := make([]mailSummary, 0, len(bufs))
	for i := len(bufs) - 1; i >= 0; i-- { // newest first
		b := bufs[i]
		m := mailSummary{UID: uint32(b.UID), Size: b.RFC822Size, Date: b.InternalDate.Format(time.RFC3339)}
		if b.Envelope != nil {
			m.Subject = b.Envelope.Subject
			if len(b.Envelope.From) > 0 {
				f := b.Envelope.From[0]
				if f.Name != "" {
					m.From = f.Name + " <" + f.Addr() + ">"
				} else {
					m.From = f.Addr()
				}
			}
			for _, t := range b.Envelope.To {
				m.To = append(m.To, t.Addr())
			}
		}
		for _, f := range b.Flags {
			if f == imap.FlagSeen {
				m.Seen = true
			}
		}
		out = append(out, m)
	}
	c.JSON(http.StatusOK, gin.H{"total": int(total), "messages": out})
}

// mailMessage fetches one message body by UID, preferring text/plain.
func (s *Server) mailMessage(c *gin.Context) {
	mailbox := c.DefaultQuery("mailbox", "INBOX")
	var uidNum uint32
	if _, err := fmt.Sscanf(c.Param("uid"), "%d", &uidNum); err != nil || uidNum == 0 {
		c.String(http.StatusBadRequest, "bad uid")
		return
	}
	client, _, err := s.imapDial(c.Request.Context(), currentUser(c).ID, c.Param("id"))
	if err != nil {
		c.String(http.StatusBadGateway, err.Error())
		return
	}
	defer client.Close()
	if _, err := client.Select(mailbox, nil).Wait(); err != nil {
		c.String(http.StatusBadGateway, "select failed")
		return
	}

	section := &imap.FetchItemBodySection{}
	bufs, err := client.Fetch(imap.UIDSetNum(imap.UID(uidNum)), &imap.FetchOptions{
		Envelope:    true,
		Flags:       true,
		BodySection: []*imap.FetchItemBodySection{section},
	}).Collect()
	if err != nil || len(bufs) == 0 {
		c.String(http.StatusNotFound, "message not found")
		return
	}
	raw := bytes.NewReader(bufs[0].FindBodySection(section))
	mr, err := mail.CreateReader(raw)
	if err != nil {
		c.String(http.StatusBadGateway, "parse failed")
		return
	}
	h := mr.Header
	from := ""
	if addrs, err := h.AddressList("From"); err == nil && len(addrs) > 0 {
		from = addrs[0].String()
	}
	subject, _ := h.Subject()
	to := []string{}
	if addrs, err := h.AddressList("To"); err == nil {
		for _, a := range addrs {
			to = append(to, a.String())
		}
	}

	// Walk parts; prefer plain text, keep html as fallback.
	var textPart, htmlPart string
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			ct, _, _ := h.ContentType()
			body, _ := io.ReadAll(io.LimitReader(p.Body, 2<<20))
			if ct == "text/plain" && textPart == "" {
				textPart = string(body)
			} else if ct == "text/html" && htmlPart == "" {
				htmlPart = string(body)
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"uid": uidNum, "from": from, "to": to, "subject": subject,
		"text": textPart, "html": htmlPart,
	})
}

// --- Flags / delete / send ---

func (s *Server) mailFlag(c *gin.Context) {
	mailbox := c.DefaultQuery("mailbox", "INBOX")
	var body struct {
		Seen bool `json:"seen"`
	}
	_ = c.ShouldBindJSON(&body)
	var uidNum uint32
	if _, err := fmt.Sscanf(c.Param("uid"), "%d", &uidNum); err != nil || uidNum == 0 {
		c.String(http.StatusBadRequest, "bad uid")
		return
	}
	client, _, err := s.imapDial(c.Request.Context(), currentUser(c).ID, c.Param("id"))
	if err != nil {
		c.String(http.StatusBadGateway, err.Error())
		return
	}
	defer client.Close()
	if _, err := client.Select(mailbox, nil).Wait(); err != nil {
		c.String(http.StatusBadGateway, "select failed")
		return
	}
	op := imap.StoreFlagsAdd
	if !body.Seen {
		op = imap.StoreFlagsDel
	}
	_, err = client.Store(imap.UIDSetNum(imap.UID(uidNum)), &imap.StoreFlags{
		Op: op, Silent: true, Flags: []imap.Flag{imap.FlagSeen},
	}, nil).Collect()
	if err != nil {
		c.String(http.StatusBadGateway, "flag failed")
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) mailDelete(c *gin.Context) {
	mailbox := c.DefaultQuery("mailbox", "INBOX")
	var uidNum uint32
	if _, err := fmt.Sscanf(c.Param("uid"), "%d", &uidNum); err != nil || uidNum == 0 {
		c.String(http.StatusBadRequest, "bad uid")
		return
	}
	client, _, err := s.imapDial(c.Request.Context(), currentUser(c).ID, c.Param("id"))
	if err != nil {
		c.String(http.StatusBadGateway, err.Error())
		return
	}
	defer client.Close()
	if _, err := client.Select(mailbox, nil).Wait(); err != nil {
		c.String(http.StatusBadGateway, "select failed")
		return
	}
	uids := imap.UIDSetNum(imap.UID(uidNum))
	// Prefer MOVE (RFC 6851); fall back to \Deleted + EXPUNGE.
	if _, err := client.Move(uids, "Trash").Wait(); err != nil {
		if _, err := client.Store(uids, &imap.StoreFlags{
			Op: imap.StoreFlagsAdd, Silent: true, Flags: []imap.Flag{imap.FlagDeleted},
		}, nil).Collect(); err != nil {
			c.String(http.StatusBadGateway, "delete failed")
			return
		}
		_, _ = client.UIDExpunge(uids).Collect()
	}
	c.Status(http.StatusNoContent)
}

// mailSend composes a plain-text RFC822 message and submits it over SMTP.
func (s *Server) mailSend(c *gin.Context) {
	var body struct {
		To      string `json:"to"`
		Cc      string `json:"cc"`
		Subject string `json:"subject"`
		Text    string `json:"text"`
	}
	if !bind(c, &body) || !strings.Contains(body.To, "@") {
		c.String(http.StatusBadRequest, "recipient required")
		return
	}
	acct, password, err := s.store.MailCredentials(c.Request.Context(), currentUser(c).ID, c.Param("id"))
	if err != nil {
		c.String(http.StatusNotFound, "unknown account")
		return
	}
	var msg bytes.Buffer
	fmt.Fprintf(&msg, "From: %s\r\nTo: %s\r\n", acct.Email, body.To)
	if body.Cc != "" {
		fmt.Fprintf(&msg, "Cc: %s\r\n", body.Cc)
	}
	fmt.Fprintf(&msg, "Subject: %s\r\nDate: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		sanitizeHeader(body.Subject), time.Now().Format(time.RFC1123Z), body.Text)

	addr := fmt.Sprintf("%s:%d", acct.SMTPHost, acct.SMTPPort)
	var client *smtp.Client
	if acct.SMTPPort == 465 {
		client, err = smtp.DialTLS(addr, nil)
	} else {
		client, err = smtp.DialStartTLS(addr, nil)
	}
	if err != nil {
		c.String(http.StatusBadGateway, "smtp dial failed")
		return
	}
	defer client.Close()
	if err := client.Auth(sasl.NewPlainClient("", acct.Username, password)); err != nil {
		c.String(http.StatusBadGateway, "smtp auth failed")
		return
	}
	rcpts := append(splitAddrs(body.To), splitAddrs(body.Cc)...)
	if err := client.SendMail(acct.Email, rcpts, &msg); err != nil {
		c.String(http.StatusBadGateway, "send failed")
		return
	}
	c.Status(http.StatusAccepted)
}

func sanitizeHeader(s string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(s)
}

func splitAddrs(s string) []string {
	out := []string{}
	for _, a := range strings.Split(s, ",") {
		if a = strings.TrimSpace(a); a != "" {
			out = append(out, a)
		}
	}
	return out
}
