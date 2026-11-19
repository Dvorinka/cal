package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func mcpToolText(t *testing.T, raw string) (string, bool) {
	t.Helper()
	var resp struct {
		Result struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("rpc: %v %s", err, raw)
	}
	if resp.Error != nil {
		t.Fatalf("rpc error: %s", raw)
	}
	if len(resp.Result.Content) == 0 {
		t.Fatalf("empty content: %s", raw)
	}
	return resp.Result.Content[0].Text, resp.Result.IsError
}

func TestMCPShoppingRoundTrip(t *testing.T) {
	srv, session := newTestServer(t)

	status, body := doJSONStatus(t, srv, http.MethodGet, "/api/mcp/tools", "", "")
	if status != http.StatusUnauthorized || !strings.Contains(body, "authentication required") {
		t.Fatalf("catalog without session: %d %s", status, body)
	}
	status, body = doJSONStatus(t, srv, http.MethodGet, "/api/mcp/tools", "", session)
	if status != http.StatusOK || !strings.Contains(body, "list_shopping") || !strings.Contains(body, mcpProtocol) {
		t.Fatalf("catalog: %d %s", status, body)
	}

	settings := doJSON(t, srv, http.MethodGet, "/api/settings", "", session)
	var st struct {
		ApiToken string `json:"apiToken"`
	}
	if err := json.Unmarshal([]byte(settings), &st); err != nil || st.ApiToken == "" {
		t.Fatalf("settings: %s", settings)
	}
	call := func(name, args string) (string, bool) {
		t.Helper()
		raw := doJSON(t, srv, http.MethodPost, "/api/mcp",
			fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":%q,"arguments":%s}}`, name, args),
			st.ApiToken)
		return mcpToolText(t, raw)
	}

	text, isErr := call("list_shopping", `{}`)
	if isErr || text != "[]" {
		t.Fatalf("empty lists: err=%v %s", isErr, text)
	}

	text, isErr = call("add_shopping_item", `{"name":"  Milk  ","quantity":"2"}`)
	if isErr || !strings.Contains(text, `"name":"Milk"`) {
		t.Fatalf("add: err=%v %s", isErr, text)
	}
	var added struct {
		ID     string `json:"id"`
		ListID string `json:"listId"`
	}
	if err := json.Unmarshal([]byte(text), &added); err != nil || added.ID == "" {
		t.Fatalf("added item: %s", text)
	}

	text, isErr = call("list_shopping", `{}`)
	if isErr || !strings.Contains(text, `"name":"Shopping"`) || !strings.Contains(text, `"open":1`) {
		t.Fatalf("lists after add: err=%v %s", isErr, text)
	}

	text, isErr = call("add_shopping_item", `{"name":"  "}`)
	if !isErr || !strings.Contains(text, "name required") {
		t.Fatalf("blank name: err=%v %s", isErr, text)
	}
	text, isErr = call("add_shopping_item", `{"listId":"00000000-0000-0000-0000-000000000000","name":"Nope"}`)
	if !isErr || !strings.Contains(text, "list not found") {
		t.Fatalf("foreign list: err=%v %s", isErr, text)
	}

	office := doJSON(t, srv, http.MethodPost, "/api/shopping/lists", `{"name":"Office"}`, session)
	var officeList struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(office), &officeList); err != nil || officeList.ID == "" {
		t.Fatalf("office list: %s", office)
	}
	section := doJSON(t, srv, http.MethodPost, "/api/shopping/lists/"+officeList.ID+"/sections", `{"name":"Desk"}`, session)
	var sec struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(section), &sec); err != nil || sec.ID == "" {
		t.Fatalf("section: %s", section)
	}
	text, isErr = call("add_shopping_item", fmt.Sprintf(`{"listId":%q,"sectionId":%q,"name":"Staples"}`, added.ListID, sec.ID))
	if !isErr || !strings.Contains(text, "section not found") {
		t.Fatalf("cross-list section: err=%v %s", isErr, text)
	}
	text, isErr = call("add_shopping_item", fmt.Sprintf(`{"listId":%q,"sectionId":%q,"name":"Staples"}`, officeList.ID, sec.ID))
	if isErr || !strings.Contains(text, `"name":"Staples"`) {
		t.Fatalf("sectioned add: err=%v %s", isErr, text)
	}

	detail, isErr := call("shopping_list", fmt.Sprintf(`{"listId":%q}`, officeList.ID))
	if isErr || !strings.Contains(detail, "Desk") || !strings.Contains(detail, "Staples") {
		t.Fatalf("detail: err=%v %s", isErr, detail)
	}

	text, isErr = call("set_shopping_item", fmt.Sprintf(`{"id":%q,"checked":true}`, added.ID))
	if isErr || !strings.Contains(text, `"checked":true`) {
		t.Fatalf("check: err=%v %s", isErr, text)
	}
	text, isErr = call("list_shopping", `{}`)
	if isErr || !strings.Contains(text, `"name":"Shopping"`) || !strings.Contains(text, `"open":0`) {
		t.Fatalf("open count: err=%v %s", isErr, text)
	}
	text, isErr = call("set_shopping_item", `{"id":"00000000-0000-0000-0000-000000000000","checked":true}`)
	if !isErr || !strings.Contains(text, "item not found") {
		t.Fatalf("missing item: err=%v %s", isErr, text)
	}
}
