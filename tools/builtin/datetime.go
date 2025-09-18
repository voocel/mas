package builtin

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
	"github.com/voocel/mas/tools"
)

// DateTimeTool offers time and date utilities
type DateTimeTool struct {
	*tools.BaseTool
}

// DateTimeInput captures time-related parameters
type DateTimeInput struct {
	Action   string `json:"action" description:"Action type: now, format, parse, add, diff, timezone"`
	DateTime string `json:"datetime,omitempty" description:"Datetime string"`
	Format   string `json:"format,omitempty" description:"Datetime format"`
	Amount   int    `json:"amount,omitempty" description:"Amount"`
	Unit     string `json:"unit,omitempty" description:"Time unit: year, month, day, hour, minute, second"`
	Timezone string `json:"timezone,omitempty" description:"Timezone (e.g., Asia/Shanghai, UTC)"`
	Target   string `json:"target,omitempty" description:"Target datetime (used to compute differences)"`
}

// DateTimeOutput describes the tool result
type DateTimeOutput struct {
	Success   bool   `json:"success"`
	Result    string `json:"result,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
	Timezone  string `json:"timezone,omitempty"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
}

// NewDateTimeTool constructs the datetime tool
func NewDateTimeTool() *DateTimeTool {
	schema := tools.CreateToolSchema(
		"Datetime tool supporting now, formatting, parsing, and arithmetic operations",
		map[string]interface{}{
			"action":   tools.StringProperty("Action type: now, format, parse, add, diff, timezone"),
			"datetime": tools.StringProperty("Datetime string"),
			"format":   tools.StringProperty("Datetime format"),
			"amount":   tools.NumberProperty("Amount"),
			"unit":     tools.StringProperty("Time unit: year, month, day, hour, minute, second"),
			"timezone": tools.StringProperty("Timezone (e.g., Asia/Shanghai, UTC)"),
			"target":   tools.StringProperty("Target datetime (used to compute differences)"),
		},
		[]string{"action"},
	)

	baseTool := tools.NewBaseTool("datetime", "Datetime tool supporting now, formatting, parsing, and arithmetic operations", schema)

	return &DateTimeTool{
		BaseTool: baseTool,
	}
}

// Execute performs the requested datetime operation
func (t *DateTimeTool) Execute(ctx runtime.Context, input json.RawMessage) (json.RawMessage, error) {
	var dtInput DateTimeInput
	if err := json.Unmarshal(input, &dtInput); err != nil {
		return nil, schema.NewToolError(t.Name(), "parse_input", err)
	}

	switch dtInput.Action {
	case "now":
		return t.getCurrentTime(dtInput.Timezone, dtInput.Format)
	case "format":
		return t.formatTime(dtInput.DateTime, dtInput.Format, dtInput.Timezone)
	case "parse":
		return t.parseTime(dtInput.DateTime, dtInput.Format, dtInput.Timezone)
	case "add":
		return t.addTime(dtInput.DateTime, dtInput.Amount, dtInput.Unit, dtInput.Timezone)
	case "diff":
		return t.diffTime(dtInput.DateTime, dtInput.Target, dtInput.Unit)
	case "timezone":
		return t.convertTimezone(dtInput.DateTime, dtInput.Timezone)
	default:
		output := DateTimeOutput{
			Success: false,
			Error:   fmt.Sprintf("unsupported action: %s", dtInput.Action),
		}
		return json.Marshal(output)
	}
}

// getCurrentTime returns the current time
func (t *DateTimeTool) getCurrentTime(timezone, format string) (json.RawMessage, error) {
	now := time.Now()

	// Apply timezone
	if timezone != "" {
		loc, err := time.LoadLocation(timezone)
		if err != nil {
			output := DateTimeOutput{
				Success: false,
				Error:   fmt.Sprintf("invalid timezone: %v", err),
			}
			return json.Marshal(output)
		}
		now = now.In(loc)
	}

	// Format the time
	result := now.Format(time.RFC3339)
	if format != "" {
		result = now.Format(t.parseFormat(format))
	}

	output := DateTimeOutput{
		Success:   true,
		Result:    result,
		Timestamp: now.Unix(),
		Timezone:  now.Location().String(),
		Message:   "current time retrieved successfully",
	}
	return json.Marshal(output)
}

