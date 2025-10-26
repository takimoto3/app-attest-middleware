package logger_test

import (
	"bytes"
	"log"
	"strings"
	"sync"
	"testing"

	"github.com/takimoto3/app-attest-middleware/logger"
)

func TestStdLogger_LevelPrefixes(t *testing.T) {
	var buf bytes.Buffer
	l := &logger.StdLogger{
		Logger: log.New(&buf, "", 0),
	}

	// test each log level
	l.Debugf("debug message")
	l.Infof("info message")
	l.Warningf("warn message")
	l.Errorf("error message")
	l.Criticalf("critical message")

	output := buf.String()
	tests := []struct {
		level   string
		message string
	}{
		{"[Debug]", "debug message"},
		{"[Info]", "info message"},
		{"[Warning]", "warn message"},
		{"[Error]", "error message"},
		{"[Critical]", "critical message"},
	}

	for _, tt := range tests {
		if !strings.Contains(output, tt.level+" "+tt.message) {
			t.Errorf("expected output to contain %q, got %q", tt.level+" "+tt.message, output)
		}
	}
}

func TestStdLogger_Concurrent(t *testing.T) {
	var buf bytes.Buffer
	l := &logger.StdLogger{
		Logger: log.New(&buf, "", 0),
	}
	const n = 100

	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			l.Infof("message %d", i)
		}(i)
	}
	wg.Wait()

	output := buf.String()
	// Check a few expected messages (prefix exists)
	for i := 0; i < 10; i++ {
		if !strings.Contains(output, "[Info]") {
			t.Errorf("expected output to contain [Info], got %q", output)
		}
	}
}
