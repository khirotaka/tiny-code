package agent

import (
	"os/exec"

	"github.com/anthropics/anthropic-sdk-go"
)

const toolExecBash toolType = "exec_bash"

var execBashToolParam = anthropic.ToolParam{
	Name:        toolExecBash.String(),
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
