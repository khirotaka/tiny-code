package agent

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/anthropics/anthropic-sdk-go"
)

const toolWriteFile toolType = "write_file"

var writeToolParam = anthropic.ToolParam{
	Name:        toolWriteFile.String(),
	Description: anthropic.String("Write content to a file. Create directories as needed. Path is relative to the sandbox directory."),
	InputSchema: anthropic.ToolInputSchemaParam{
		Properties: map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "File path relative to sandbox (e.g. main.go)",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Content to write to the file",
			},
		},
		Required: []string{"path", "content"},
	},
}

func writeFile(input map[string]any) toolResult {
	path, ok := input["path"].(string)
	if !ok {
		return toolResult{
			"path is required",
			true,
		}
	}
	content, ok := input["content"].(string)
	if !ok {
		return toolResult{
			"content is required",
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
	if err := os.MkdirAll(filepath.Dir(safe), 0755); err != nil {
		return toolResult{
			content: err.Error(),
			isError: true,
		}
	}
	if err := os.WriteFile(safe, []byte(content), 0644); err != nil {
		return toolResult{
			content: err.Error(),
			isError: true,
		}
	}

	return toolResult{
		fmt.Sprintf("wrote %d bytes to %s", len(content), path),
		false,
	}
}
