package httpapi

import (
	"testing"

	"github.com/gin-gonic/gin"
)

// The MCP surface is the agent contract: every tool name must be unique and
// every declared required field must exist in properties.
func TestMCPToolsWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, tool := range mcpTools {
		name, _ := tool["name"].(string)
		if name == "" || seen[name] {
			t.Fatalf("bad or duplicate tool name %q", name)
		}
		seen[name] = true
		schema, _ := tool["inputSchema"].(gin.H)
		props, _ := schema["properties"].(gin.H)
		required, _ := schema["required"].([]string)
		for _, r := range required {
			if _, ok := props[r]; !ok {
				t.Fatalf("%s: required field %q missing from properties", name, r)
			}
		}
	}
}

// People parity: agents must reach the same person surface humans have
// (CRUD + relations + timeline), per roadmap principle 2.
func TestMCPPeopleParity(t *testing.T) {
	want := []string{
		"list_people", "create_person", "update_person",
		"delete_person", "person_upcoming",
		"list_person_relations", "link_people", "unlink_people",
		"person_timeline", "add_timeline_item", "update_timeline_item",
		"delete_timeline_item",
	}
	seen := map[string]bool{}
	for _, tool := range mcpTools {
		if name, _ := tool["name"].(string); name != "" {
			seen[name] = true
		}
	}
	for _, name := range want {
		if !seen[name] {
			t.Fatalf("tool %q not registered", name)
		}
	}
}
