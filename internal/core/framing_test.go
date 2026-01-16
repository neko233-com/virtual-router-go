package core

import (
	"bytes"
	"testing"
)

func TestFrameReadWrite(t *testing.T) {
	payload := []byte("hello frame")
	buf := bytes.NewBuffer(nil)

	if err := WriteFrame(buf, payload); err != nil {
		t.Fatalf("WriteFrame error: %v", err)
	}

	out, err := ReadFrame(buf)
	if err != nil {
		t.Fatalf("ReadFrame error: %v", err)
	}

	if !bytes.Equal(out, payload) {
		t.Fatalf("payload mismatch: got=%s", string(out))
	}
}
