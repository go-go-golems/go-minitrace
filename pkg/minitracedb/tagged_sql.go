package minitracedb

import (
	"fmt"
	"regexp"
	"strings"
)

var namedSQLParameter = regexp.MustCompile(`[\x3a@\$][a-zA-Z0-9_\x80-\xFF]`)

// TaggedSQL inserts positional placeholders and checks their lexical positions.
// SQLite preparation and authorization still run in QueryRunner.
func TaggedSQL(parts []string) (string, error) {
	if len(parts) == 0 {
		return "", fmt.Errorf("sql requires a tagged template")
	}
	var sql strings.Builder
	offsets := make([]int, 0, len(parts)-1)
	for i, part := range parts {
		if i > 0 {
			offsets = append(offsets, sql.Len())
			sql.WriteByte('?')
		}
		sql.WriteString(part)
		if sql.Len() > 1<<20 {
			return "", fmt.Errorf("tagged SQL exceeds 1 MiB")
		}
	}
	statement := sql.String()
	masked := stripSQLLiteralsAndComments(statement)
	for _, index := range offsets {
		if masked[index] != '?' ||
			(index+1 < len(masked) && masked[index+1] >= '0' && masked[index+1] <= '9') {
			return "", fmt.Errorf("sql interpolation must occupy a value position outside quotes and comments")
		}
	}
	if strings.Count(masked, "?") != len(offsets) || namedSQLParameter.MatchString(masked) {
		return "", fmt.Errorf("sql bindings must come from template interpolations")
	}
	return validateReadOnlySQLiteQuery(statement, QueryOptions{})
}
