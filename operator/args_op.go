package operator

import (
	"fmt"
	"strings"
)

// ArgsOperatorImpl implements ArgsOperator.
type ArgsOperatorImpl struct{}

// NewArgsOperator returns a new ArgsOperatorImpl.
func NewArgsOperator() *ArgsOperatorImpl { return &ArgsOperatorImpl{} }

// Parse parses --key value style arguments into a map.
// Flags without a value (or followed by another flag) get value "true".
func (a *ArgsOperatorImpl) Parse(args []string) map[string]string {
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

// Require returns the value for key or an error if missing.
func (a *ArgsOperatorImpl) Require(args map[string]string, key, usage string) (string, error) {
	if val, ok := args[key]; ok && val != "" {
		return val, nil
	}
	return "", fmt.Errorf("required flag missing: --%s (%s)", key, usage)
}

// Get returns the value for key or defaultVal if missing.
func (a *ArgsOperatorImpl) Get(args map[string]string, key, defaultVal string) string {
	if val, ok := args[key]; ok && val != "" {
		return val
	}
	return defaultVal
}

// HasFlag checks if a boolean flag is set to "true".
func (a *ArgsOperatorImpl) HasFlag(args map[string]string, key string) bool {
	val, ok := args[key]
	return ok && val == "true"
}
