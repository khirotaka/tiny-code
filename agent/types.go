package agent

const (
	SkillPath = ".tiny-code/skills"
	AgentPath = ".tiny-code/agents"
)

type SkillMeta struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// AgentMeta is parsed from .tiny-code/agents/{name}.md frontmatter
type AgentMeta struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description"`
	Tools       stringSlice `yaml:"tools"`
}

// stringSlice handles both scalar (`tools: Read`) and list (`tools: [Read, Write]`) YAML forms
type stringSlice []string

func (s *stringSlice) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var single string
	if err := unmarshal(&single); err == nil {
		*s = []string{single}
		return nil
	}
	var multi []string
	if err := unmarshal(&multi); err != nil {
		return err
	}
	*s = multi
	return nil
}

type EventType int

const (
	EventText EventType = iota
	EventToolUse
	EventToolResult
	EventError
	EventDone
)

type StreamEvent struct {
	Type    EventType
	Text    string         // EventText, EventError message
	Tool    string         // EventToolUse
	Input   map[string]any // EventToolUse
	Content string         // EventToolResult
	IsError bool           // EventToolResult
	Err     error          // EventError
}

type toolType string

func (t toolType) String() string {
	return string(t)
}

type toolResult struct {
	content string
	isError bool
}
