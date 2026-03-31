package loader

import "reflect"

func normalizeStructuredValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		normalized := make(map[string]any, len(typed))
		for key, inner := range typed {
			normalized[key] = normalizeStructuredValue(inner)
		}
		return normalized
	case map[any]any:
		normalized := make(map[string]any, len(typed))
		for key, inner := range typed {
			normalized[stringifyJSONValue(key)] = normalizeStructuredValue(inner)
		}
		return normalized
	case []any:
		normalized := make([]any, len(typed))
		for i, inner := range typed {
			normalized[i] = normalizeStructuredValue(inner)
		}
		return normalized
	default:
		return normalizeStructuredReflect(value)
	}
}

func normalizeStructuredReflect(value any) any {
	if value == nil {
		return nil
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Map:
		normalized := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			normalized[stringifyJSONValue(iter.Key().Interface())] = normalizeStructuredValue(iter.Value().Interface())
		}
		return normalized
	case reflect.Slice, reflect.Array:
		normalized := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			normalized[i] = normalizeStructuredValue(rv.Index(i).Interface())
		}
		return normalized
	default:
		return value
	}
}

func structuredRecords(value any) []any {
	normalized := normalizeStructuredValue(value)
	if records, ok := normalized.([]any); ok {
		return records
	}
	if record, ok := normalized.(map[string]any); ok && len(record) == 1 {
		for _, inner := range record {
			if rows, ok := inner.([]any); ok && sliceContainsOnlyMaps(rows) {
				return rows
			}
		}
	}
	return []any{normalized}
}

func sliceContainsOnlyMaps(values []any) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if _, ok := value.(map[string]any); !ok {
			return false
		}
	}
	return true
}
