package dal

import (
	"bytes"
	"testing"
)

func TestLoggerImpl_Init(t *testing.T) {
	l := NewLogger()

	t.Run("verbose", func(t *testing.T) {
		var buf bytes.Buffer
		l.Init(true, &buf)
	})

	t.Run("quiet", func(t *testing.T) {
		var buf bytes.Buffer
		l.Init(false, &buf)
	})
}
