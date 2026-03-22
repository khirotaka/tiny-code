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
	maxTurns         = 20
	maxTokens        = 4096
	model            = anthropic.ModelClaudeHaiku4_5
	baseSystemPrompt = `あなたはコーディングエージェントです。ファイルの読み込み、書き込み、そして bashコマンドの実行ができます。
コードを修正する前に、必ずツールを使用して既存のコードを検査してください。
最終的な回答は簡潔にしてください。`
	contextThreshold = 8_000
)

type Agent struct {
	client       *Client
	systemPrompt string
	messages     []anthropic.MessageParam
	eventCh      chan<- StreamEvent
	tools        []toolType
	isRoot       bool
}

func buildSystemPrompt(rules string, skills []SkillMeta, agents []AgentMeta) string {
	var sb strings.Builder
	sb.WriteString(baseSystemPrompt)

	if len(skills) > 0 {
		sb.WriteString("\n<available_skills>\n")
		sb.WriteString("ユーザーのリクエストに応じて、以下のスキルを `/skill-name` 形式で提案できます。\n")
		for _, s := range skills {
			fmt.Fprintf(&sb, "- /%s: %s\n", s.Name, s.Description)
		}
		sb.WriteString("\n</available_skills>\n")
	}

	if len(agents) > 0 {
		sb.WriteString("\n<available_agents>\n")
		sb.WriteString("サブタスクを委譲するために、以下のエージェントを run_agent ツールで呼び出せます。\n")
		for _, a := range agents {
			fmt.Fprintf(&sb, "- %s: %s\n", a.Name, a.Description)
		}
		sb.WriteString("\n</available_agents>\n")
	}

	if rules != "" {
		sb.WriteString("\n<rules>\n")
		sb.WriteString(rules)
		sb.WriteString("\n</rules>\n")
	}

	return sb.String()
}

func New(client *Client, eventCh chan StreamEvent, rules string, skills []SkillMeta, agents []AgentMeta) *Agent {
	return &Agent{
		client:       client,
		systemPrompt: buildSystemPrompt(rules, skills, agents),
		eventCh:      eventCh,
		tools:        []toolType{toolReadFile, toolWriteFile, toolExecBash, toolLoadSkill, toolRunAgent},
		isRoot:       true,
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
		resp, err := a.client.call(ctx, anthropic.MessageNewParams{
			Model:     model,
			MaxTokens: maxTokens,
			System:    systemParams,
			Messages:  a.messages,
			Tools:     getToolDefinitions(a.tools),
		})
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

		// LLMの回答を履歴に追加
		a.messages = append(a.messages, resp.ToParam())
		// トークン使用量チェック
		if resp.Usage.InputTokens > contextThreshold {
			_ = a.compact(ctx)
		}

		switch resp.StopReason {
		case anthropic.StopReasonEndTurn:
			// テキスト応答を出力して終了
			for _, block := range resp.Content {
				if block.Type == "text" {
					a.eventCh <- StreamEvent{Type: EventText, Text: block.Text}
				}
			}
			if a.isRoot {
				a.eventCh <- StreamEvent{Type: EventDone}
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

					a.eventCh <- StreamEvent{Type: EventToolUse, Tool: block.Name, Input: input}

					// ツール実行
					tool, err := parseTool(block.Name)
					if err != nil {
						return err
					}
					result := a.executeTool(ctx, tool, input)
					a.eventCh <- StreamEvent{
						Type:    EventToolResult,
						Content: result.content,
						IsError: result.isError,
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
	compactResp, err := a.client.call(ctx, anthropic.MessageNewParams{
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

	if len(compactResp.Content) == 0 {
		return fmt.Errorf("compact response has no content")
	}
	summary := compactResp.Content[0].Text
	a.messages = []anthropic.MessageParam{
		anthropic.NewUserMessage(
			anthropic.NewTextBlock("[会話履歴の要約]\n" + summary),
		),
	}

	return nil
}
