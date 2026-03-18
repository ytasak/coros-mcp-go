package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// apiResponse is the common response envelope from COROS API.
type apiResponse struct {
	Result  string          `json:"result"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// HTTP helpers

func getJSON(client *http.Client, rawURL string, params url.Values, out interface{}) error {
	if params != nil {
		rawURL += "?" + params.Encode()
	}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return err
	}
	for k, vs := range currentSession.getAuthHeaders() {
		for _, v := range vs {
			req.Header.Set(k, v)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func postJSON(client *http.Client, rawURL string, params url.Values, payload interface{}, out interface{}) error {
	if params != nil {
		rawURL += "?" + params.Encode()
	}
	var reqBody io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequest("POST", rawURL, reqBody)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, vs := range currentSession.getAuthHeaders() {
		for _, v := range vs {
			req.Header.Set(k, v)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

// API functions

func getUserInfo(client *http.Client) (string, error) {
	var resp apiResponse
	if err := getJSON(client, baseURL+"/account/query", nil, &resp); err != nil {
		return "", err
	}
	if resp.Result != "0000" {
		return apiError(resp), nil
	}

	var d map[string]interface{}
	json.Unmarshal(resp.Data, &d)

	nickName := strVal(d, "nickName")
	if nickName == "" {
		nickName = strVal(d, "nick")
	}
	if nickName == "" {
		nickName = "N/A"
	}

	gender := "N/A"
	if g, ok := d["gender"].(float64); ok {
		switch int(g) {
		case 1:
			gender = "Male"
		case 2:
			gender = "Female"
		}
	}

	return fmt.Sprintf(
		"COROS User: %s\n  userId: %s\n  Email: %s\n  Birthday: %s\n  Gender: %s",
		nickName,
		strValOr(d, "userId", "N/A"),
		strValOr(d, "email", "N/A"),
		strValOr(d, "birthday", "N/A"),
		gender,
	), nil
}

type workoutItem struct {
	Mode         int     `json:"mode"`
	SubMode      int     `json:"subMode"`
	Distance     float64 `json:"distance"`
	WorkoutTime  int     `json:"workoutTime"`
	TotalTime    int     `json:"totalTime"`
	AvgSpeed     float64 `json:"avgSpeed"`
	Calorie      float64 `json:"calorie"`
	AvgCadence   int     `json:"avgCadence"`
	AvgFrequency int     `json:"avgFrequency"`
	StartTime    int64   `json:"startTime"`
	LabelID      string  `json:"labelId"`
	SportType    string  `json:"sportType"`
	Name         string  `json:"name"`
}

func (w workoutItem) duration() int {
	if w.WorkoutTime > 0 {
		return w.WorkoutTime
	}
	return w.TotalTime
}

func (w workoutItem) cadence() int {
	if w.AvgCadence > 0 {
		return w.AvgCadence
	}
	return w.AvgFrequency
}

type activityQueryData struct {
	DataList []workoutItem `json:"dataList"`
	Count    int           `json:"count"`
}

func getWorkouts(client *http.Client, startDate, endDate string, page, size int) (string, error) {
	params := url.Values{
		"size":       {fmt.Sprintf("%d", size)},
		"pageNumber": {fmt.Sprintf("%d", page)},
	}
	if startDate != "" {
		params.Set("startDay", startDate)
	}
	if endDate != "" {
		params.Set("endDay", endDate)
	}

	var resp apiResponse
	if err := getJSON(client, baseURL+"/activity/query", params, &resp); err != nil {
		return "", err
	}
	if resp.Result != "0000" {
		return apiError(resp), nil
	}

	var data activityQueryData
	json.Unmarshal(resp.Data, &data)

	if len(data.DataList) == 0 {
		return "No workouts found", nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Workouts (%d/%d total):\n", len(data.DataList), data.Count)
	for _, w := range data.DataList {
		sport := workoutTypeName(w.Mode, w.SubMode)
		distKm := math.Round(w.Distance/1000*100) / 100
		dur := formatDuration(w.duration())
		pace := formatPace(w.AvgSpeed)
		cal := int(math.Round(w.Calorie / 1000))
		dt := parseTimestamp(w.StartTime)

		sb.WriteString(fmt.Sprintf("\n  [%s] %s", dt, sport))
		if w.Name != "" {
			fmt.Fprintf(&sb, " - %s", w.Name)
		}
		fmt.Fprintf(&sb, "\n    Distance: %.2f km | Duration: %s | Pace: %s", distKm, dur, pace)
		fmt.Fprintf(&sb, "\n    Calories: %d kcal | Cadence: %d spm", cal, w.cadence())
		fmt.Fprintf(&sb, "\n    labelId: %s | sportType: %s", w.LabelID, w.SportType)
	}
	return sb.String(), nil
}

func getWorkoutDetail(client *http.Client, labelID, sportType string) (string, error) {
	params := url.Values{
		"labelId":   {labelID},
		"sportType": {sportType},
	}

	var resp apiResponse
	if err := postJSON(client, baseURL+"/activity/detail/query", params, nil, &resp); err != nil {
		return "", err
	}
	if resp.Result != "0000" {
		return apiError(resp), nil
	}

	var d interface{}
	json.Unmarshal(resp.Data, &d)
	b, _ := json.MarshalIndent(d, "", "  ")
	return string(b), nil
}

func getWorkoutFile(client *http.Client, labelID, sportType, fileType string) (string, error) {
	fileTypeMap := map[string]string{"fit": "4", "tcx": "3", "gpx": "5", "kml": "6"}
	ft := fileTypeMap[fileType]
	if ft == "" {
		ft = "4"
	}

	params := url.Values{
		"labelId":   {labelID},
		"sportType": {sportType},
		"fileType":  {ft},
	}

	var resp apiResponse
	if err := postJSON(client, baseURL+"/activity/detail/download", params, nil, &resp); err != nil {
		return "", err
	}
	if resp.Result != "0000" {
		return apiError(resp), nil
	}

	var d map[string]interface{}
	json.Unmarshal(resp.Data, &d)
	fileURL := strValOr(d, "fileUrl", "N/A")
	return fmt.Sprintf("Download URL (labelId: %s):\n  %s", labelID, fileURL), nil
}

func getRecentRuns(client *http.Client, days int) (string, error) {
	if days > 90 {
		days = 90
	}
	now := time.Now().UTC()
	start := formatDate(now.AddDate(0, 0, -days))
	end := formatDate(now)

	runModes := map[int]bool{8: true, 15: true, 20: true}
	var allRuns []workoutItem

	for page := 1; ; page++ {
		params := url.Values{
			"size":       {"50"},
			"pageNumber": {fmt.Sprintf("%d", page)},
			"startDay":   {start},
			"endDay":     {end},
		}

		var resp apiResponse
		if err := getJSON(client, baseURL+"/activity/query", params, &resp); err != nil {
			return "", err
		}
		if resp.Result != "0000" {
			return apiError(resp), nil
		}

		var data activityQueryData
		json.Unmarshal(resp.Data, &data)
		if len(data.DataList) == 0 {
			break
		}

		for _, w := range data.DataList {
			if runModes[w.Mode] {
				allRuns = append(allRuns, w)
			}
		}

		if page*50 >= data.Count {
			break
		}
	}

	if len(allRuns) == 0 {
		return fmt.Sprintf("No runs found in the last %d days", days), nil
	}

	var totalDist float64
	var totalDur int
	var totalCal float64
	for _, r := range allRuns {
		totalDist += r.Distance
		totalDur += r.duration()
		totalCal += r.Calorie
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Last %d days: %d runs | %.1f km | %s | %d kcal\n",
		days, len(allRuns), totalDist/1000, formatDuration(totalDur), int(math.Round(totalCal/1000)))

	sort.Slice(allRuns, func(i, j int) bool {
		return allRuns[i].StartTime > allRuns[j].StartTime
	})

	for _, r := range allRuns {
		sport := workoutTypeName(r.Mode, r.SubMode)
		distKm := math.Round(r.Distance/1000*100) / 100
		dur := formatDuration(r.duration())
		pace := formatPace(r.AvgSpeed)
		dt := parseTimestamp(r.StartTime)

		fmt.Fprintf(&sb, "\n  [%s] %s: %.2f km in %s @ %s", dt[:10], sport, distKm, dur, pace)
		if cad := r.cadence(); cad > 0 {
			fmt.Fprintf(&sb, " | %d spm", cad)
		}
		if r.Name != "" {
			fmt.Fprintf(&sb, " (%s)", r.Name)
		}
	}
	return sb.String(), nil
}

func getTrainingSummary(client *http.Client, startDate, endDate string) (string, error) {
	now := time.Now().UTC()
	if startDate == "" {
		startDate = formatDate(now.AddDate(0, 0, -30))
	}
	if endDate == "" {
		endDate = formatDate(now)
	}

	var allWorkouts []workoutItem
	for page := 1; ; page++ {
		params := url.Values{
			"size":       {"50"},
			"pageNumber": {fmt.Sprintf("%d", page)},
			"startDay":   {startDate},
			"endDay":     {endDate},
		}

		var resp apiResponse
		if err := getJSON(client, baseURL+"/activity/query", params, &resp); err != nil {
			return "", err
		}
		if resp.Result != "0000" {
			return apiError(resp), nil
		}

		var data activityQueryData
		json.Unmarshal(resp.Data, &data)
		if len(data.DataList) == 0 {
			break
		}
		allWorkouts = append(allWorkouts, data.DataList...)

		if page*50 >= data.Count {
			break
		}
	}

	if len(allWorkouts) == 0 {
		return fmt.Sprintf("No workouts found between %s and %s", startDate, endDate), nil
	}

	type sportTotal struct {
		Count    int
		Distance float64
		Duration int
		Calories float64
	}
	totals := map[string]*sportTotal{}

	for _, w := range allWorkouts {
		sport := workoutTypeName(w.Mode, w.SubMode)
		t, ok := totals[sport]
		if !ok {
			t = &sportTotal{}
			totals[sport] = t
		}
		t.Count++
		t.Distance += w.Distance
		t.Duration += w.duration()
		t.Calories += w.Calorie
	}

	var grandDist float64
	var grandDur int
	var grandCal float64
	for _, v := range totals {
		grandDist += v.Distance
		grandDur += v.Duration
		grandCal += v.Calories
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Training Summary %s -> %s\n", startDate, endDate)
	fmt.Fprintf(&sb, "\n  Total: %d workouts | %.1f km | %s | %d kcal\n",
		len(allWorkouts), grandDist/1000, formatDuration(grandDur), int(math.Round(grandCal/1000)))
	sb.WriteString("\n  By sport:")

	// Sort by distance descending
	type kv struct {
		sport string
		total *sportTotal
	}
	var sorted []kv
	for k, v := range totals {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].total.Distance > sorted[j].total.Distance
	})

	for _, s := range sorted {
		fmt.Fprintf(&sb, "\n    %s: %dx | %.1f km | %s | %d kcal",
			s.sport, s.total.Count, s.total.Distance/1000,
			formatDuration(s.total.Duration), int(math.Round(s.total.Calories/1000)))
	}
	return sb.String(), nil
}

func getWorkoutComments(client *http.Client, labelID string) (string, error) {
	params := url.Values{
		"dataId": {labelID},
		"type":   {"1"},
	}

	var resp apiResponse
	if err := getJSON(client, baseURL+"/leavingmessage/list", params, &resp); err != nil {
		return "", err
	}
	if resp.Result != "0000" {
		return apiError(resp), nil
	}

	var comments []map[string]interface{}
	json.Unmarshal(resp.Data, &comments)

	if len(comments) == 0 {
		return fmt.Sprintf("No comments on workout %s", labelID), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Comments on workout %s (%d):\n", labelID, len(comments))
	for _, c := range comments {
		author := strValOr(c, "nickName", "Unknown")
		content := strValOr(c, "content", "")
		ts := int64(numVal(c, "createTime"))
		if ts > 1e12 {
			ts /= 1000
		}
		dt := parseTimestamp(ts)
		fmt.Fprintf(&sb, "\n  [%s] %s: %s", dt, author, content)
	}
	return sb.String(), nil
}

func getImportList(client *http.Client, size int) (string, error) {
	body := map[string]int{"size": size}

	var resp apiResponse
	if err := postJSON(client, baseURL+"/activity/fit/getImportSportList", nil, body, &resp); err != nil {
		return "", err
	}
	if resp.Result != "0000" {
		return apiError(resp), nil
	}

	var imports []interface{}
	json.Unmarshal(resp.Data, &imports)

	if len(imports) == 0 {
		return "No imported workouts", nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Imported workouts (%d):\n", len(imports))
	for _, item := range imports {
		b, _ := json.MarshalIndent(item, "", "  ")
		sb.WriteString("\n")
		sb.Write(b)
	}
	return sb.String(), nil
}

// helpers

func apiError(resp apiResponse) string {
	return fmt.Sprintf("API Error %s: %s", resp.Result, resp.Message)
}

func strVal(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func strValOr(m map[string]interface{}, key, fallback string) string {
	if s := strVal(m, key); s != "" {
		return s
	}
	return fallback
}

func numVal(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}
