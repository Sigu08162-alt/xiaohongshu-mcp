package main

import "fmt"

// getString extracts a required string param. Returns error if missing or empty.
func getString(args map[string]any, key string) (string, error) {
	v, ok := args[key].(string)
	if !ok || v == "" {
		return "", fmt.Errorf("missing required param: %s", key)
	}
	return v, nil
}

// getStringOpt extracts an optional string param. Returns "" if missing.
func getStringOpt(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return v
}

// getBool extracts an optional bool param. Returns false if missing.
func getBool(args map[string]any, key string) bool {
	v, _ := args[key].(bool)
	return v
}

// getInt extracts an optional int param. Returns defaultVal if missing.
func getInt(args map[string]any, key string, defaultVal int) int {
	switch v := args[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return defaultVal
}

// getStringSlice extracts an optional []string param.
func getStringSlice(args map[string]any, key string) []string {
	raw, _ := args[key].([]any)
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		if s, ok := r.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
