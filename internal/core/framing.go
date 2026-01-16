package core

import (
	"encoding/binary"
	"errors"
	"io"
)

const MaxFrameSize = 10 * 1024 * 1024

func EncodeFrame(payload []byte) []byte {
	out := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(out[:4], uint32(len(payload)))
	copy(out[4:], payload)
	return out
}

func ReadFrame(r io.Reader) ([]byte, error) {
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return nil, err
	}
	length := int(binary.BigEndian.Uint32(lenBuf))
	if length < 0 || length > MaxFrameSize {
		return nil, errors.New("frame length out of range")
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func WriteFrame(w io.Writer, payload []byte) error {
	_, err := w.Write(EncodeFrame(payload))
	return err
}
