package core_test

import (
	"bytes"
	"testing"

	"github.com/neko233-com/virtual-router-go/internal/core"
)

func TestFrameReadWrite(t *testing.T) {
	payload := []byte("hello frame")
	buf := bytes.NewBuffer(nil)

	if err := core.WriteFrame(buf, payload); err != nil {
		t.Fatalf("WriteFrame error: %v", err)
	}

	out, err := core.ReadFrame(buf)
	if err != nil {
		t.Fatalf("ReadFrame error: %v", err)
	}

	if !bytes.Equal(out, payload) {
		t.Fatalf("payload mismatch: got=%s", string(out))
	}
}
