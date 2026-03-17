package virtual_router_server_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	server "github.com/neko233-com/virtual-router-go/internal/VirtualRouterServer"
)

func TestLogCapture_WriteAndGetRecent(t *testing.T) {
	capture := server.NewLogCaptureTestHelper(3)

	_, _ = capture.Write([]byte("line-a\nline-b\n"))
	_, _ = capture.Write([]byte("line-c\nline-d\n"))

	all := capture.GetRecent(10)
	if len(all) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(all))
	}

	if !strings.HasSuffix(all[0], "line-b") {
		t.Fatalf("expected first line to end with line-b, got %q", all[0])
	}
	if !strings.HasSuffix(all[1], "line-c") {
		t.Fatalf("expected second line to end with line-c, got %q", all[1])
	}
	if !strings.HasSuffix(all[2], "line-d") {
		t.Fatalf("expected third line to end with line-d, got %q", all[2])
	}

	lastTwo := capture.GetRecent(2)
	if len(lastTwo) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lastTwo))
	}
	if !strings.HasSuffix(lastTwo[0], "line-c") || !strings.HasSuffix(lastTwo[1], "line-d") {
		t.Fatalf("unexpected last two lines: %#v", lastTwo)
	}
}

func TestRotatingFileWriter_RotatesFiles(t *testing.T) {
	dir := t.TempDir()
	w, err := server.NewRotatingFileWriterForTest(dir, "test.log", 30, 3)
	if err != nil {
		t.Fatalf("new rotating writer error: %v", err)
	}
	t.Cleanup(func() {
		_ = w.Close()
	})

	for i := 0; i < 10; i++ {
		_, err = w.Write([]byte("line-0123456789\n"))
		if err != nil {
			t.Fatalf("write error: %v", err)
		}
	}

	if _, err := os.Stat(filepath.Join(dir, "test.log")); err != nil {
		t.Fatalf("expected current log file exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "test.log.1")); err != nil {
		t.Fatalf("expected rotated log .1 exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "test.log.2")); err != nil {
		t.Fatalf("expected rotated log .2 exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "test.log.3")); err == nil {
		t.Fatalf("expected .3 not exists because maxFiles=3")
	}
}
