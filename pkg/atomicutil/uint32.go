package atomicutil

import (
	"encoding/json"
	"sync/atomic"
)

type Uint32 struct {
	atomic.Uint32
}

func (u *Uint32) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.Load())
}

func (u *Uint32) UnmarshalJSON(data []byte) error {
	var v uint32
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	u.Store(v)

	return nil
}
