package session

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ToolCall is a model's request to invoke a named tool with arguments.
type ToolCall struct {
	Name string
	Args map[string]any
}

// ToolResult is the outcome of executing a ToolCall, fed back to the model.
type ToolResult struct {
	Name     string
	Response map[string]any
}

// Message is one turn in a conversation. Exactly one shape is populated: plain
// text (user/assistant), an assistant ToolCall, or a tool result (Role ==
// RoleTool with ToolResult set). These are vendor-neutral domain types — no
// provider SDK types appear here.
type Message struct {
	Role       Role
	Content    string
	ToolCall   *ToolCall
	ToolResult *ToolResult
}

func UserMessage(content string) Message {
	return Message{Role: RoleUser, Content: content}
}

func AssistantMessage(content string) Message {
	return Message{Role: RoleAssistant, Content: content}
}

func AssistantToolCall(call *ToolCall) Message {
	return Message{Role: RoleAssistant, ToolCall: call}
}

func ToolResultMessage(name string, response map[string]any) Message {
	return Message{Role: RoleTool, ToolResult: &ToolResult{Name: name, Response: response}}
}
