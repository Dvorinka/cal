package httpapi

// Minimal MCP (Model Context Protocol) server over streamable HTTP.
// Implements initialize / ping / tools/list / tools/call with bearer-token
// auth, so any MCP client can drive a Cal instance directly.

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cal/apps/api/internal/store"

	"github.com/gin-gonic/gin"
)

const mcpProtocol = "2025-03-26"

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

func rpcResult(id json.RawMessage, result any) gin.H {
	return gin.H{"jsonrpc": "2.0", "id": id, "result": result}
}

func rpcError(id json.RawMessage, code int, msg string) gin.H {
	return gin.H{"jsonrpc": "2.0", "id": id, "error": gin.H{"code": code, "message": msg}}
}

func toolText(text string, isErr bool) gin.H {
	return gin.H{"content": []gin.H{{"type": "text", "text": text}}, "isError": isErr}
}

var mcpTools = []gin.H{
	{
		"name":        "list_entries",
		"description": "List planner entries (tasks, notes, links, events) in a date range or by search query.",
		"inputSchema": gin.H{
			"type": "object",
			"properties": gin.H{
				"from": gin.H{"type": "string", "description": "Start date YYYY-MM-DD (optional)"},
				"to":   gin.H{"type": "string", "description": "End date YYYY-MM-DD (optional)"},
				"q":    gin.H{"type": "string", "description": "Search query (optional)"},
			},
		},
	},
	{
		"name":        "today",
		"description": "Get today's entries: the daily agenda.",
		"inputSchema": gin.H{"type": "object", "properties": gin.H{}},
	},
	{
		"name":        "create_entry",
		"description": "Create a task, note, link or event. Times are HH:MM, date is YYYY-MM-DD. Recur: none|daily|weekly|monthly|yearly (tasks only). remind: minutes before start.",
		"inputSchema": gin.H{
			"type": "object",
			"properties": gin.H{
				"title":     gin.H{"type": "string"},
				"type":      gin.H{"type": "string", "enum": []string{"task", "note", "link", "event"}},
				"date":      gin.H{"type": "string", "description": "YYYY-MM-DD"},
				"startTime": gin.H{"type": "string", "description": "HH:MM"},
				"endTime":   gin.H{"type": "string", "description": "HH:MM"},
				"color":     gin.H{"type": "string", "enum": []string{"slate", "mint", "sky", "violet", "amber", "orange", "rose", "red"}},
				"tags":      gin.H{"type": "array", "items": gin.H{"type": "string"}},
				"content":   gin.H{"type": "string"},
				"linkUrl":   gin.H{"type": "string"},
				"recur":     gin.H{"type": "string", "enum": []string{"none", "daily", "weekly", "monthly", "yearly"}},
				"remind":    gin.H{"type": "integer", "description": "minutes before start to remind"},
			},
			"required": []string{"title", "type", "date"},
		},
	},
	{
		"name":        "update_entry",
		"description": "Patch an entry. Only provided fields change. Set completed=true on a recurring task to spawn the next occurrence.",
		"inputSchema": gin.H{
			"type": "object",
			"properties": gin.H{
				"id":        gin.H{"type": "string"},
				"title":     gin.H{"type": "string"},
				"date":      gin.H{"type": "string"},
				"startTime": gin.H{"type": "string"},
				"endTime":   gin.H{"type": "string"},
				"completed": gin.H{"type": "boolean"},
				"color":     gin.H{"type": "string"},
				"tags":      gin.H{"type": "array", "items": gin.H{"type": "string"}},
				"content":   gin.H{"type": "string"},
				"recur":     gin.H{"type": "string"},
				"remind":    gin.H{"type": "integer"},
			},
			"required": []string{"id"},
		},
	},
	{
		"name":        "delete_entry",
		"description": "Delete an entry by id.",
		"inputSchema": gin.H{
			"type":       "object",
			"properties": gin.H{"id": gin.H{"type": "string"}},
			"required":   []string{"id"},
		},
	},
	{
		"name":        "list_feeds",
		"description": "List subscribed external calendar feeds.",
		"inputSchema": gin.H{"type": "object", "properties": gin.H{}},
	},
	{
		"name":        "get_entry",
		"description": "Get a single entry with its full content (e.g. a note's markdown body).",
		"inputSchema": gin.H{
			"type":       "object",
			"properties": gin.H{"id": gin.H{"type": "string"}},
			"required":   []string{"id"},
		},
	},
	{
		"name":        "append_note",
		"description": "Append markdown text to a note's body (or any entry's content). The user never has to open the editor.",
		"inputSchema": gin.H{
			"type": "object",
			"properties": gin.H{
				"id":   gin.H{"type": "string"},
				"text": gin.H{"type": "string", "description": "Markdown to append"},
			},
			"required": []string{"id", "text"},
		},
	},
	{
		"name":        "search_entries",
		"description": "Full-text search across titles and content; matches tags exactly.",
		"inputSchema": gin.H{
			"type":       "object",
			"properties": gin.H{"q": gin.H{"type": "string"}},
			"required":   []string{"q"},
		},
	},
	{
		"name":        "list_accounts",
		"description": "List connected CalDAV calendar accounts.",
		"inputSchema": gin.H{"type": "object", "properties": gin.H{}},
	},
	{
		"name":        "complete_task",
		"description": "Mark a task done; recurring tasks spawn their next occurrence.",
		"inputSchema": gin.H{
			"type":       "object",
			"properties": gin.H{"id": gin.H{"type": "string"}},
			"required":   []string{"id"},
		},
	},
	{
		"name":        "weekly_review",
		"description": "Digest of the past week: tasks done, slipped, notes written, streak, busiest day.",
		"inputSchema": gin.H{"type": "object", "properties": gin.H{}},
	},
	{
		"name":        "list_habits",
		"description": "Habit streaks for recurring tasks tagged #habit.",
		"inputSchema": gin.H{"type": "object", "properties": gin.H{}},
	},
	{
		"name":        "list_files",
		"description": "List uploaded files with their public share tokens.",
		"inputSchema": gin.H{"type": "object", "properties": gin.H{}},
	},
	{
		"name":        "list_boards",
		"description": "List kanban boards.",
		"inputSchema": gin.H{"type": "object", "properties": gin.H{}},
	},
	{
		"name":        "start_timer",
		"description": "Start the focus timer, optionally on a task entry or project board.",
		"inputSchema": gin.H{
			"type": "object",
			"properties": gin.H{
				"entryId":   gin.H{"type": "string"},
				"note":      gin.H{"type": "string"},
				"planned":   gin.H{"type": "integer", "description": "pomodoro minutes"},
				"billable":  gin.H{"type": "boolean"},
				"projectId": gin.H{"type": "string", "description": "board id"},
			},
		},
	},
	{
		"name":        "stop_timer",
		"description": "Stop the running focus timer.",
		"inputSchema": gin.H{"type": "object", "properties": gin.H{}},
	},
	{
		"name":        "time_summary",
		"description": "Minutes tracked today/this week + billable amount + per-task breakdown.",
		"inputSchema": gin.H{"type": "object", "properties": gin.H{}},
	},
	{
		"name":        "github_inbox",
		"description": "Open GitHub issues and PRs involving the user (needs a PAT in settings).",
		"inputSchema": gin.H{"type": "object", "properties": gin.H{}},
	},
	{
		"name":        "board_view",
		"description": "Columns and cards of a board.",
		"inputSchema": gin.H{
			"type":       "object",
			"properties": gin.H{"boardId": gin.H{"type": "string"}},
			"required":   []string{"boardId"},
		},
	},
	{
		"name":        "create_card",
		"description": "Create a task on a board column (title required; date defaults to today).",
		"inputSchema": gin.H{
			"type": "object",
			"properties": gin.H{
				"boardId":  gin.H{"type": "string"},
				"columnId": gin.H{"type": "string"},
				"title":    gin.H{"type": "string"},
				"date":     gin.H{"type": "string"},
				"content":  gin.H{"type": "string"},
			},
			"required": []string{"boardId", "title"},
		},
	},
	{
		"name":        "move_card",
		"description": "Move a card to another column (position optional).",
		"inputSchema": gin.H{
			"type": "object",
			"properties": gin.H{
				"id":       gin.H{"type": "string"},
				"columnId": gin.H{"type": "string"},
				"position": gin.H{"type": "number"},
			},
			"required": []string{"id", "columnId"},
		},
	},
}

