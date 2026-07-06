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
// --cwd-contains and --since filters. When since is set, locators without a
// parseable start timestamp are excluded.
func keepLocator(locator adapters.SessionLocator, cwdContains string, since *time.Time) bool {
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
	return true
}
