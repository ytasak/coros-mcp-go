package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// Tool argument structs (schemas are auto-inferred by the SDK)

type emptyArgs struct{}

type getWorkoutsArgs struct {
	StartDate string `json:"start_date,omitempty" jsonschema:"description=Start date YYYYMMDD. Defaults to 30 days ago."`
	EndDate   string `json:"end_date,omitempty" jsonschema:"description=End date YYYYMMDD. Defaults to today."`
	Page      int    `json:"page,omitempty" jsonschema:"description=Page number (default 1)"`
	Size      int    `json:"size,omitempty" jsonschema:"description=Results per page (default 20)"`
}

type workoutIDArgs struct {
	LabelID   string `json:"label_id" jsonschema:"required,description=Workout labelId from get_workouts"`
	SportType string `json:"sport_type" jsonschema:"required,description=sportType code from get_workouts (e.g. 101 for run)"`
}

type getWorkoutFileArgs struct {
	LabelID   string `json:"label_id" jsonschema:"required,description=Workout labelId from get_workouts"`
	SportType string `json:"sport_type" jsonschema:"required,description=sportType code from get_workouts (e.g. 101 for run)"`
	FileType  string `json:"file_type,omitempty" jsonschema:"description=File format: fit\\, tcx\\, gpx\\, or kml (default: fit)"`
}

type getRecentRunsArgs struct {
	Days int `json:"days,omitempty" jsonschema:"description=Number of days to look back (max 90\\, default 14)"`
}

type dateRangeArgs struct {
	StartDate string `json:"start_date,omitempty" jsonschema:"description=Start date YYYYMMDD. Defaults to 30 days ago."`
	EndDate   string `json:"end_date,omitempty" jsonschema:"description=End date YYYYMMDD. Defaults to today."`
}

type labelIDArgs struct {
	LabelID string `json:"label_id" jsonschema:"required,description=Workout labelId from get_workouts"`
}

type getSizeArgs struct {
	Size int `json:"size,omitempty" jsonschema:"description=Number of results (default 10)"`
}

// withAuth wraps an API call with authentication and auto-retry on token expiry.
func withAuth(fn func(*http.Client) (string, error)) (*mcp.CallToolResult, any, error) {
	if err := ensureLoggedIn(httpClient, false); err != nil {
		return errResult(err.Error()), nil, nil
	}

	result, err := fn(httpClient)
	if err != nil {
		return errResult(err.Error()), nil, nil
	}

	if isAuthError(result) {
		log.Println("Token expired, re-authenticating...")
		if err := ensureLoggedIn(httpClient, true); err != nil {
			return errResult(err.Error()), nil, nil
		}
		result, err = fn(httpClient)
		if err != nil {
			return errResult(err.Error()), nil, nil
		}
	}

	return textResult(result), nil, nil
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

func errResult(msg string) *mcp.CallToolResult {
	r := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
	r.SetError(fmt.Errorf("%s", msg))
	return r
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
	s := mcp.NewServer(&mcp.Implementation{
		Name:    "coros-mcp",
		Version: "0.1.0",
	}, nil)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_user_info",
		Description: "Get COROS user profile info",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args emptyArgs) (*mcp.CallToolResult, any, error) {
		return withAuth(func(c *http.Client) (string, error) {
			return getUserInfo(c)
		})
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_workouts",
		Description: "Get workout records for a date range. Returns list of activities with distance, pace, duration, calories, etc.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args getWorkoutsArgs) (*mcp.CallToolResult, any, error) {
		page := args.Page
		if page == 0 {
			page = 1
		}
		size := args.Size
		if size == 0 {
			size = 20
		}
		return withAuth(func(c *http.Client) (string, error) {
			return getWorkouts(c, args.StartDate, args.EndDate, page, size)
		})
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_workout_detail",
		Description: "Get detailed data for a specific workout by labelId",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args workoutIDArgs) (*mcp.CallToolResult, any, error) {
		return withAuth(func(c *http.Client) (string, error) {
			return getWorkoutDetail(c, args.LabelID, args.SportType)
		})
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_workout_file",
		Description: "Get download URL for a workout file (.fit/.tcx/.gpx)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args getWorkoutFileArgs) (*mcp.CallToolResult, any, error) {
		fileType := args.FileType
		if fileType == "" {
			fileType = "fit"
		}
		return withAuth(func(c *http.Client) (string, error) {
			return getWorkoutFile(c, args.LabelID, args.SportType, fileType)
		})
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_recent_runs",
		Description: "Get recent running workouts (outdoor + trail runs) for the last N days with formatted stats.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args getRecentRunsArgs) (*mcp.CallToolResult, any, error) {
		days := args.Days
		if days == 0 {
			days = 14
		}
		return withAuth(func(c *http.Client) (string, error) {
			return getRecentRuns(c, days)
		})
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_training_summary",
		Description: "Get a training summary for a period: total distance, time, calories, number of workouts, breakdown by sport type.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args dateRangeArgs) (*mcp.CallToolResult, any, error) {
		return withAuth(func(c *http.Client) (string, error) {
			return getTrainingSummary(c, args.StartDate, args.EndDate)
		})
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_workout_comments",
		Description: "Get comments on a specific workout",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args labelIDArgs) (*mcp.CallToolResult, any, error) {
		return withAuth(func(c *http.Client) (string, error) {
			return getWorkoutComments(c, args.LabelID)
		})
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_import_list",
		Description: "Get list of recently imported workout files",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args getSizeArgs) (*mcp.CallToolResult, any, error) {
		size := args.Size
		if size == 0 {
			size = 10
		}
		return withAuth(func(c *http.Client) (string, error) {
			return getImportList(c, size)
		})
	})

	if err := s.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
