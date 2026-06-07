package minitracedb

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func OpenSQLiteMemory(ctx context.Context, prefix string) (*sql.DB, error) {
	name, err := uniqueMemoryName(prefix)
	if err != nil {
		return nil, err
	}
	dsn := "file:" + name + "?mode=memory&cache=shared"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite memory: %w", err)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys=ON;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable sqlite foreign keys: %w", err)
	}
	return db, nil
}

func uniqueMemoryName(prefix string) (string, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "minitrace"
	}
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("randomize sqlite memory name: %w", err)
	}
	return prefix + "_" + hex.EncodeToString(b), nil
}
