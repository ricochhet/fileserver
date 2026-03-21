package atomicutil

import (
	"encoding/json"
	"sync/atomic"
)

type Uintptr struct {
	atomic.Uintptr
}

func (u *Uintptr) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.Load())
}

func (u *Uintptr) UnmarshalJSON(data []byte) error {
	var v uintptr
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	u.Store(v)

	return nil
}
