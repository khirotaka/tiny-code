package agent

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

var sandboxDir string

func parseTool(name string) (toolType, error) {
	switch toolType(name) {
	case toolReadFile, toolWriteFile, toolExecBash, toolLoadSkill:
		return toolType(name), nil
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func init() {
	abs, err := filepath.Abs("./sandbox")
	if err != nil {
		panic(err)
	}
	sandboxDir = abs
}

func getToolDefinitions(allowTools []toolType) []anthropic.ToolUnionParam {
	var toolParamsDefinitions = []anthropic.ToolParam{}

	for _, tool := range allowTools {
		switch tool {
		case toolReadFile:
			toolParamsDefinitions = append(toolParamsDefinitions, readToolParam)
		case toolWriteFile:
			toolParamsDefinitions = append(toolParamsDefinitions, writeToolParam)
		case toolExecBash:
			toolParamsDefinitions = append(toolParamsDefinitions, execBashToolParam)
		case toolLoadSkill:
			toolParamsDefinitions = append(toolParamsDefinitions, loadSkillToolParam)
		}
	}

	var tools = make([]anthropic.ToolUnionParam, len(toolParamsDefinitions))
	for i, toolParam := range toolParamsDefinitions {
		tools[i] = anthropic.ToolUnionParam{
			OfTool: &toolParam,
		}
	}

	return tools
}

func executeTool(name toolType, input map[string]any) toolResult {
	switch name {
	case toolReadFile:
		return readFile(input)
	case toolWriteFile:
		return writeFile(input)
	case toolExecBash:
		return execBash(input)
	case toolLoadSkill:
		return loadSkill(input)
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
		return "", fmt.Errorf("path traversal not allowed: %s", raw)
	}
	return abs, nil
}