func (s *Server) mcpAuth(c *gin.Context) (store.User, bool) {
	token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	if token == "" || token == c.GetHeader("Authorization") {
		return store.User{}, false
	}
	user, err := s.store.UserByApiToken(c.Request.Context(), token)
	if err != nil {
		return store.User{}, false
	}
	return user, true
}

func (s *Server) mcp(c *gin.Context) {
	user, ok := s.mcpAuth(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid api token"})
		return
	}
	var req rpcRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.JSONRPC != "2.0" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON-RPC request"})
		return
	}

	switch {
	case req.Method == "initialize":
		c.JSON(http.StatusOK, rpcResult(req.ID, gin.H{
			"protocolVersion": mcpProtocol,
			"capabilities": gin.H{
				"tools":     gin.H{"listChanged": false},
				"resources": gin.H{"listChanged": false},
				"prompts":   gin.H{"listChanged": false},
			},
			"serverInfo": gin.H{"name": "cal", "version": "1.0.0"},
		}))
	case strings.HasPrefix(req.Method, "notifications/"):
		c.Status(http.StatusAccepted)
	case req.Method == "ping":
		c.JSON(http.StatusOK, rpcResult(req.ID, gin.H{}))
	case req.Method == "tools/list":
		c.JSON(http.StatusOK, rpcResult(req.ID, gin.H{"tools": mcpTools}))
	case req.Method == "tools/call":
		s.mcpCall(c, user, req)
	case req.Method == "resources/list":
		// One live resource per logical collection — entries themselves are
		// fetched via tools so the list stays bounded.
		c.JSON(http.StatusOK, rpcResult(req.ID, gin.H{"resources": []gin.H{
			{"uri": "cal://today", "name": "Today", "description": "Today's agenda as JSON", "mimeType": "application/json"},
			{"uri": "cal://week", "name": "This week", "description": "Entries for the current week", "mimeType": "application/json"},
			{"uri": "cal://open-tasks", "name": "Open tasks", "description": "Every incomplete task", "mimeType": "application/json"},
		}}))
	case req.Method == "resources/read":
		var p struct {
			URI string `json:"uri"`
		}
		_ = json.Unmarshal(req.Params, &p)
		ctx := c.Request.Context()
		var entries []store.Entry
		var err error
		today := time.Now().Format(time.DateOnly)
		switch p.URI {
		case "cal://today":
			entries, err = s.store.ListEntries(ctx, user.ID, today, today, "")
		case "cal://week":
			end := time.Now().AddDate(0, 0, 7).Format(time.DateOnly)
			entries, err = s.store.ListEntries(ctx, user.ID, today, end, "")
		case "cal://open-tasks":
			entries, err = s.store.ListEntries(ctx, user.ID, "", "", "")
			open := entries[:0]
			for _, e := range entries {
				if e.Type == "task" && !e.Completed {
					open = append(open, e)
				}
			}
			entries = open
		default:
			c.JSON(http.StatusOK, rpcError(req.ID, -32602, "unknown resource"))
			return
		}
		if err != nil {
			c.JSON(http.StatusOK, rpcError(req.ID, -32603, "query failed"))
			return
		}
		data, _ := json.Marshal(entries)
		c.JSON(http.StatusOK, rpcResult(req.ID, gin.H{"contents": []gin.H{
			{"uri": p.URI, "mimeType": "application/json", "text": string(data)},
		}}))
	case req.Method == "prompts/list":
		c.JSON(http.StatusOK, rpcResult(req.ID, gin.H{"prompts": []gin.H{
			{"name": "daily-plan", "description": "Turn today's agenda into a realistic plan", "arguments": []gin.H{}},
			{"name": "weekly-review", "description": "Summarize the past week and propose next week's focus", "arguments": []gin.H{}},
		}}))
	case req.Method == "prompts/get":
		var p struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(req.Params, &p)
		texts := map[string]string{
			"daily-plan":    "Here is today's agenda (cal://today). Help me plan the day: order the timed blocks realistically, flag what's overpacked, and suggest which tasks to defer. After we agree, update entries or add new ones via the tools.",
			"weekly-review": "Pull cal://week plus all open tasks (cal://open-tasks). Summarize what I completed, what slipped, and propose three focus items for next week. Offer to create the review as a note entry.",
		}
		text, ok := texts[p.Name]
		if !ok {
			c.JSON(http.StatusOK, rpcError(req.ID, -32602, "unknown prompt"))
			return
		}
		c.JSON(http.StatusOK, rpcResult(req.ID, gin.H{
			"description": p.Name,
			"messages":    []gin.H{{"role": "user", "content": gin.H{"type": "text", "text": text}}},
		}))
	default:
		c.JSON(http.StatusOK, rpcError(req.ID, -32601, "method not found"))
	}
}

