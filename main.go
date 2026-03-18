package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// withAuth wraps a tool handler with authentication and auto-retry on token expiry.
func withAuth(fn func(*http.Client, map[string]interface{}) (string, error)) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if err := ensureLoggedIn(httpClient, false); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := fn(httpClient, request.GetArguments())
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Retry on auth error
		if isAuthError(result) {
			log.Println("Token expired, re-authenticating...")
			if err := ensureLoggedIn(httpClient, true); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			result, err = fn(httpClient, request.GetArguments())
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
		}

		return mcp.NewToolResultText(result), nil
	}
}

func isAuthError(result string) bool {
	for code := range authErrorCodes {
		if strings.Contains(result, fmt.Sprintf("API Error %s:", code)) {
			return true
		}
	}
	return false
}

func main() {
	s := server.NewMCPServer("coros-mcp", "0.1.0")

	// get_user_info
	s.AddTool(
		mcp.NewTool("get_user_info",
			mcp.WithDescription("Get COROS user profile info"),
		),
		withAuth(func(c *http.Client, _ map[string]interface{}) (string, error) {
			return getUserInfo(c)
		}),
	)

	// get_workouts
	s.AddTool(
		mcp.NewTool("get_workouts",
			mcp.WithDescription("Get workout records for a date range. Returns list of activities with distance, pace, duration, calories, etc."),
			mcp.WithString("start_date", mcp.Description("Start date YYYYMMDD. Defaults to 30 days ago.")),
			mcp.WithString("end_date", mcp.Description("End date YYYYMMDD. Defaults to today.")),
			mcp.WithNumber("page", mcp.Description("Page number (default 1)")),
			mcp.WithNumber("size", mcp.Description("Results per page (default 20)")),
		),
		withAuth(func(c *http.Client, args map[string]interface{}) (string, error) {
			startDate, _ := args["start_date"].(string)
			endDate, _ := args["end_date"].(string)
			page := intArg(args, "page", 1)
			size := intArg(args, "size", 20)
			return getWorkouts(c, startDate, endDate, page, size)
		}),
	)

	// get_workout_detail
	s.AddTool(
		mcp.NewTool("get_workout_detail",
			mcp.WithDescription("Get detailed data for a specific workout by labelId"),
			mcp.WithString("label_id", mcp.Required(), mcp.Description("Workout labelId from get_workouts")),
			mcp.WithString("sport_type", mcp.Required(), mcp.Description("sportType code from get_workouts (e.g. 101 for run)")),
		),
		withAuth(func(c *http.Client, args map[string]interface{}) (string, error) {
			return getWorkoutDetail(c, args["label_id"].(string), stringArg(args, "sport_type", ""))
		}),
	)

	// get_workout_file
	s.AddTool(
		mcp.NewTool("get_workout_file",
			mcp.WithDescription("Get download URL for a workout file (.fit/.tcx/.gpx)"),
			mcp.WithString("label_id", mcp.Required(), mcp.Description("Workout labelId from get_workouts")),
			mcp.WithString("sport_type", mcp.Required(), mcp.Description("sportType code from get_workouts (e.g. 101 for run)")),
			mcp.WithString("file_type", mcp.Description("File format: fit, tcx, gpx, or kml (default: fit)")),
		),
		withAuth(func(c *http.Client, args map[string]interface{}) (string, error) {
			return getWorkoutFile(c, args["label_id"].(string), stringArg(args, "sport_type", ""), stringArg(args, "file_type", "fit"))
		}),
	)

	// get_recent_runs
	s.AddTool(
		mcp.NewTool("get_recent_runs",
			mcp.WithDescription("Get recent running workouts (outdoor + trail runs) for the last N days with formatted stats."),
			mcp.WithNumber("days", mcp.Description("Number of days to look back (max 90, default 14)")),
		),
		withAuth(func(c *http.Client, args map[string]interface{}) (string, error) {
			return getRecentRuns(c, intArg(args, "days", 14))
		}),
	)

	// get_training_summary
	s.AddTool(
		mcp.NewTool("get_training_summary",
			mcp.WithDescription("Get a training summary for a period: total distance, time, calories, number of workouts, breakdown by sport type."),
			mcp.WithString("start_date", mcp.Description("Start date YYYYMMDD. Defaults to 30 days ago.")),
			mcp.WithString("end_date", mcp.Description("End date YYYYMMDD. Defaults to today.")),
		),
		withAuth(func(c *http.Client, args map[string]interface{}) (string, error) {
			return getTrainingSummary(c, stringArg(args, "start_date", ""), stringArg(args, "end_date", ""))
		}),
	)

	// get_workout_comments
	s.AddTool(
		mcp.NewTool("get_workout_comments",
			mcp.WithDescription("Get comments on a specific workout"),
			mcp.WithString("label_id", mcp.Required(), mcp.Description("Workout labelId from get_workouts")),
		),
		withAuth(func(c *http.Client, args map[string]interface{}) (string, error) {
			return getWorkoutComments(c, args["label_id"].(string))
		}),
	)

	// get_import_list
	s.AddTool(
		mcp.NewTool("get_import_list",
			mcp.WithDescription("Get list of recently imported workout files"),
			mcp.WithNumber("size", mcp.Description("Number of results (default 10)")),
		),
		withAuth(func(c *http.Client, args map[string]interface{}) (string, error) {
			return getImportList(c, intArg(args, "size", 10))
		}),
	)

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// argument helpers

func intArg(args map[string]interface{}, key string, defaultVal int) int {
	if v, ok := args[key]; ok {
		if f, ok := v.(float64); ok {
			return int(f)
		}
	}
	return defaultVal
}

func stringArg(args map[string]interface{}, key, defaultVal string) string {
	if v, ok := args[key].(string); ok && v != "" {
		return v
	}
	return defaultVal
}
