package rpc

import (
	"encoding/json"
	"strconv"
)

func int64ToString(v int64) string {
	return strconv.FormatInt(v, 10)
}

func UnmarshalArg[T any](raw json.RawMessage) (T, error) {
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		return v, err
	}
	return v, nil
}
