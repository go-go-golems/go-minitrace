package discover

import (
	"github.com/go-go-golems/go-minitrace/pkg/adapters"
)

type activityTimestampScanner func(path string) (string, error)

func populateLastActivity(locator *adapters.SessionLocator, scan activityTimestampScanner) error {
	lastActivityAt, err := scan(locator.SourcePath)
	if err != nil {
		return err
	}
	locator.LastActivityAt = lastActivityAt
	return nil
}
