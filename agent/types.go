package agent

const SkillPath = ".tiny-code/skills"

type SkillMeta struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type toolType string

func (t toolType) String() string {
	return string(t)
}

type toolResult struct {
	content string
	isError bool
}
