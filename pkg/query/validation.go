package query

import (
	"strings"

	"github.com/pkg/errors"
)

func ValidateReadOnlyQuery(sqlText string) error {
	normalized, err := NormalizeQueryForValidation(sqlText)
	if err != nil {
		return err
	}

	allowedPrefixes := []string{"select", "with", "explain", "describe", "show"}
	lower := strings.ToLower(normalized)
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return nil
		}
	}

	return errors.New("only read-only SELECT, WITH, EXPLAIN, DESCRIBE, and SHOW queries are allowed")
}

func NormalizeQueryForValidation(sqlText string) (string, error) {
	trimmed := strings.TrimSpace(sqlText)
	if trimmed == "" {
		return "", errors.New("sql is required")
	}

	for {
		switch {
		case strings.HasPrefix(trimmed, "--"):
			newlineIdx := strings.Index(trimmed, "\n")
			if newlineIdx == -1 {
				return "", errors.New("sql is required")
			}
			trimmed = strings.TrimSpace(trimmed[newlineIdx+1:])
		case strings.HasPrefix(trimmed, "/*"):
			endIdx := strings.Index(trimmed, "*/")
			if endIdx == -1 {
				return "", errors.New("unterminated SQL comment")
			}
			trimmed = strings.TrimSpace(trimmed[endIdx+2:])
		default:
			goto done
		}
	}

done:
	if trimmed == "" {
		return "", errors.New("sql is required")
	}

	withoutTrailingSemicolon := strings.TrimSpace(strings.TrimSuffix(trimmed, ";"))
	if withoutTrailingSemicolon == "" {
		return "", errors.New("sql is required")
	}
	if strings.Contains(withoutTrailingSemicolon, ";") {
		return "", errors.New("multiple SQL statements are not allowed")
	}

	return withoutTrailingSemicolon, nil
}
