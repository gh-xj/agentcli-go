package app

import "fmt"

func UsageError(message string) error {
	return fmt.Errorf("usage: %s", message)
}
