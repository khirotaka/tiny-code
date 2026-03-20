package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/khirotaka/tiny-code/agent"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error loading .env file: %v\n", err)
		os.Exit(1)
	}

	a := agent.New()
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

			skill = string(skillData)
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