// formatTime formats a datetime
func (t *DateTimeTool) formatTime(datetime, format, timezone string) (json.RawMessage, error) {
	if datetime == "" {
		output := DateTimeOutput{
			Success: false,
			Error:   "datetime cannot be empty",
		}
		return json.Marshal(output)
	}

	// Parse the datetime
	parsedTime, err := t.parseTimeString(datetime)
	if err != nil {
		output := DateTimeOutput{
			Success: false,
			Error:   fmt.Sprintf("failed to parse datetime: %v", err),
		}
		return json.Marshal(output)
	}

	// Apply timezone
	if timezone != "" {
		loc, err := time.LoadLocation(timezone)
		if err != nil {
			output := DateTimeOutput{
				Success: false,
				Error:   fmt.Sprintf("invalid timezone: %v", err),
			}
			return json.Marshal(output)
		}
		parsedTime = parsedTime.In(loc)
	}

	// Format the datetime
	result := parsedTime.Format(time.RFC3339)
	if format != "" {
		result = parsedTime.Format(t.parseFormat(format))
	}

	output := DateTimeOutput{
		Success:   true,
		Result:    result,
		Timestamp: parsedTime.Unix(),
		Timezone:  parsedTime.Location().String(),
		Message:   "time formatted successfully",
	}
	return json.Marshal(output)
}

// parseTime parses the datetime string
func (t *DateTimeTool) parseTime(datetime, format, timezone string) (json.RawMessage, error) {
	if datetime == "" {
		output := DateTimeOutput{
			Success: false,
			Error:   "datetime cannot be empty",
		}
		return json.Marshal(output)
	}

	var parsedTime time.Time
	var err error

	if format != "" {
		parsedTime, err = time.Parse(t.parseFormat(format), datetime)
	} else {
		parsedTime, err = t.parseTimeString(datetime)
	}

	if err != nil {
		output := DateTimeOutput{
			Success: false,
			Error:   fmt.Sprintf("failed to parse datetime: %v", err),
		}
		return json.Marshal(output)
	}

	// Apply timezone
	if timezone != "" {
		loc, err := time.LoadLocation(timezone)
		if err != nil {
			output := DateTimeOutput{
				Success: false,
				Error:   fmt.Sprintf("invalid timezone: %v", err),
			}
			return json.Marshal(output)
		}
		parsedTime = parsedTime.In(loc)
	}

	output := DateTimeOutput{
		Success:   true,
		Result:    parsedTime.Format(time.RFC3339),
		Timestamp: parsedTime.Unix(),
		Timezone:  parsedTime.Location().String(),
		Message:   "time parsed successfully",
	}
	return json.Marshal(output)
}

// addTime performs time arithmetic
func (t *DateTimeTool) addTime(datetime string, amount int, unit, timezone string) (json.RawMessage, error) {
	if datetime == "" {
		output := DateTimeOutput{
			Success: false,
			Error:   "datetime cannot be empty",
		}
		return json.Marshal(output)
	}

	// Parse the datetime
	parsedTime, err := t.parseTimeString(datetime)
	if err != nil {
		output := DateTimeOutput{
			Success: false,
			Error:   fmt.Sprintf("failed to parse datetime: %v", err),
		}
		return json.Marshal(output)
	}

	// Add the duration
	var result time.Time
	switch unit {
	case "year":
		result = parsedTime.AddDate(amount, 0, 0)
	case "month":
		result = parsedTime.AddDate(0, amount, 0)
	case "day":
		result = parsedTime.AddDate(0, 0, amount)
	case "hour":
		result = parsedTime.Add(time.Duration(amount) * time.Hour)
	case "minute":
		result = parsedTime.Add(time.Duration(amount) * time.Minute)
	case "second":
		result = parsedTime.Add(time.Duration(amount) * time.Second)
	default:
		output := DateTimeOutput{
			Success: false,
			Error:   fmt.Sprintf("unsupported time unit: %s", unit),
		}
		return json.Marshal(output)
	}

	// Apply timezone
	if timezone != "" {
		loc, err := time.LoadLocation(timezone)
		if err != nil {
			output := DateTimeOutput{
				Success: false,
				Error:   fmt.Sprintf("invalid timezone: %v", err),
			}
			return json.Marshal(output)
		}
		result = result.In(loc)
	}

	output := DateTimeOutput{
		Success:   true,
		Result:    result.Format(time.RFC3339),
		Timestamp: result.Unix(),
		Timezone:  result.Location().String(),
		Message:   fmt.Sprintf("added %d %s(s) successfully", amount, unit),
	}
	return json.Marshal(output)
}

