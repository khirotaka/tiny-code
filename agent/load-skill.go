package agent

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/anthropics/anthropic-sdk-go"
)

const toolLoadSkill toolType = "load_skill"

var loadSkillToolParam = anthropic.ToolParam{
	Name:        toolLoadSkill.String(),
	Description: anthropic.String("Specify the skill name to load the detailed instructions for that skill. The system prompt displays a list of available skills, so actively access any relevant skills."),
	InputSchema: anthropic.ToolInputSchemaParam{
		Properties: map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Skill name",
			},
		},
		Required: []string{"name"},
	},
}

func loadSkill(input map[string]any) toolResult {
	name, ok := input["name"].(string)
	if !ok {
		return toolResult{
			"name is required",
			true,
		}
	}

	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		return toolResult{
			"invalid skill name",
			true,
		}
	}
	data, err := os.ReadFile(filepath.Join(SkillPath, name, "SKILL.md"))
	if err != nil {
		return toolResult{
			err.Error(),
			true,
		}
	}
	var m SkillMeta
	body, err := frontmatter.Parse(bytes.NewReader(data), &m)
	if err != nil {
		return toolResult{
			err.Error(),
			true,
		}
	}
	return toolResult{
		string(body),
		false,
	}
}
