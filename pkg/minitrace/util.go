package minitrace

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	SchemaVersion = "minitrace-v0.2.0"
	TruncateLimit = 10240
)

var (
	IdleThreshold = 5 * time.Minute

	safeIDPattern = regexp.MustCompile(`[^a-zA-Z0-9_\-.]`)
	periodPattern = regexp.MustCompile(`^\d{4}-\d{2}$`)
)

func ptr[T any](v T) *T {
	return &v
}

func ParseTimestamp(ts string) (time.Time, bool) {
	if strings.TrimSpace(ts) == "" {
		return time.Time{}, false
	}

	normalized := strings.Replace(ts, "Z", "+00:00", 1)
	parsed, err := time.Parse(time.RFC3339, normalized)
	if err != nil {
		return time.Time{}, false
	}

	return parsed.UTC(), true
}

func FormatTimestamp(ts time.Time) string {
	return ts.UTC().Format(time.RFC3339)
}

func NowUTC() time.Time {
	return time.Now().UTC()
}

func SafeFromTimestamp(epoch any, autoMS bool) (time.Time, bool) {
	if epoch == nil {
		return time.Time{}, false
	}

	value, err := strconv.ParseFloat(fmt.Sprint(epoch), 64)
	if err != nil {
		return time.Time{}, false
	}
	if autoMS && value > 1e12 {
		value = value / 1000
	}

	seconds := int64(value)
	nanos := int64((value - float64(seconds)) * float64(time.Second))
	return time.Unix(seconds, nanos).UTC(), true
}

func SanitizeID(rawID string) string {
	if rawID == "" {
		return "unknown"
	}

	clean := safeIDPattern.ReplaceAllString(rawID, "_")
	clean = strings.TrimLeft(clean, ".")
	if clean == "" {
		return "unknown"
	}
	if len(clean) > 200 {
		return clean[:200]
	}
	return clean
}

func SanitizePeriod(period string) string {
	if periodPattern.MatchString(period) {
		return period
	}
	return "unknown"
}

func SafeInt(value any, defaults int) int {
	if value == nil {
		return defaults
	}

	switch v := value.(type) {
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		if v < int64(math.MinInt) || v > int64(math.MaxInt) {
			return defaults
		}
		return int(v)
	case uint:
		if v > uint(math.MaxInt) {
			return defaults
		}
		return int(v)
	case uint8:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		if v > uint64(math.MaxInt) {
			return defaults
		}
		return int(v)
	case float32:
		if v < float32(math.MinInt) || v > float32(math.MaxInt) {
			return defaults
		}
		return int(v)
	case float64:
		if v < float64(math.MinInt) || v > float64(math.MaxInt) {
			return defaults
		}
		return int(v)
	case string:
		i, err := strconv.Atoi(v)
		if err != nil {
			return defaults
		}
		return i
	default:
		i, err := strconv.Atoi(fmt.Sprint(value))
		if err != nil {
			return defaults
		}
		return i
	}
}

func TruncateContent(content any, limit int) (*string, *int, *string) {
	if content == nil {
		return nil, nil, nil
	}

	text := fmt.Sprint(content)
	encoded := []byte(text)
	fullBytes := len(encoded)
	if fullBytes <= limit {
		return ptr(text), nil, nil
	}

	sum := sha256.Sum256(encoded)
	fullHash := "sha256:" + hex.EncodeToString(sum[:])
	truncated := strings.ToValidUTF8(string(encoded[:limit]), "")
	truncated = truncated + "\n[truncated]"

	return ptr(truncated), ptr(fullBytes), ptr(fullHash)
}

// DurationBetweenMS returns the difference between two RFC3339 timestamps in
// milliseconds, or nil when either timestamp is missing/unparseable or the
// end precedes the start.
func DurationBetweenMS(start, end *string) *int {
	if start == nil || end == nil {
		return nil
	}
	startTime, ok := ParseTimestamp(*start)
	if !ok {
		return nil
	}
	endTime, ok := ParseTimestamp(*end)
	if !ok {
		return nil
	}
	if endTime.Before(startTime) {
		return nil
	}
	durationMS := int(endTime.Sub(startTime).Milliseconds())
	return &durationMS
}

func NormalizePath(filePath string) string {
	if filePath == "" {
		return filePath
	}

	clean := filepath.Clean(filePath)
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return clean
	}

	if clean == home {
		return "~"
	}

	prefix := home + string(os.PathSeparator)
	if strings.HasPrefix(clean, prefix) {
		return "~" + strings.TrimPrefix(clean, home)
	}

	return clean
}

func DeduplicateToolCalls(toolCalls []ToolCall, keyFn func(ToolCall) string) ([]ToolCall, int) {
	if keyFn == nil {
		keyFn = func(tc ToolCall) string {
			return tc.ID
		}
	}

	seen := map[string]struct{}{}
	unique := make([]ToolCall, 0, len(toolCalls))
	duplicates := 0

	for _, toolCall := range toolCalls {
		key := keyFn(toolCall)
		if key == "" {
			unique = append(unique, toolCall)
			continue
		}
		if _, ok := seen[key]; ok {
			duplicates++
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, toolCall)
	}

	return unique, duplicates
}

func DetectPIIInPaths(toolCalls []ToolCall) bool {
	for _, toolCall := range toolCalls {
		if toolCall.Input.FilePath == nil {
			continue
		}
		if strings.Contains(*toolCall.Input.FilePath, "/Users/") || strings.Contains(*toolCall.Input.FilePath, "/home/") {
			return true
		}
	}
	return false
}

func ExtractTitle(turns []Turn, maxLen int) *string {
	if maxLen <= 0 {
		maxLen = 80
	}

	for _, turn := range turns {
		if turn.Role != "user" || turn.Source == nil || *turn.Source != "human" {
			continue
		}
		if strings.TrimSpace(turn.Content) == "" {
			continue
		}

		title := strings.ReplaceAll(turn.Content, "\n", " ")
		title = strings.TrimSpace(title)
		if len(title) > maxLen {
			title = title[:maxLen]
		}
		return ptr(title)
	}

	return nil
}

func sortedTimes(timestamps []time.Time) []time.Time {
	cloned := slices.Clone(timestamps)
	slices.SortFunc(cloned, func(a, b time.Time) int {
		switch {
		case a.Before(b):
			return -1
		case a.After(b):
			return 1
		default:
			return 0
		}
	})
	return cloned
}
