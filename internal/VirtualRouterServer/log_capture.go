package VirtualRouterServer

import (
	"io"
	"log"
	"strings"
	"sync"
	"time"
)

type logCapture struct {
	mu       sync.Mutex
	capacity int
	lines    []string
}

var (
	logCaptureOnce sync.Once
	globalLogs     = &logCapture{capacity: 500, lines: make([]string, 0, 500)}
)

func InstallProcessLogCapture(capacity int) {
	if capacity > 0 {
		globalLogs.capacity = capacity
	}

	logCaptureOnce.Do(func() {
		origin := log.Writer()
		log.SetOutput(io.MultiWriter(origin, globalLogs))
	})
}

func GetRecentProcessLogs(limit int) []string {
	return globalLogs.getRecent(limit)
}

func (c *logCapture) Write(p []byte) (n int, err error) {
	text := string(p)
	segments := strings.Split(text, "\n")

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, seg := range segments {
		line := strings.TrimSpace(seg)
		if line == "" {
			continue
		}
		c.lines = append(c.lines, time.Now().Format("2006-01-02 15:04:05")+" "+line)
		if len(c.lines) > c.capacity {
			over := len(c.lines) - c.capacity
			c.lines = c.lines[over:]
		}
	}

	return len(p), nil
}

func (c *logCapture) getRecent(limit int) []string {
	if limit <= 0 {
		limit = 100
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	total := len(c.lines)
	if total == 0 {
		return []string{}
	}
	if limit > total {
		limit = total
	}
	start := total - limit
	out := make([]string, 0, limit)
	out = append(out, c.lines[start:]...)
	return out
}
