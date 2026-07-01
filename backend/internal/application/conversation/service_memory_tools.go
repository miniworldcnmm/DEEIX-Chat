package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	appmemory "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/memory"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/llm"
)

const (
	memoryAddToolName    = "memory_add"
	memoryUpdateToolName = "memory_update"
	memoryDeleteToolName = "memory_delete"
)

const memoryManagementPrompt = `# 记忆管理规则

你拥有一个用于长期保存用户信息的记忆工具。每次收到用户消息时，都必须先判断其中是否包含值得长期保存、更新或删除的信息。

## 必须写入记忆的情况

出现以下任一情况时，必须调用记忆工具，不得只在回复中口头表示已经记住：

1. 用户明确要求记忆，例如：

   * “记住……”
   * “以后都……”
   * “从现在开始……”
   * “下次别忘了……”
   * “把这个保存下来”
   * “加入记忆”

2. 用户明确要求删除记忆，例如：

   * “忘掉……”
   * “不要再记这个”
   * “删除关于……的记忆”

3. 用户提供了未来很可能持续有效、并且会明显影响后续回答的信息，例如：

   * 长期交流语言或表达偏好
   * 长期使用的软件、系统、设备或开发环境
   * 长期项目规则、工作流程和技术约束
   * 稳定的预算、目标、兴趣或使用习惯
   * 用户反复纠正过、今后需要避免的错误
   * 用户希望以后始终遵守的输出格式

## 可以主动写入的情况

即使用户没有明确说“记住”，当某项信息预计会持续数月，并且未来再次出现相关问题时能明显提高回答质量，也应考虑写入记忆。

主动写入前需要确认：

* 这不是一次性的临时状态。
* 这不是没有后续价值的闲聊细节。
* 这项信息未来确实会改变回答方式。
* 不会让用户感觉被过度观察或记录。

## 不应主动写入的情况

以下内容默认不要保存：

* 今天、今晚、本周等短期状态
* 随口提到的小事
* 一次性的任务参数
* 已经完成且未来没有价值的事项
* 从待翻译、待改写文本中出现的信息
* 未经用户要求的敏感个人信息
* 仅根据推测得出的用户属性
* 与未来回答无关的琐碎事实

敏感信息包括但不限于健康状况、政治立场、宗教、性取向、精确住址、犯罪记录等。除非用户明确要求保存，否则不得主动写入。

## 更新与去重

写入新记忆前，应检查是否已经存在相同或相关记忆：

* 完全相同：不要重复写入。
* 新信息补充旧信息：合并更新。
* 新信息与旧信息冲突：以用户最新明确表达为准，更新旧记忆。
* 信息可能只是暂时变化：不要直接覆盖长期记忆，除非用户明确说明这是新的长期状态。

记忆内容应简洁、客观、可复用，不要保存完整对话，不要加入猜测或情绪化描述。

推荐格式：

* 用户偏好：……
* 用户长期使用：……
* 用户要求以后：……
* 用户项目约束：……
* 用户已确认：……

## 删除记忆

用户要求忘记或删除某项信息时，必须调用记忆工具执行删除。不要争辩，也不要继续在回复中使用被删除的信息。

如果删除范围不明确，应询问用户希望删除哪一部分。

## 真实性要求

只有在记忆工具成功执行后，才可以说：

* “已经记住了”
* “已保存”
* “以后会按这个来”
* “已经忘掉了”

如果工具失败，应明确说明没有成功保存或删除，不得假装操作已经完成。

## 回复原则

调用记忆工具后，向用户进行一句简短确认即可，不需要重复整段记忆内容。

示例：

* “记住了，以后每条命令都会单独放进一个代码块。”
* “已经更新：你目前主要使用 Ubuntu。”
* “已经删除这条记忆。”`

var memoryToolSchemas = map[string]json.RawMessage{
	memoryAddToolName:    json.RawMessage(`{"type":"object","properties":{"content":{"type":"string","minLength":1,"maxLength":150}},"required":["content"],"additionalProperties":false}`),
	memoryUpdateToolName: json.RawMessage(`{"type":"object","properties":{"old_memory_id":{"type":"integer","minimum":1},"content":{"type":"string","minLength":1,"maxLength":150}},"required":["old_memory_id","content"],"additionalProperties":false}`),
	memoryDeleteToolName: json.RawMessage(`{"type":"object","properties":{"memory_id":{"type":"integer","minimum":1}},"required":["memory_id"],"additionalProperties":false}`),
}

