package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/jinto/ina/daemon"
	"github.com/jinto/ina/store"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	s := server.NewMCPServer("ina", "1.0.0",
		server.WithToolCapabilities(true),
	)

	addReportProgress(s)
	addMarkBlocked(s)
	addCheckAgents(s)
	addLearn(s)
	addRecall(s)
	addLogEvent(s)

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("mcp server error: %v", err)
	}
}

type progressReport struct {
	InProgress string `json:"in_progress"`
	Completed  string `json:"completed"`
	Remaining  string `json:"remaining"`
	Context    string `json:"context"`
}

func addReportProgress(s *server.MCPServer) {
	tool := mcp.NewTool("ina_report_progress",
		mcp.WithDescription("Report task progress to the ina watchdog daemon. Call this periodically to keep the daemon informed of your work."),
		mcp.WithString("completed", mcp.Description("Comma-separated list of completed items")),
		mcp.WithString("in_progress", mcp.Description("What you're currently working on"), mcp.Required()),
		mcp.WithString("remaining", mcp.Description("Comma-separated list of remaining items")),
		mcp.WithString("context", mcp.Description("Context for another agent to continue if you crash")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		inProgress, _ := args["in_progress"].(string)
		completed, _ := args["completed"].(string)
		remaining, _ := args["remaining"].(string)
		ctxStr, _ := args["context"].(string)

		return callDaemon(daemon.ActionProgress, progressReport{
			InProgress: inProgress,
			Completed:  completed,
			Remaining:  remaining,
			Context:    ctxStr,
		})
	})
}

func addMarkBlocked(s *server.MCPServer) {
	tool := mcp.NewTool("ina_mark_blocked",
		mcp.WithDescription("Tell the ina daemon that you are blocked and need human intervention."),
		mcp.WithString("reason", mcp.Description("Why you are blocked"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		reason, _ := req.GetArguments()["reason"].(string)

		return callDaemon(daemon.ActionBlocked, struct {
			Reason string `json:"reason"`
		}{Reason: reason})
	})
}

func addCheckAgents(s *server.MCPServer) {
	tool := mcp.NewTool("ina_check_agents",
		mcp.WithDescription("Check the status of all agents tracked by the ina watchdog daemon."),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resp, err := sendToDaemon(daemon.ActionStatus, nil)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("daemon error: %v", err)), nil
		}

		var data any
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Tracked agents:\n%s", resp.Data)), nil
		}
		indented, _ := json.MarshalIndent(data, "", "  ")
		return mcp.NewToolResultText(fmt.Sprintf("Tracked agents:\n%s", indented)), nil
	})
}

func addLearn(s *server.MCPServer) {
	tool := mcp.NewTool("ina_learn",
		mcp.WithDescription("Record a learning (pattern, pitfall, or architectural insight) discovered during work. Persists across sessions so future reviews and investigations can recall it."),
		mcp.WithString("type", mcp.Description("Category: pattern, pitfall, preference, or architecture"), mcp.Required(),
			mcp.Enum("pattern", "pitfall", "preference", "architecture")),
		mcp.WithString("key", mcp.Description("Short kebab-case identifier, e.g. 'test-isolation' or 'nil-channel-send'"), mcp.Required()),
		mcp.WithString("insight", mcp.Description("One-sentence description of the learning"), mcp.Required()),
		mcp.WithNumber("confidence", mcp.Description("Confidence level 1-10 (default 7)")),
		mcp.WithString("source", mcp.Description("Which skill discovered this, e.g. 'review', 'investigate', 'build'")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		dir, err := store.ProjectDir()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("project dir: %v", err)), nil
		}

		confidence := 7
		if c, ok := args["confidence"].(float64); ok && c >= 1 && c <= 10 {
			confidence = int(c)
		}
		source, _ := args["source"].(string)

		l := store.Learning{
			Type:       args["type"].(string),
			Key:        args["key"].(string),
			Insight:    args["insight"].(string),
			Confidence: confidence,
			Source:     source,
		}

		if err := store.SaveLearning(dir, l); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("save: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Learned: [%s] %s — %s (confidence %d/10)", l.Type, l.Key, l.Insight, l.Confidence)), nil
	})
}

func addRecall(s *server.MCPServer) {
	tool := mcp.NewTool("ina_recall",
		mcp.WithDescription("Search past learnings for this project. Returns patterns, pitfalls, and insights recorded in previous sessions."),
		mcp.WithString("query", mcp.Description("Search term to filter by key, insight, or type (empty returns recent)")),
		mcp.WithString("type", mcp.Description("Filter by category: pattern, pitfall, preference, or architecture")),
		mcp.WithNumber("limit", mcp.Description("Max results to return (default 20)")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		dir, err := store.ProjectDir()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("project dir: %v", err)), nil
		}

		query, _ := args["query"].(string)
		typeFilter, _ := args["type"].(string)
		limit := 20
		if l, ok := args["limit"].(float64); ok && l > 0 {
			limit = int(l)
		}

		results, err := store.SearchLearnings(dir, query, typeFilter, limit)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("search: %v", err)), nil
		}
		if len(results) == 0 {
			return mcp.NewToolResultText("No learnings found for this project."), nil
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "Found %d learnings:\n\n", len(results))
		for _, l := range results {
			fmt.Fprintf(&sb, "- [%s] **%s**: %s (confidence %d/10", l.Type, l.Key, l.Insight, l.Confidence)
			if l.Source != "" {
				fmt.Fprintf(&sb, ", from %s", l.Source)
			}
			sb.WriteString(")\n")
		}
		return mcp.NewToolResultText(sb.String()), nil
	})
}

func addLogEvent(s *server.MCPServer) {
	tool := mcp.NewTool("ina_log_event",
		mcp.WithDescription("Record a structured event (review result, build outcome, test result) for this project. Used by /ina:ship for pre-flight dashboards."),
		mcp.WithString("skill", mcp.Description("Which skill produced this event, e.g. 'review', 'build', 'test'"), mcp.Required()),
		mcp.WithString("status", mcp.Description("Outcome: clean, issues_found, pass, fail"), mcp.Required()),
		mcp.WithString("summary", mcp.Description("One-line summary of the result")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		dir, err := store.ProjectDir()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("project dir: %v", err)), nil
		}

		summary, _ := args["summary"].(string)

		e := store.Event{
			Skill:   args["skill"].(string),
			Status:  args["status"].(string),
			Summary: summary,
		}

		if err := store.LogEvent(dir, e); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("log: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Event logged: [%s] %s — %s", e.Skill, e.Status, e.Commit)), nil
	})
}

func callDaemon(action string, payload any) (*mcp.CallToolResult, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshal error: %v", err)), nil
	}
	resp, err := daemon.SendCommand(daemon.Command{Action: action, Data: data})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("daemon error: %v", err)), nil
	}
	return mcp.NewToolResultText(resp.Message), nil
}

func sendToDaemon(action string, data json.RawMessage) (*daemon.Response, error) {
	return daemon.SendCommand(daemon.Command{Action: action, Data: data})
}
