package conversation

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	appmemory "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/memory"
	domainmemory "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/memory"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/llm"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/mcp"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
)

type conversationMemoryStub struct {
	items  map[uint]domainmemory.UserMemory
	nextID uint
	audits []appmemory.AuditInput
}

func (s *conversationMemoryStub) RecordAudit(_ context.Context, input appmemory.AuditInput) {
	s.audits = append(s.audits, input)
}

func newConversationMemoryStub() *conversationMemoryStub {
	return &conversationMemoryStub{items: map[uint]domainmemory.UserMemory{}, nextID: 1}
}

func (s *conversationMemoryStub) AddUserMemory(_ context.Context, userID uint, content string, updatedBy string) (*domainmemory.UserMemory, error) {
	item := domainmemory.UserMemory{ID: s.nextID, UserID: userID, MemoryKey: "memory:test", Value: strings.TrimSpace(content), Scope: "memory", UpdatedBy: updatedBy}
	s.nextID++
	s.items[item.ID] = item
	return &item, nil
}

func (s *conversationMemoryStub) UpdateUserMemory(_ context.Context, userID uint, memoryID uint, content string, updatedBy string) (*domainmemory.UserMemory, error) {
	item, ok := s.items[memoryID]
	if !ok || item.UserID != userID {
		return nil, repository.ErrNotFound
	}
	item.Value = strings.TrimSpace(content)
	item.UpdatedBy = updatedBy
	s.items[memoryID] = item
	return &item, nil
}

func (s *conversationMemoryStub) DeleteUserMemoryByID(_ context.Context, userID uint, memoryID uint) error {
	item, ok := s.items[memoryID]
	if !ok || item.UserID != userID {
		return repository.ErrNotFound
	}
	delete(s.items, memoryID)
	return nil
}

func (s *conversationMemoryStub) UpsertUserMemory(_ context.Context, _ uint, _ string, _ string, _ string, _ string) error {
	return nil
}

func (s *conversationMemoryStub) ListUserMemories(_ context.Context, userID uint) ([]domainmemory.UserMemory, error) {
	result := make([]domainmemory.UserMemory, 0)
	for _, item := range s.items {
		if item.UserID == userID {
			result = append(result, item)
		}
	}
	return result, nil
}

func (s *conversationMemoryStub) SearchUserMemoriesByEmbedding(_ context.Context, _ uint, _ []float32, _ int, _ float64) ([]domainmemory.UserMemory, error) {
	return nil, nil
}

func (s *conversationMemoryStub) UpsertUserMemoryEmbedding(_ context.Context, _ uint, _ string, _ string, _ []float32) error {
	return nil
}

func TestWithMemoryToolsAddsPortableDefinitions(t *testing.T) {
	runtime := withMemoryTools(selectedToolRuntime{}, true)
	if len(runtime.definitions) != 3 {
		t.Fatalf("expected three memory tools, got %#v", runtime.definitions)
	}
	want := []string{memoryAddToolName, memoryUpdateToolName, memoryDeleteToolName}
	for index, name := range want {
		if runtime.definitions[index].Name != name {
			t.Fatalf("expected tool %d to be %s, got %#v", index, name, runtime.definitions[index])
		}
		if _, ok := runtime.memoryTools[name]; !ok {
			t.Fatalf("expected %s to be registered as internal memory tool", name)
		}
		if len(runtime.schemas[name]) == 0 {
			t.Fatalf("expected schema for %s", name)
		}
		if name != memoryDeleteToolName {
			var schema struct {
				Properties map[string]struct {
					MaxLength int `json:"maxLength"`
				} `json:"properties"`
			}
			if err := json.Unmarshal(runtime.schemas[name], &schema); err != nil {
				t.Fatalf("decode schema for %s: %v", name, err)
			}
			if schema.Properties["content"].MaxLength != 150 {
				t.Fatalf("expected %s content maxLength 150, got %#v", name, schema.Properties["content"])
			}
		}
	}
}

func TestWithMemoryToolsRenamesConflictingMCPTool(t *testing.T) {
	runtime := selectedToolRuntime{
		definitions:  []llm.ToolDefinition{{Name: memoryAddToolName, Description: "external"}},
		nameMap:      map[string]string{memoryAddToolName: "memory.add"},
		mcpConfigs:   map[string]mcp.CallConfig{memoryAddToolName: {BaseURL: "https://example.com"}},
		schemas:      map[string]json.RawMessage{memoryAddToolName: json.RawMessage(`{"type":"object"}`)},
		mcpToolCount: 1,
	}
	result := withMemoryTools(runtime, true)
	if len(result.definitions) != 4 || result.definitions[3].Name != "memory_add_2" {
		t.Fatalf("expected conflicting MCP tool to be renamed, got %#v", result.definitions)
	}
	if result.nameMap["memory_add_2"] != "memory.add" {
		t.Fatalf("expected renamed tool to retain execution name, got %#v", result.nameMap)
	}
	if _, ok := result.mcpConfigs["memory_add_2"]; !ok {
		t.Fatalf("expected renamed MCP config, got %#v", result.mcpConfigs)
	}
}