// diffTime calculates the difference between datetimes
func (t *DateTimeTool) diffTime(datetime, target, unit string) (json.RawMessage, error) {
	if datetime == "" || target == "" {
		output := DateTimeOutput{
			Success: false,
			Error:   "both datetime and target are required",
		}
		return json.Marshal(output)
	}

	// Parse the datetime
	time1, err := t.parseTimeString(datetime)
	if err != nil {
		output := DateTimeOutput{
			Success: false,
			Error:   fmt.Sprintf("failed to parse datetime: %v", err),
		}
		return json.Marshal(output)
	}

	time2, err := t.parseTimeString(target)
	if err != nil {
		output := DateTimeOutput{
			Success: false,
			Error:   fmt.Sprintf("failed to parse target: %v", err),
		}
		return json.Marshal(output)
	}

	// Compute the difference
	diff := time2.Sub(time1)
	var result string

	switch unit {
	case "second":
		result = fmt.Sprintf("%.0f", diff.Seconds())
	case "minute":
		result = fmt.Sprintf("%.2f", diff.Minutes())
	case "hour":
		result = fmt.Sprintf("%.2f", diff.Hours())
	case "day":
		result = fmt.Sprintf("%.2f", diff.Hours()/24)
	default:
		result = diff.String()
	}

	output := DateTimeOutput{
		Success: true,
		Result:  result,
		Message: fmt.Sprintf("time difference calculated in %s", unit),
	}
	return json.Marshal(output)
}

// convertTimezone converts the timezone
func (t *DateTimeTool) convertTimezone(datetime, timezone string) (json.RawMessage, error) {
	if datetime == "" || timezone == "" {
		output := DateTimeOutput{
			Success: false,
			Error:   "both datetime and timezone are required",
		}
		return json.Marshal(output)
	}

	// Parse the datetime
	parsedTime, err := t.parseTimeString(datetime)
	if err != nil {
		output := DateTimeOutput{
			Success: false,
			Error:   fmt.Sprintf("failed to parse datetime: %v", err),
		}
		return json.Marshal(output)
	}

	// Convert timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		output := DateTimeOutput{
			Success: false,
			Error:   fmt.Sprintf("invalid timezone: %v", err),
		}
		return json.Marshal(output)
	}

	result := parsedTime.In(loc)

	output := DateTimeOutput{
		Success:   true,
		Result:    result.Format(time.RFC3339),
		Timestamp: result.Unix(),
		Timezone:  result.Location().String(),
		Message:   fmt.Sprintf("timezone converted to %s", timezone),
	}
	return json.Marshal(output)
}

// parseTimeString parses a datetime string using multiple layouts
func (t *DateTimeTool) parseTimeString(datetime string) (time.Time, error) {
	// Try parsing as a timestamp
	if timestamp, err := strconv.ParseInt(datetime, 10, 64); err == nil {
		return time.Unix(timestamp, 0), nil
	}

	// Try common layouts
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02",
		"15:04:05",
		"2006/01/02 15:04:05",
		"2006/01/02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, datetime); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse datetime: %s", datetime)
}

// parseFormat interprets the provided layout
func (t *DateTimeTool) parseFormat(format string) string {
	// Support several common format aliases
	switch format {
	case "iso":
		return time.RFC3339
	case "date":
		return "2006-01-02"
	case "time":
		return "15:04:05"
	case "datetime":
		return "2006-01-02 15:04:05"
	default:
		return format
	}
}

// ExecuteAsync performs the datetime operation asynchronously
func (t *DateTimeTool) ExecuteAsync(ctx runtime.Context, input json.RawMessage) (<-chan tools.ToolResult, error) {
	resultChan := make(chan tools.ToolResult, 1)

	go func() {
		defer close(resultChan)

		result, err := t.Execute(ctx, input)
		if err != nil {
			resultChan <- tools.ToolResult{
				Success: false,
				Error:   err.Error(),
			}
			return
		}

		resultChan <- tools.ToolResult{
			Success: true,
			Data:    result,
		}
	}()

	return resultChan, nil
}
