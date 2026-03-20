package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

const (
	maxTurns     = 20
	maxTokens    = 4096
	model        = anthropic.ModelClaudeHaiku4_5
	systemPrompt = `あなたはコーディングエージェントです。ファイルの読み込み、書き込み、そして bashコマンドの実行ができます。
コードを修正する前に、必ずツールを使用して既存のコードを検査してください。
最終的な回答は簡潔にしてください。`
	contextThreshold = 8_000
)

type Agent struct {
	client       *anthropic.Client
	systemPrompt string
	messages     []anthropic.MessageParam
}

func buildSystemPrompt(rules string, skills []SkillMeta) string {
	var sb strings.Builder
	sb.WriteString(systemPrompt)

	if len(skills) > 0 {
		sb.WriteString("\n<available_skills>\n")
		sb.WriteString("ユーザーのリクエストに応じて、以下のスキルを `/skill-name` 形式で提案できます。\n")
		for _, s := range skills {
			fmt.Fprintf(&sb, "- /%s: %s\n", s.Name, s.Description)
		}
		sb.WriteString("\n</available_skills>\n")
	}

	if rules != "" {
		sb.WriteString("\n<rules>\n")
		sb.WriteString(rules)
		sb.WriteString("\n</rules>\n")
	}

	return sb.String()
}

func New(rules string, skills []SkillMeta) *Agent {
	// 自動的に 環境変数 ANTHROPIC_API_KEY が参照される
	client := anthropic.NewClient()
	return &Agent{
		client:       &client,
		systemPrompt: buildSystemPrompt(rules, skills),
	}
}

// ユーザーのリクエストを受け取りエージェントループを実行する
func (a *Agent) Run(ctx context.Context, userInput, skillData string) error {

	// sandboxディレクトリを確保
	if err := os.MkdirAll(sandboxDir, 0755); err != nil {
		return fmt.Errorf("failed to create sandbox dir: %w", err)
	}

	// ユーザーメッセージを履歴に追加
	a.messages = append(a.messages, anthropic.NewUserMessage(
		anthropic.NewTextBlock(userInput),
	))

	fmt.Println()

	systemParams := []anthropic.TextBlockParam{
		{
			Text: a.systemPrompt,
		},
	}

	if skillData != "" {
		systemParams = append(systemParams, anthropic.TextBlockParam{
			Text: skillData,
		})
	}

	for range maxTurns {
		resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     model,
			MaxTokens: maxTokens,
			System:    systemParams,
			Messages:  a.messages,
			Tools:     getToolDefinitions(),
		})
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

		// LLMの回答を履歴に追加
		a.messages = append(a.messages, resp.ToParam())
		// トークン使用量チェック
		if resp.Usage.InputTokens > contextThreshold {
			if err := a.compact(ctx); err != nil {
				return fmt.Errorf("failed to compact message history: %w", err)
			}
		}

		switch resp.StopReason {
		case anthropic.StopReasonEndTurn:
			// テキスト応答を出力して終了
			for _, block := range resp.Content {
				if block.Type == "text" {
					fmt.Println(block.Text)
				}
			}
			return nil

		case anthropic.StopReasonToolUse:
			// tool_use ブロックを処理
			var toolResults []anthropic.ContentBlockParamUnion

			for _, block := range resp.Content {
				switch variant := block.AsAny().(type) {
				case anthropic.ToolUseBlock:
					var input map[string]any

					if err := json.Unmarshal([]byte(variant.JSON.Input.Raw()), &input); err != nil {
						return fmt.Errorf("failed to unmarshal tool input: %w", err)
					}

					fmt.Printf("🔧 %s(%v)\n", block.Name, formatInput(input))

					// ツール実行
					result := executeTool(block.Name, input)
					if result.isError {
						fmt.Printf("  ❌ %s\n", result.content)
					} else {
						preview := result.content
						if len(preview) > 100 {
							preview = preview[:100] + "..."
						}
						fmt.Printf("  ✅ %s\n", preview)
					}

					toolResults = append(toolResults, anthropic.NewToolResultBlock(
						block.ID,
						result.content,
						result.isError,
					))
				}

			}

			// tool_result を user メッセージとして履歴に追加
			a.messages = append(a.messages, anthropic.NewUserMessage(toolResults...))
		default:
			return fmt.Errorf("unexpected stop reason: %s", resp.StopReason)
		}
	}

	return fmt.Errorf("reached max turns (%d)", maxTurns)
}

func (a *Agent) compact(ctx context.Context) error {
	compactResp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     model,
		MaxTokens: maxTokens,
		System: []anthropic.TextBlockParam{
			{
				Text: "以下の会話履歴を、エージェントが作業を継続するために必要な情報を保持した形で簡潔に要約してください。",
			},
		},
		Messages: a.messages,
	})
	if err != nil {
		return fmt.Errorf("failed to compact message history: %w", err)
	}

	summary := compactResp.Content[0].Text
	a.messages = []anthropic.MessageParam{
		anthropic.NewUserMessage(
			anthropic.NewTextBlock("[会話履歴の要約]\n" + summary),
		),
	}

	return nil
}

// ツール引数を見やすく整形する
func formatInput(input map[string]any) string {
	if path, ok := input["path"].(string); ok {
		return fmt.Sprintf("path=%q", path)
	}
	if command, ok := input["command"].(string); ok {
		if len(command) > 60 {
			command = command[:60] + "..."
		}
		return fmt.Sprintf("command=%q", command)
	}
	return fmt.Sprintf("%v", input)
}
