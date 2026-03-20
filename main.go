package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/joho/godotenv"
	"github.com/khirotaka/tiny-code/agent"
)

// AGENTS.md を読み込む
// 1. $XDG_CONFIG_HOME/tiny-code/AGENTS.md
// 2. カレントディレクトリの AGENTS.md
// 3. カレントディレクトリの AGENTS.local.md
func loadAgentsFile() string {
	var promptBuilder strings.Builder

	// 1. Global $XDG_CONFIG_HOME/tiny-code/AGENTS.md
	configDir, ok := os.LookupEnv("XDG_CONFIG_HOME")
	if !ok || configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			configDir = filepath.Join(homeDir, ".config")
		}
	}
	if configDir != "" {
		configPath := filepath.Join(configDir, "tiny-code", "AGENTS.md")
		if _, err := os.Stat(configPath); err == nil {
			data, err := os.ReadFile(configPath)
			if err == nil && len(data) > 0 {
				promptBuilder.WriteString("<global_agent_rule>\n")
				promptBuilder.Write(data)
				promptBuilder.WriteString("\n</global_agent_rule>\n")
			}
		}
	}
	// 2. プロジェクトスコープの AGENTS.md
	projectPath := "AGENTS.md"
	if _, err := os.Stat(projectPath); err == nil {
		data, err := os.ReadFile(projectPath)
		if err == nil && len(data) > 0 {
			promptBuilder.WriteString("<project_agent_rule>\n")
			promptBuilder.Write(data)
			promptBuilder.WriteString("\n</project_agent_rule>\n")
		}
	}
	// 3. プロジェクトスコープの AGENTS.local.md
	localPath := "AGENTS.local.md"
	if _, err := os.Stat(localPath); err == nil {
		data, err := os.ReadFile(localPath)
		if err == nil && len(data) > 0 {
			promptBuilder.WriteString("<local_agent_rule>\n")
			promptBuilder.Write(data)
			promptBuilder.WriteString("\n</local_agent_rule>\n")
		}
	}

	return promptBuilder.String()
}

// カレントディレクトリの .tiny-code/skills/ ディレクトリにある全ての SKILL.md の frontmatter を収集する
func loadSkills() ([]agent.Meta, error) {
	var skills []agent.Meta
	err := filepath.Walk(agent.SkillPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		var skill agent.Meta
		_, err = frontmatter.Parse(bytes.NewReader(data), &skill)
		if err != nil {
			return err
		}

		skills = append(skills, skill)
		return nil
	})

	return skills, err
}

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error loading .env file: %v\n", err)
		os.Exit(1)
	}

	rules := loadAgentsFile()
	skills, err := loadSkills()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}
	a := agent.New(rules, skills)
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("🤖 tiny-code agent (type 'exit' to quit)")

	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "exit" || input == "quit" {
			fmt.Println("Bye!")
			break
		}

		var (
			skill       string
			userMessage string
		)

		if after, ok := strings.CutPrefix(input, "/"); ok {
			skillName, args, _ := strings.Cut(after, " ")
			// skills/{skillName}/SKILL.md を読み込んでエージェントに渡す
			skillData, err := os.ReadFile(filepath.Join(agent.SkillPath, skillName, "SKILL.md"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
				continue
			}

			var m agent.Meta
			body, err := frontmatter.Parse(bytes.NewReader(skillData), &m)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
				continue
			}

			skill = string(body)
			userMessage = args
			if userMessage == "" {
				userMessage = "スキルの手順に従って実行してください。"
			}
		} else {
			userMessage = input
		}

		if err := a.Run(context.Background(), userMessage, skill); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		}
	}
}
