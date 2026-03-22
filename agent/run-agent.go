package agent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/anthropics/anthropic-sdk-go"
)

const toolRunAgent toolType = "run_agent"

var runAgentToolParam = anthropic.ToolParam{
	Name:        toolRunAgent.String(),
	Description: anthropic.String("Invoke a named sub-agent to handle a sub-task. The sub-agent runs with its own tool set and message history but shares the same API client and output channel."),
	InputSchema: anthropic.ToolInputSchemaParam{
		Properties: map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Agent name (matches a file in .tiny-code/agents/)",
			},
			"task": map[string]any{
				"type":        "string",
				"description": "Task description to pass to the sub-agent",
			},
		},
		Required: []string{"name", "task"},
	},
}

func (a *Agent) runAgent(ctx context.Context, input map[string]any) toolResult {
	name, ok := input["name"].(string)
	if !ok {
		return toolResult{"name is required", true}
	}
	task, ok := input["task"].(string)
	if !ok {
		return toolResult{"task is required", true}
	}
	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		return toolResult{"invalid agent name", true}
	}

	data, err := os.ReadFile(filepath.Join(AgentPath, name+".md"))
	if err != nil {
		return toolResult{err.Error(), true}
	}

	var meta AgentMeta
	body, err := frontmatter.Parse(bytes.NewReader(data), &meta)
	if err != nil {
		return toolResult{err.Error(), true}
	}

	childTools := parseToolList(meta.Tools)

	child := &Agent{
		client:       a.client,
		systemPrompt: baseSystemPrompt + "\n" + string(body),
		eventCh:      a.eventCh,
		tools:        childTools,
		isRoot:       false,
	}

	if err := child.Run(ctx, task, ""); err != nil {
		return toolResult{fmt.Sprintf("sub-agent error: %v", err), true}
	}
	return toolResult{"sub-agent completed", false}
}

// parseToolList maps frontmatter tool names to toolType values.
func parseToolList(names []string) []toolType {
	m := map[string]toolType{
		"Read":      toolReadFile,
		"Write":     toolWriteFile,
		"Bash":      toolExecBash,
		"LoadSkill": toolLoadSkill,
		"RunAgent":  toolRunAgent,
	}
	var result []toolType
	for _, n := range names {
		if t, ok := m[n]; ok {
			result = append(result, t)
		}
	}
	if len(result) == 0 {
		return []toolType{toolReadFile}
	}
	return result
}
