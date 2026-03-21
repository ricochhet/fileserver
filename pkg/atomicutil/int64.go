package atomicutil

import (
	"encoding/json"
	"sync/atomic"
)

type Int64 struct {
	atomic.Int64
}

func (i *Int64) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.Load())
}

func (i *Int64) UnmarshalJSON(data []byte) error {
	var v int64
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	i.Store(v)

	return nil
}