func (s *Server) mcpCall(c *gin.Context, user store.User, req rpcRequest) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		c.JSON(http.StatusOK, rpcError(req.ID, -32602, "invalid params"))
		return
	}
	ctx := c.Request.Context()
	respond := func(result gin.H) { c.JSON(http.StatusOK, rpcResult(req.ID, result)) }
	fail := func(msg string) { respond(toolText(msg, true)) }

	switch params.Name {
	case "list_entries", "today":
		var args struct {
			From string `json:"from"`
			To   string `json:"to"`
			Q    string `json:"q"`
		}
		_ = json.Unmarshal(params.Arguments, &args)
		if params.Name == "today" {
			today := time.Now().Format(time.DateOnly)
			args.From, args.To = today, today
		}
		entries, err := s.store.ListEntries(ctx, user.ID, args.From, args.To, args.Q)
		if err != nil {
			fail("query failed")
			return
		}
		data, _ := json.Marshal(entries)
		respond(toolText(string(data), false))

	case "create_entry":
		var input store.EntryInput
		if err := json.Unmarshal(params.Arguments, &input); err != nil {
			fail("invalid arguments")
			return
		}
		if err := validateEntryInput(&input); err != nil {
			fail(err.Error())
			return
		}
		entry, err := s.store.CreateEntry(ctx, user.ID, input)
		if err != nil {
			fail("create failed: " + err.Error())
			return
		}
		data, _ := json.Marshal(entry)
		respond(toolText(string(data), false))

	case "update_entry":
		// Args carry the patch fields alongside id.
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(params.Arguments, &raw); err != nil {
			fail("invalid arguments")
			return
		}
		var id string
		if v, ok := raw["id"]; ok {
			_ = json.Unmarshal(v, &id)
			delete(raw, "id")
		}
		if id == "" {
			fail("id is required")
			return
		}
		cleaned, _ := json.Marshal(raw)
		var entryPatch store.EntryPatch
		if err := json.Unmarshal(cleaned, &entryPatch); err != nil {
			fail("invalid arguments")
			return
		}
		if _, ok := raw["tags"]; ok {
			entryPatch.HasTags = true
		}
		updated, err := s.store.UpdateEntry(ctx, user.ID, id, entryPatch)
		if errors.Is(err, store.ErrNotFound) {
			fail("entry not found")
			return
		} else if errors.Is(err, store.ErrInvalid) {
			fail("invalid update")
			return
		} else if err != nil {
			fail("update failed")
			return
		}
		data, _ := json.Marshal(updated)
		respond(toolText(string(data), false))

	case "delete_entry":
		var args struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params.Arguments, &args)
		if err := s.store.DeleteEntry(ctx, user.ID, args.ID); err != nil {
			fail("entry not found")
			return
		}
		respond(toolText("deleted", false))

	case "list_feeds":
		feeds, err := s.store.ListFeeds(ctx, user.ID)
		if err != nil {
			fail("query failed")
			return
		}
		data, _ := json.Marshal(feeds)
		respond(toolText(string(data), false))

	case "get_entry", "append_note":
		var args struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		}
		_ = json.Unmarshal(params.Arguments, &args)
		if args.ID == "" {
			fail("id is required")
			return
		}
		entry, err := s.store.Entry(ctx, user.ID, args.ID)
		if err != nil {
			fail("entry not found")
			return
		}
		if params.Name == "get_entry" {
			data, _ := json.Marshal(entry)
			respond(toolText(string(data), false))
			return
		}
		if args.Text == "" {
			fail("text is required")
			return
		}
		body := entry.Content
		if body != "" && !strings.HasSuffix(body, "\n") {
			body += "\n"
		}
		_, err = s.store.UpdateEntry(ctx, user.ID, args.ID, store.EntryPatch{Content: &[]string{body + args.Text}[0]})
		if err != nil {
			fail("update failed")
			return
		}
		respond(toolText("appended", false))

	case "search_entries":
		var args struct {
			Q string `json:"q"`
		}
		_ = json.Unmarshal(params.Arguments, &args)
		if strings.TrimSpace(args.Q) == "" {
			fail("q is required")
			return
		}
		entries, err := s.store.ListEntries(ctx, user.ID, "", "", args.Q)
		if err != nil {
			fail("query failed")
			return
		}
		data, _ := json.Marshal(entries)
		respond(toolText(string(data), false))

	case "list_accounts":
		accounts, err := s.store.CaldavAccounts(ctx, user.ID)
		if err != nil {
			fail("query failed")
			return
		}
		data, _ := json.Marshal(accounts)
		respond(toolText(string(data), false))

	case "complete_task":
		var args struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params.Arguments, &args)
		entry, err := s.store.Entry(ctx, user.ID, args.ID)
		if err != nil {
			fail("entry not found")
			return
		}
		if entry.Type != "task" {
			fail("only tasks can be completed")
			return
		}
		done := true
		updated, err := s.store.UpdateEntry(ctx, user.ID, args.ID, store.EntryPatch{Completed: &done})
		if err != nil {
			fail("update failed")
			return
		}
		data, _ := json.Marshal(updated)
		respond(toolText(string(data), false))

	case "weekly_review":
		review, err := s.weekReview(ctx, user.ID)
		if err != nil {
			fail("review failed")
			return
		}
		data, _ := json.Marshal(review)
		respond(toolText(string(data), false))

	case "list_habits":
		habits, err := s.habitStreakList(ctx, user.ID)
		if err != nil {
			fail("query failed")
			return
		}
		data, _ := json.Marshal(habits)
		respond(toolText(string(data), false))

	case "list_files":
		files, err := s.store.ListFiles(ctx, user.ID)
		if err != nil {
			fail("query failed")
			return
		}
		data, _ := json.Marshal(files)
		respond(toolText(string(data), false))

	case "list_boards":
		boards, err := s.store.ListBoards(ctx, user.ID)
		if err != nil {
			fail("query failed")
			return
		}
		data, _ := json.Marshal(boards)
		respond(toolText(string(data), false))

	case "start_timer":
		var args struct {
			EntryID   string `json:"entryId"`
			Note      string `json:"note"`
			Planned   int    `json:"planned"`
			Billable  bool   `json:"billable"`
			ProjectID string `json:"projectId"`
		}
		_ = json.Unmarshal(params.Arguments, &args)
		var ePtr, pPtr *string
		if args.EntryID != "" {
			ePtr = &args.EntryID
		}
		if args.ProjectID != "" {
			pPtr = &args.ProjectID
		}
		var rate *float64
		if st, err := s.store.Settings(ctx, user.ID); err == nil {
			rate = st.DefaultRate
		}
		t, err := s.store.StartTimer(ctx, user.ID, ePtr, args.Note, args.Planned, args.Billable, rate, pPtr)
		if err != nil {
			fail("a timer is already running")
			return
		}
		data, _ := json.Marshal(t)
		respond(toolText(string(data), false))

	case "stop_timer":
		t, err := s.store.StopTimer(ctx, user.ID)
		if err != nil {
			fail("no running timer")
			return
		}
		data, _ := json.Marshal(t)
		respond(toolText(string(data), false))

	case "time_summary":
		sum, err := s.store.TimeSummary(ctx, user.ID)
		if err != nil {
			fail("query failed")
			return
		}
		data, _ := json.Marshal(sum)
		respond(toolText(string(data), false))

	case "github_inbox":
		st, err := s.store.Settings(ctx, user.ID)
		if err != nil || st.GithubToken == "" {
			fail("no github token in settings")
			return
		}
		var res struct {
			Items []ghIssue `json:"items"`
		}
		if err := ghGet(ctx, st.GithubToken, "/search/issues?q="+url.QueryEscape("is:open involves:@me")+"&per_page=50", &res); err != nil {
			fail("github fetch failed")
			return
		}
		data, _ := json.Marshal(res.Items)
		respond(toolText(string(data), false))

	case "board_view":
		var args struct {
			BoardID string `json:"boardId"`
		}
		_ = json.Unmarshal(params.Arguments, &args)
		cols, _ := s.store.BoardColumns(ctx, user.ID, args.BoardID)
		cards, _ := s.store.BoardCards(ctx, user.ID, args.BoardID)
		data, _ := json.Marshal(gin.H{"columns": cols, "cards": cards})
		respond(toolText(string(data), false))

	case "create_card":
		var args struct {
			BoardID  string `json:"boardId"`
			ColumnID string `json:"columnId"`
			Title    string `json:"title"`
			Date     string `json:"date"`
			Content  string `json:"content"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil || args.BoardID == "" || strings.TrimSpace(args.Title) == "" {
			fail("boardId and title are required")
			return
		}
		if !s.store.BoardExists(ctx, user.ID, args.BoardID) {
			fail("unknown board")
			return
		}
		if args.Date == "" {
			args.Date = time.Now().Format("2006-01-02")
		}
		input := store.EntryInput{Title: args.Title, Type: "task", Date: args.Date, Content: args.Content, BoardID: &args.BoardID}
		if args.ColumnID != "" {
			input.ColumnID = &args.ColumnID
		}
		// Append at the column's end.
		cards, _ := s.store.BoardCards(ctx, user.ID, args.BoardID)
		top := 0.0
		for _, card := range cards {
			if card.ColumnID != nil && args.ColumnID != "" && *card.ColumnID == args.ColumnID && card.Position != nil && *card.Position > top {
				top = *card.Position
			}
		}
		pos := top + 1024
		input.Position = &pos
		entry, err := s.store.CreateEntry(ctx, user.ID, input)
		if err != nil {
			fail("create failed")
			return
		}
		data, _ := json.Marshal(entry)
		respond(toolText(string(data), false))

	case "move_card":
		var args struct {
			ID       string  `json:"id"`
			ColumnID string  `json:"columnId"`
			Position float64 `json:"position"`
		}
		_ = json.Unmarshal(params.Arguments, &args)
		entry, err := s.store.Entry(ctx, user.ID, args.ID)
		if err != nil || entry.BoardID == nil {
			fail("card not found")
			return
		}
		if err := s.store.MoveCard(ctx, user.ID, args.ID, *entry.BoardID, &args.ColumnID, args.Position); err != nil {
			fail("move failed")
			return
		}
		respond(toolText("moved", false))

	default:
		c.JSON(http.StatusOK, rpcError(req.ID, -32602, "unknown tool"))
	}
}

// validateEntryInput mirrors the create handler's validation for tool calls.
func validateEntryInput(input *store.EntryInput) error {
	if strings.TrimSpace(input.Title) == "" {
		return errors.New("title is required")
	}
	if input.Type == "" {
		input.Type = "task"
	}
	switch input.Type {
	case "task", "note", "link", "event":
	default:
		return errors.New("type must be task, note, link or event")
	}
	if input.Date == "" {
		return errors.New("date is required")
	}
	if input.Recur == "" {
		input.Recur = "none"
	}
	if input.Recur != "none" && input.Type != "task" {
		return errors.New("only tasks can recur")
	}
	return nil
}
