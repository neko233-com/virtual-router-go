package VirtualRouterServer

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
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
	logRotateCfg   = logRotationConfig{
		Dir:      "logs",
		BaseName: "router-center.log",
		MaxBytes: 20 * 1024 * 1024,
		MaxFiles: 5,
	}
)

type logRotationConfig struct {
	Dir      string
	BaseName string
	MaxBytes int64
	MaxFiles int
}

type rotatingFileWriter struct {
	mu         sync.Mutex
	dir        string
	baseName   string
	maxBytes   int64
	maxFiles   int
	file       *os.File
	currentLen int64
}

func (w *rotatingFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	w.currentLen = 0
	return err
}

func InstallProcessLogCapture(capacity int) {
	if capacity > 0 {
		globalLogs.capacity = capacity
	}

	logCaptureOnce.Do(func() {
		origin := os.Stdout
		writers := []io.Writer{origin, globalLogs}

		var rotateWriter io.Writer
		if rotateWriter, err := newRotatingFileWriter(logRotateCfg); err == nil {
			writers = append(writers, rotateWriter)
		} else {
			slog.Warn("日志文件轮转初始化失败，仅输出到控制台和内存", "error", err)
		}

		handler := slog.NewTextHandler(io.MultiWriter(writers...), &slog.HandlerOptions{Level: slog.LevelInfo})
		logger := slog.New(handler)
		slog.SetDefault(logger)

		if rotateWriter != nil {
			slog.Info("日志文件轮转已启用", "dir", logRotateCfg.Dir, "maxBytes", logRotateCfg.MaxBytes, "maxFiles", logRotateCfg.MaxFiles)
		}
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

func newRotatingFileWriter(cfg logRotationConfig) (*rotatingFileWriter, error) {
	if strings.TrimSpace(cfg.Dir) == "" {
		cfg.Dir = "logs"
	}
	if strings.TrimSpace(cfg.BaseName) == "" {
		cfg.BaseName = "router-center.log"
	}
	if cfg.MaxBytes <= 0 {
		cfg.MaxBytes = 20 * 1024 * 1024
	}
	if cfg.MaxFiles < 2 {
		cfg.MaxFiles = 5
	}

	if err := os.MkdirAll(cfg.Dir, 0755); err != nil {
		return nil, err
	}

	w := &rotatingFileWriter{
		dir:      cfg.Dir,
		baseName: cfg.BaseName,
		maxBytes: cfg.MaxBytes,
		maxFiles: cfg.MaxFiles,
	}
	if err := w.openCurrent(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *rotatingFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		if err := w.openCurrent(); err != nil {
			return 0, err
		}
	}

	if w.currentLen+int64(len(p)) > w.maxBytes {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	n, err := w.file.Write(p)
	w.currentLen += int64(n)
	return n, err
}

func (w *rotatingFileWriter) openCurrent() error {
	path := filepath.Join(w.dir, w.baseName)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return err
	}
	w.file = f
	w.currentLen = info.Size()
	return nil
}

func (w *rotatingFileWriter) rotate() error {
	if w.file != nil {
		_ = w.file.Close()
		w.file = nil
	}

	basePath := filepath.Join(w.dir, w.baseName)
	oldest := fmt.Sprintf("%s.%d", basePath, w.maxFiles-1)
	_ = os.Remove(oldest)

	for i := w.maxFiles - 2; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", basePath, i)
		dst := fmt.Sprintf("%s.%d", basePath, i+1)
		if _, err := os.Stat(src); err == nil {
			_ = os.Rename(src, dst)
		}
	}

	if _, err := os.Stat(basePath); err == nil {
		_ = os.Rename(basePath, basePath+".1")
	}

	return w.openCurrent()
}
