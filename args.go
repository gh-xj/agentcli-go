package gokit

import (
	"strings"

	"github.com/rs/zerolog/log"
)

// ParseArgs parses --key value style arguments into a map.
// Flags without a value (or followed by another flag) get value "true".
func ParseArgs(args []string) map[string]string {
	result := make(map[string]string)
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") {
			key := strings.TrimPrefix(args[i], "--")
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				result[key] = args[i+1]
				i++
			} else {
				result[key] = "true"
			}
		}
	}
	return result
}

// RequireArg gets a required argument or calls log.Fatal.
func RequireArg(args map[string]string, key, usage string) string {
	if val, ok := args[key]; ok && val != "" {
		return val
	}
	log.Fatal().Str("flag", "--"+key).Str("usage", usage).Msg("required flag missing")
	return ""
}

// GetArg gets an optional argument with a default value.
func GetArg(args map[string]string, key, defaultVal string) string {
	if val, ok := args[key]; ok && val != "" {
		return val
	}
	return defaultVal
}

// HasFlag checks if a boolean flag is set.
func HasFlag(args map[string]string, key string) bool {
	val, ok := args[key]
	return ok && val == "true"
}
