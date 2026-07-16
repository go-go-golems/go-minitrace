package minitrace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/pkg/errors"
)

// PublicationStatus describes what staged batch publication did with one
// archive destination.
type PublicationStatus string

const (
	PublicationCreated   PublicationStatus = "created"
	PublicationUnchanged PublicationStatus = "unchanged"
	PublicationReplaced  PublicationStatus = "replaced"
)

// PublicationResult is the per-session result of a staged batch publication.
type PublicationResult struct {
	SessionID string
	Status    PublicationStatus
	Entry     *SessionIndexEntry
}

type stagedSession struct {
	session     *Session
	period      string
	destination string
	stagedPath  string
	backupPath  string
	status      PublicationStatus
}

// PublishSessionBatch stages every changed archive before publishing any of
// them. Collision checks also run for the complete batch before publication.
// If a rename fails, changes made by this invocation are rolled back. A process
// crash can still interrupt the multi-file rename sequence; callers must not
// treat this as filesystem-wide atomicity.
func PublishSessionBatch(sessions []*Session, outputDir string, policy CollisionPolicy) ([]PublicationResult, error) {
	if policy != CollisionError && policy != CollisionReplace {
		return nil, errors.Errorf("unsupported collision policy %q", policy)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, errors.Wrap(err, "creating archive output directory")
	}

	stageRoot, err := os.MkdirTemp(outputDir, ".minitrace-batch-")
	if err != nil {
		return nil, errors.Wrap(err, "creating batch staging directory")
	}
	defer func() { _ = os.RemoveAll(stageRoot) }()

	planned := make([]stagedSession, 0, len(sessions))
	destinations := make(map[string]*Session)
	for _, session := range sessions {
		if session == nil {
			return nil, errors.New("batch contains nil session")
		}
		period := sessionPeriod(session)
		if session.Provenance.SourcePath != nil {
			normalized := NormalizePath(*session.Provenance.SourcePath)
			session.Provenance.SourcePath = &normalized
		}
		relative := filepath.Join("active", period, SanitizeID(session.ID)+".minitrace.json")
		destination := filepath.Join(outputDir, relative)
		if previous, ok := destinations[destination]; ok {
			if !sameSourceFingerprint(previous, session) || !sameSessionContent(previous, session) {
				return nil, errors.Errorf("batch contains conflicting sessions for destination %s", destination)
			}
			continue
		}
		destinations[destination] = session

		item := stagedSession{session: session, period: period, destination: destination, status: PublicationCreated}
		if existing, _, readErr := readExistingSession(destination); readErr == nil {
			if sameSourceFingerprint(existing, session) {
				if sameSessionContent(existing, session) {
					item.status = PublicationUnchanged
					planned = append(planned, item)
					continue
				}
				// The same source bytes may produce changed derived metadata, such
				// as newly discovered Claude subagent backlinks. This is a safe
				// replacement because source ownership is unchanged.
				item.status = PublicationReplaced
			} else if policy != CollisionReplace {
				return nil, errors.Errorf("archive collision for session %q at %s; use explicit replacement only after verifying source provenance", session.ID, destination)
			}
			item.status = PublicationReplaced
		} else if !os.IsNotExist(readErr) {
			return nil, readErr
		}

		payload, marshalErr := json.MarshalIndent(session, "", "  ")
		if marshalErr != nil {
			return nil, errors.Wrapf(marshalErr, "marshaling session %s", session.ID)
		}
		payload = append(payload, '\n')
		item.stagedPath = filepath.Join(stageRoot, "new", relative)
		item.backupPath = filepath.Join(stageRoot, "backup", relative)
		if mkdirErr := os.MkdirAll(filepath.Dir(item.stagedPath), 0o755); mkdirErr != nil {
			return nil, errors.Wrap(mkdirErr, "creating staged archive directory")
		}
		if writeErr := writeSyncedFile(item.stagedPath, payload); writeErr != nil {
			return nil, writeErr
		}
		planned = append(planned, item)
	}

	sort.Slice(planned, func(i, j int) bool { return planned[i].destination < planned[j].destination })
	published := make([]int, 0, len(planned))
	rollback := func() {
		for i := len(published) - 1; i >= 0; i-- {
			item := planned[published[i]]
			_ = os.Remove(item.destination)
			if item.status == PublicationReplaced {
				_ = os.Rename(item.backupPath, item.destination)
			}
		}
	}
	for i := range planned {
		item := &planned[i]
		if item.status == PublicationUnchanged {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(item.destination), 0o755); err != nil {
			rollback()
			return nil, errors.Wrap(err, "creating archive publication directory")
		}
		if item.status == PublicationReplaced {
			if err := os.MkdirAll(filepath.Dir(item.backupPath), 0o755); err != nil {
				rollback()
				return nil, errors.Wrap(err, "creating archive backup directory")
			}
			if err := os.Rename(item.destination, item.backupPath); err != nil {
				rollback()
				return nil, errors.Wrap(err, "backing up replaced archive")
			}
		}
		if err := os.Rename(item.stagedPath, item.destination); err != nil {
			if item.status == PublicationReplaced {
				_ = os.Rename(item.backupPath, item.destination)
			}
			rollback()
			return nil, errors.Wrap(err, "publishing staged archive")
		}
		published = append(published, i)
	}

	results := make([]PublicationResult, 0, len(planned))
	for _, item := range planned {
		info, err := os.Stat(item.destination)
		if err != nil {
			return nil, errors.Wrap(err, "stating published archive")
		}
		results = append(results, PublicationResult{
			SessionID: item.session.ID,
			Status:    item.status,
			Entry:     sessionIndexEntry(item.session, item.period, item.destination, info.Size()),
		})
	}
	return results, nil
}

func sameSessionContent(left, right *Session) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftPayload) == string(rightPayload)
}

func sessionPeriod(session *Session) string {
	period := "unknown"
	if session.Timing.StartedAt != nil && len(*session.Timing.StartedAt) >= 7 {
		period = SanitizePeriod((*session.Timing.StartedAt)[:7])
	}
	return period
}

func writeSyncedFile(path string, payload []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return errors.Wrap(err, "creating staged archive")
	}
	if _, err := file.Write(payload); err != nil {
		_ = file.Close()
		return errors.Wrap(err, "writing staged archive")
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return errors.Wrap(err, "syncing staged archive")
	}
	if err := file.Close(); err != nil {
		return errors.Wrap(err, "closing staged archive")
	}
	return nil
}
