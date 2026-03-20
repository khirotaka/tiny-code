package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

var sandboxDir string

const (
	toolReadFile  string = "read_file"
	toolWriteFile string = "write_file"
	toolExecBash  string = "exec_bash"
)

func init() {
	abs, err := filepath.Abs("./sandbox")
	if err != nil {
		panic(err)
	}
	sandboxDir = abs
}

func getToolDefinitions() []anthropic.ToolUnionParam {
	var toolParamsDefinitions = []anthropic.ToolParam{
		{
			Name:        toolReadFile,
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
		},
		{
			Name:        toolWriteFile,
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
		},
		{
			Name:        toolExecBash,
			Description: anthropic.String("Execute a bash command inside the sandbox directory"),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{
					"command": map[string]any{
						"type":        "string",
						"description": "Bash command to execute",
					},
				},
				Required: []string{"command"},
			},
		},
	}

	var tools = make([]anthropic.ToolUnionParam, len(toolParamsDefinitions))
	for i, toolParam := range toolParamsDefinitions {
		tools[i] = anthropic.ToolUnionParam{
			OfTool: &toolParam,
		}
	}

	return tools
}

type toolResult struct {
	content string
	isError bool
}

func executeTool(name string, input map[string]any) toolResult {
	switch name {
	case toolReadFile:
		return readFile(input)
	case toolWriteFile:
		return writeFile(input)
	case toolExecBash:
		return execBash(input)
	default:
		return toolResult{
			fmt.Sprintf("unknown tool: %s", name),
			true,
		}
	}
}

func safePath(raw string) (string, error) {
	cleaned := filepath.Join(sandboxDir, filepath.Clean("/"+raw))
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(abs, sandboxDir) {
		return "", fmt.Errorf("path travasal not allowed: %s", raw)
	}
	return abs, nil
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

func execBash(input map[string]any) toolResult {
	command, ok := input["command"].(string)
	if !ok {
		return toolResult{
			"command is required",
			true,
		}
	}
	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = sandboxDir
	out, err := cmd.CombinedOutput()
	result := string(out)
	if err != nil {
		return toolResult{
			result + "\n" + err.Error(),
			true,
		}
	}
	return toolResult{
		result,
		false,
	}
}
