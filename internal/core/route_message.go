package core

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

type RouteMessage struct {
	FromRouteId string
	ToRouteId   string
	MessageType *RouteMessageType
	Data        *string
}

// EncodePayload 编码内部负载（不含长度前缀）
func (m *RouteMessage) EncodePayload() ([]byte, error) {
	if m == nil {
		return nil, errors.New("RouteMessage is nil")
	}
	fromBytes := []byte(m.FromRouteId)
	toBytes := []byte(m.ToRouteId)
	var dataBytes []byte
	if m.Data != nil {
		dataBytes = []byte(*m.Data)
	}

	payloadLen := 4 + len(fromBytes) + 4 + len(toBytes) + 4 + 4 + len(dataBytes)
	buf := bytes.NewBuffer(make([]byte, 0, payloadLen))

	if err := writeInt32(buf, int32(len(fromBytes))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(fromBytes); err != nil {
		return nil, err
	}

	if err := writeInt32(buf, int32(len(toBytes))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(toBytes); err != nil {
		return nil, err
	}

	if m.MessageType == nil {
		if err := writeInt32(buf, -1); err != nil {
			return nil, err
		}
	} else {
		if err := writeInt32(buf, int32(*m.MessageType)); err != nil {
			return nil, err
		}
	}

	if m.Data == nil {
		if err := writeInt32(buf, -1); err != nil {
			return nil, err
		}
	} else {
		if err := writeInt32(buf, int32(len(dataBytes))); err != nil {
			return nil, err
		}
		if _, err := buf.Write(dataBytes); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func DecodeRouteMessagePayload(payload []byte) (*RouteMessage, error) {
	if len(payload) == 0 {
		return nil, errors.New("empty payload")
	}
	reader := bytes.NewReader(payload)

	// 兼容性修复：检查可能存在的额外长度前缀
	if reader.Len() >= 4 {
		peek, _ := reader.Seek(0, io.SeekCurrent)
		var possibleLen int32
		if err := binary.Read(reader, binary.BigEndian, &possibleLen); err == nil {
			if int(possibleLen) == reader.Len()-4 {
				// 跳过额外长度前缀
			} else {
				// 回退
				_, _ = reader.Seek(peek, io.SeekStart)
			}
		} else {
			_, _ = reader.Seek(peek, io.SeekStart)
		}
	}

	fromLen, err := readInt32(reader)
	if err != nil || fromLen < 0 {
		return nil, errors.New("invalid fromRouteId length")
	}
	fromBytes := make([]byte, fromLen)
	if _, err := io.ReadFull(reader, fromBytes); err != nil {
		return nil, err
	}

	toLen, err := readInt32(reader)
	if err != nil || toLen < 0 {
		return nil, errors.New("invalid toRouteId length")
	}
	toBytes := make([]byte, toLen)
	if _, err := io.ReadFull(reader, toBytes); err != nil {
		return nil, err
	}

	ordinal, err := readInt32(reader)
	if err != nil {
		return nil, err
	}
	var msgType *RouteMessageType
	if ordinal >= 0 {
		if mt, ok := RouteMessageTypeFromOrdinal(ordinal); ok {
			msgType = mt
		}
	}

	dataLen, err := readInt32(reader)
	if err != nil {
		return nil, err
	}
	var data *string
	if dataLen >= 0 {
		dBytes := make([]byte, dataLen)
		if _, err := io.ReadFull(reader, dBytes); err != nil {
			return nil, err
		}
		dStr := string(dBytes)
		data = &dStr
	}

	return &RouteMessage{
		FromRouteId: string(fromBytes),
		ToRouteId:   string(toBytes),
		MessageType: msgType,
		Data:        data,
	}, nil
}

func writeInt32(w io.Writer, v int32) error {
	return binary.Write(w, binary.BigEndian, v)
}

func readInt32(r io.Reader) (int32, error) {
	var v int32
	err := binary.Read(r, binary.BigEndian, &v)
	return v, err
}
