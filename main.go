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

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error loading .env file: %v\n", err)
		os.Exit(1)
	}

	// カレントディレクトリの skills/ ディレクトリにある全ての SKILL.md の frontmatter を収集する
	var skills []agent.Meta
	if err := filepath.Walk("skills", func(path string, info os.FileInfo, err error) error {
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
	}); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}

	a := agent.New(skills)
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
			skillData, err := os.ReadFile(filepath.Join("skills", skillName, "SKILL.md"))
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
