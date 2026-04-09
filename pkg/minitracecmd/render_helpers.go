package minitracecmd

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func sqlEscape(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func sqlString(value any) (string, error) {
	s, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("sqlString expects string, got %T", value)
	}
	return "'" + sqlEscape(s) + "'", nil
}

func sqlLike(value any) (string, error) {
	s, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("sqlLike expects string, got %T", value)
	}
	return "'%" + sqlEscape(s) + "%'", nil
}

func sqlStringIn(values any) (string, error) {
	strs, err := toStringSlice(values)
	if err != nil {
		return "", err
	}
	if len(strs) == 0 {
		return "", nil
	}

	quoted := make([]string, 0, len(strs))
	for _, value := range strs {
		quoted = append(quoted, "'"+sqlEscape(value)+"'")
	}
	return strings.Join(quoted, ", "), nil
}

func sqlIntIn(values any) (string, error) {
	ints, err := toInt64Slice(values)
	if err != nil {
		return "", err
	}
	if len(ints) == 0 {
		return "", nil
	}

	parts := make([]string, 0, len(ints))
	for _, value := range ints {
		parts = append(parts, strconv.FormatInt(value, 10))
	}
	return strings.Join(parts, ", "), nil
}

func toStringSlice(values any) ([]string, error) {
	if values == nil {
		return nil, nil
	}

	switch typed := values.(type) {
	case []string:
		return append([]string(nil), typed...), nil
	case string:
		return []string{typed}, nil
	}

	rv := reflect.ValueOf(values)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, fmt.Errorf("expected string slice, got %T", values)
	}

	ret := make([]string, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		item := rv.Index(i).Interface()
		s, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("expected string slice element, got %T", item)
		}
		ret = append(ret, s)
	}
	return ret, nil
}

func toInt64Slice(values any) ([]int64, error) {
	if values == nil {
		return nil, nil
	}

	rv := reflect.ValueOf(values)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, fmt.Errorf("expected int slice, got %T", values)
	}

	ret := make([]int64, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		item := rv.Index(i)
		//exhaustive:ignore
		switch item.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			ret = append(ret, item.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			ret = append(ret, int64(item.Uint()))
		default:
			return nil, fmt.Errorf("expected int slice element, got %T", item.Interface())
		}
	}
	return ret, nil
}
