package atomicutil

import (
	"encoding/json"
	"sync/atomic"
)

type Int32 struct {
	atomic.Int32
}

func (i *Int32) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.Load())
}

func (i *Int32) UnmarshalJSON(data []byte) error {
	var v int32
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	i.Store(v)

	return nil
}
