package discover

import (
	"strings"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/go-minitrace/pkg/adapters"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/pkg/errors"
)

// filterFlags returns the shared discovery filter flags applied to every
// discover subcommand.
func filterFlags() []*fields.Definition {
	return []*fields.Definition{
		fields.New(
			"cwd-contains",
			fields.TypeString,
			fields.WithDefault(""),
			fields.WithHelp("Only keep sessions whose cwd contains this case-sensitive substring"),
		),
		fields.New(
			"since",
			fields.TypeString,
			fields.WithDefault(""),
			fields.WithHelp("Only keep sessions started at or after this time (RFC3339 or YYYY-MM-DD); sessions without a start timestamp are excluded"),
		),
	}
}

// activityFilterFlags returns filters that require full transcript scans and
// are therefore available only on adapters with native JSONL activity support.
func activityFilterFlags() []*fields.Definition {
	return []*fields.Definition{
		fields.New(
			"active-since",
			fields.TypeString,
			fields.WithDefault(""),
			fields.WithHelp("Only keep sessions with activity at or after this time (RFC3339 or YYYY-MM-DD); scans native transcripts and may be slow"),
		),
	}
}

// parseSince parses the --since flag value, accepting RFC3339 timestamps or
// YYYY-MM-DD dates (interpreted as UTC midnight). An empty value yields nil.
func parseSince(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		utc := ts.UTC()
		return &utc, nil
	}
	if ts, err := time.Parse("2006-01-02", value); err == nil {
		utc := ts.UTC()
		return &utc, nil
	}
	return nil, errors.Errorf("invalid --since value %q: expected RFC3339 timestamp or YYYY-MM-DD date", value)
}

// keepLocator reports whether a discovered session locator passes the
// --cwd-contains, --since, and --active-since filters. When a time filter is
// set, locators without the corresponding parseable timestamp are excluded.
func keepLocator(locator adapters.SessionLocator, cwdContains string, since, activeSince *time.Time) bool {
	if cwdContains != "" && !strings.Contains(locator.Cwd, cwdContains) {
		return false
	}
	if since != nil {
		startedAt, ok := minitrace.ParseTimestamp(locator.StartedAt)
		if !ok {
			return false
		}
		if startedAt.Before(*since) {
			return false
		}
	}
	if activeSince != nil {
		lastActivityAt, ok := minitrace.ParseTimestamp(locator.LastActivityAt)
		if !ok || lastActivityAt.Before(*activeSince) {
			return false
		}
	}
	return true
}