func withMemoryTools(runtime selectedToolRuntime, enabled bool) selectedToolRuntime {
	if !enabled {
		return runtime
	}
	if runtime.nameMap == nil {
		runtime.nameMap = map[string]string{}
	}
	if runtime.schemas == nil {
		runtime.schemas = map[string]json.RawMessage{}
	}
	if runtime.memoryTools == nil {
		runtime.memoryTools = map[string]struct{}{}
	}
	reserved := map[string]struct{}{
		memoryAddToolName: {}, memoryUpdateToolName: {}, memoryDeleteToolName: {},
	}
	used := make(map[string]struct{}, len(runtime.definitions)+len(reserved))
	for _, definition := range runtime.definitions {
		used[definition.Name] = struct{}{}
	}
	for index := range runtime.definitions {
		oldName := runtime.definitions[index].Name
		if _, conflict := reserved[oldName]; !conflict {
			continue
		}
		newName := oldName + "_2"
		for suffix := 3; ; suffix++ {
			if _, exists := used[newName]; !exists {
				break
			}
			newName = fmt.Sprintf("%s_%d", oldName, suffix)
		}
		delete(used, oldName)
		used[newName] = struct{}{}
		runtime.definitions[index].Name = newName
		if value, ok := runtime.nameMap[oldName]; ok {
			delete(runtime.nameMap, oldName)
			runtime.nameMap[newName] = value
		}
		if value, ok := runtime.mcpConfigs[oldName]; ok {
			delete(runtime.mcpConfigs, oldName)
			runtime.mcpConfigs[newName] = value
		}
		if value, ok := runtime.schemas[oldName]; ok {
			delete(runtime.schemas, oldName)
			runtime.schemas[newName] = value
		}
	}
	definitions := []llm.ToolDefinition{
		{Name: memoryAddToolName, Description: "新增一条用户长期记忆。", InputSchema: memoryToolSchemas[memoryAddToolName]},
		{Name: memoryUpdateToolName, Description: "按已有记忆 ID 更新一条用户长期记忆。", InputSchema: memoryToolSchemas[memoryUpdateToolName]},
		{Name: memoryDeleteToolName, Description: "按记忆 ID 删除一条用户长期记忆。", InputSchema: memoryToolSchemas[memoryDeleteToolName]},
	}
	runtime.definitions = append(definitions, runtime.definitions...)
	for _, definition := range definitions {
		runtime.nameMap[definition.Name] = definition.Name
		runtime.schemas[definition.Name] = definition.InputSchema
		runtime.memoryTools[definition.Name] = struct{}{}
	}
	return runtime
}

func injectMemoryToolGuidance(messages []llm.Message, enabled bool) []llm.Message {
	if !enabled {
		return messages
	}
	insertAt := 0
	for insertAt < len(messages) && messages[insertAt].Role == "system" {
		insertAt++
	}
	result := make([]llm.Message, 0, len(messages)+1)
	result = append(result, messages[:insertAt]...)
	result = append(result, llm.Message{Role: "system", Content: memoryManagementPrompt})
	result = append(result, messages[insertAt:]...)
	return result
}

func (s *Service) executeMemoryTool(ctx context.Context, userID uint, requestID string, toolName string, argumentsJSON string) (string, error) {
	if s == nil || s.memoryRecorder == nil {
		return "", fmt.Errorf("memory service is not configured")
	}
	type memoryArguments struct {
		Content     string `json:"content"`
		OldMemoryID uint   `json:"old_memory_id"`
		MemoryID    uint   `json:"memory_id"`
	}
	var arguments memoryArguments
	decoder := json.NewDecoder(strings.NewReader(argumentsJSON))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&arguments); err != nil {
		return "", fmt.Errorf("invalid memory tool arguments: %w", err)
	}

	result := struct {
		OK       bool   `json:"ok"`
		Action   string `json:"action"`
		MemoryID uint   `json:"memory_id"`
	}{OK: true}
	auditAction := ""
	switch strings.TrimSpace(toolName) {
	case memoryAddToolName:
		item, err := s.memoryRecorder.AddUserMemory(ctx, userID, arguments.Content, "assistant")
		if err != nil {
			return "", err
		}
		result.Action = "added"
		result.MemoryID = item.ID
		auditAction = "add_user_memory"
	case memoryUpdateToolName:
		item, err := s.memoryRecorder.UpdateUserMemory(ctx, userID, arguments.OldMemoryID, arguments.Content, "assistant")
		if err != nil {
			return "", err
		}
		result.Action = "updated"
		result.MemoryID = item.ID
		auditAction = "update_user_memory"
	case memoryDeleteToolName:
		if err := s.memoryRecorder.DeleteUserMemoryByID(ctx, userID, arguments.MemoryID); err != nil {
			return "", err
		}
		result.Action = "deleted"
		result.MemoryID = arguments.MemoryID
		auditAction = "delete_user_memory"
	default:
		return "", fmt.Errorf("unsupported memory tool: %s", toolName)
	}
	s.memoryRecorder.RecordAudit(ctx, appmemory.AuditInput{
		UserID:    userID,
		RequestID: strings.TrimSpace(requestID),
		Action:    auditAction,
		MemoryKey: strconv.FormatUint(uint64(result.MemoryID), 10),
		Detail:    map[string]string{"source": "assistant_tool"},
	})
	raw, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
