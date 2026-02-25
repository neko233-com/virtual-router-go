package VirtualRouterServer

import (
	"strings"
	"testing"
)

func TestLogCapture_WriteAndGetRecent(t *testing.T) {
	capture := &logCapture{capacity: 3, lines: make([]string, 0, 3)}

	_, _ = capture.Write([]byte("line-a\nline-b\n"))
	_, _ = capture.Write([]byte("line-c\nline-d\n"))

	all := capture.getRecent(10)
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

	lastTwo := capture.getRecent(2)
	if len(lastTwo) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lastTwo))
	}
	if !strings.HasSuffix(lastTwo[0], "line-c") || !strings.HasSuffix(lastTwo[1], "line-d") {
		t.Fatalf("unexpected last two lines: %#v", lastTwo)
	}
}
