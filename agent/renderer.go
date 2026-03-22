package agent

import "fmt"

// Renderer reads StreamEvents from eventCh and renders them to stdout.
// Blocks until the channel is closed.
func Renderer(eventCh <-chan StreamEvent) {
	for event := range eventCh {
		switch event.Type {
		case EventText:
			fmt.Println(event.Text)
		case EventToolUse:
			fmt.Printf("🔧 %s(%v)\n", event.Tool, formatInput(event.Input))
		case EventToolResult:
			if event.IsError {
				fmt.Printf("  ❌ %s\n", event.Content)
			} else {
				preview := event.Content
				if len(preview) > 100 {
					preview = preview[:100] + "..."
				}
				fmt.Printf("  ✅ %s\n", preview)
			}
		case EventError:
			fmt.Printf("❌ Error: %v\n", event.Err)
		case EventDone:
			fmt.Print("\n> ")
		}
	}
}

// formatInput formats tool input arguments for display.
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