func TestExecuteMemoryToolAddUpdateDelete(t *testing.T) {
	memories := newConversationMemoryStub()
	svc := &Service{memoryRecorder: memories}

	addedRaw, err := svc.executeMemoryTool(context.Background(), 7, "req-1", memoryAddToolName, `{"content":"用户长期使用 Ubuntu"}`)
	if err != nil {
		t.Fatalf("memory_add error = %v", err)
	}
	var added map[string]interface{}
	if err := json.Unmarshal([]byte(addedRaw), &added); err != nil {
		t.Fatalf("decode add result: %v", err)
	}
	if added["ok"] != true || added["action"] != "added" || added["memory_id"] != float64(1) {
		t.Fatalf("unexpected add result: %#v", added)
	}
	if len(memories.audits) != 1 || memories.audits[0].Action != "add_user_memory" || memories.audits[0].MemoryKey != "1" {
		t.Fatalf("expected successful add audit, got %#v", memories.audits)
	}

	updatedRaw, err := svc.executeMemoryTool(context.Background(), 7, "req-1", memoryUpdateToolName, `{"old_memory_id":1,"content":"用户长期使用 Fedora"}`)
	if err != nil {
		t.Fatalf("memory_update error = %v", err)
	}
	if !strings.Contains(updatedRaw, `"action":"updated"`) || memories.items[1].Value != "用户长期使用 Fedora" {
		t.Fatalf("unexpected update result=%s memory=%#v", updatedRaw, memories.items[1])
	}

	deletedRaw, err := svc.executeMemoryTool(context.Background(), 7, "req-1", memoryDeleteToolName, `{"memory_id":1}`)
	if err != nil {
		t.Fatalf("memory_delete error = %v", err)
	}
	if !strings.Contains(deletedRaw, `"action":"deleted"`) || len(memories.items) != 0 {
		t.Fatalf("unexpected delete result=%s items=%#v", deletedRaw, memories.items)
	}
}

func TestExecuteMemoryToolRejectsOtherUsersMemory(t *testing.T) {
	memories := newConversationMemoryStub()
	memories.items[4] = domainmemory.UserMemory{ID: 4, UserID: 8, Value: "private"}
	svc := &Service{memoryRecorder: memories}

	if _, err := svc.executeMemoryTool(context.Background(), 7, "req-2", memoryDeleteToolName, `{"memory_id":4}`); err != repository.ErrNotFound {
		t.Fatalf("expected ownership-safe not found, got %v", err)
	}
}

func TestExecuteAssistantToolCallsAllowsEnabledMemoryTool(t *testing.T) {
	memories := newConversationMemoryStub()
	svc := &Service{memoryRecorder: memories}
	runtime := withMemoryTools(selectedToolRuntime{}, true)

	result := svc.executeAssistantToolCalls(context.Background(), executeAssistantToolCallsInput{
		UserID: 7,
		ToolCalls: []llm.ToolCall{{
			ToolCallID:    "call_memory_1",
			ToolName:      memoryAddToolName,
			ArgumentsJSON: `{"content":"用户偏好：使用中文"}`,
		}},
		ToolNameMap: runtime.nameMap,
		ToolSchemas: runtime.schemas,
		MemoryTools: runtime.memoryTools,
		Ledger:      newToolExecutionLedger(),
	})
	if result.FatalErr != nil {
		t.Fatalf("expected internal memory tool to execute, got %v", result.FatalErr)
	}
	if len(result.Rows) != 1 || result.Rows[0].Status != "success" {
		t.Fatalf("expected successful tool row, got %#v", result.Rows)
	}
	if len(memories.items) != 1 {
		t.Fatalf("expected memory to be added, got %#v", memories.items)
	}
}

func TestMemoryGuidanceStaysAfterMainSystemPrompt(t *testing.T) {
	messages := []llm.Message{{Role: "system", Content: "MAIN SYSTEM"}, {Role: "user", Content: "hello"}}
	result := injectMemoryToolGuidance(messages, true)
	if len(result) != 3 || result[0].Content != "MAIN SYSTEM" {
		t.Fatalf("expected main system prompt to stay first, got %#v", result)
	}
	if !strings.Contains(result[1].Content, "# 记忆管理规则") || !strings.Contains(result[1].Content, "只有在记忆工具成功执行后") {
		t.Fatalf("expected exact memory guidance after main system prompt, got %#v", result[1])
	}
}

func TestMemoryContextUsesIDAndMarksContentAsReferenceData(t *testing.T) {
	contextXML := buildUserContextXML(userContextInput{Memory: []domainmemory.UserMemory{{ID: 42, MemoryKey: "legacy-name", Value: "用户长期使用 Ubuntu", Scope: "preference"}}})
	prompt := buildUserContextPrompt("继续", contextXML)
	for _, want := range []string{"<memories", `id="42"`, "用户长期使用 Ubuntu", "仅作为背景资料", "<q>继续</q>"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("expected prompt to contain %q, got %q", want, prompt)
		}
	}
	if strings.Contains(prompt, "legacy-name") {
		t.Fatalf("expected internal legacy key to stay hidden, got %q", prompt)
	}
}

var _ memoryRecorder = (*conversationMemoryStub)(nil)
var _ = appmemory.MaxUserMemories
