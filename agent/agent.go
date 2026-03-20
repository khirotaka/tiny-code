package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
)

const (
	maxTurns     = 20
	maxTokens    = 4096
	model        = anthropic.ModelClaudeHaiku4_5
	systemPrompt = `あなたはコーディングエージェントです。ファイルの読み込み、書き込み、そして bashコマンドの実行ができます。
コードを修正する前に、必ずツールを使用して既存のコードを検査してください。
最終的な回答は簡潔にしてください。`
)

type Agent struct {
	client   *anthropic.Client
	messages []anthropic.MessageParam
}

func New() *Agent {
	// 自動的に 環境変数 ANTHROPIC_API_KEY が参照される
	client := anthropic.NewClient()
	return &Agent{
		client: &client,
	}
}

// ユーザーのリクエストを受け取りエージェントループを実行する
func (a *Agent) Run(ctx context.Context, userInput string) error {

	// sandboxディレクトリを確保
	if err := os.MkdirAll(sandboxDir, 0755); err != nil {
		return fmt.Errorf("failed to create sandbox dir: %w", err)
	}

	// ユーザーメッセージを履歴に追加
	a.messages = append(a.messages, anthropic.NewUserMessage(
		anthropic.NewTextBlock(userInput),
	))

	fmt.Println()

	for range maxTurns {
		resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     model,
			MaxTokens: maxTokens,
			System: []anthropic.TextBlockParam{
				{
					Text: systemPrompt,
				},
			},
			Messages: a.messages,
			Tools:    getToolDefinitions(),
		})
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

		// LLMの回答を履歴に追加
		a.messages = append(a.messages, anthropic.NewAssistantMessage(resp.ToParam().Content...))

		switch resp.StopReason {
		case anthropic.StopReasonEndTurn:
			// テキスト応答を出力して終了
			for _, block := range resp.Content {
				fmt.Println(block.Text)
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
