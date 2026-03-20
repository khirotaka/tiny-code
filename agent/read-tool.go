package agent

import (
	"os"

	"github.com/anthropics/anthropic-sdk-go"
)

const toolReadFile toolType = "read_file"

var readToolParam = anthropic.ToolParam{
	Name:        toolReadFile.String(),
	Description: anthropic.String("Read the contents of a file. Path is relative to the sandbox directory"),
	InputSchema: anthropic.ToolInputSchemaParam{
		Properties: map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "File path relative to sandbox (e.g. main.go)",
			},
		},
		Required: []string{"path"},
	},
}

func readFile(input map[string]any) toolResult {
	path, ok := input["path"].(string)
	if !ok {
		return toolResult{
			"path is required",
			true,
		}
	}
	safe, err := safePath(path)
	if err != nil {
		return toolResult{
			content: err.Error(),
			isError: true,
		}
	}
	data, err := os.ReadFile(safe)
	if err != nil {
		return toolResult{
			content: err.Error(),
			isError: true,
		}
	}
	return toolResult{
		content: string(data),
		isError: false,
	}
}
