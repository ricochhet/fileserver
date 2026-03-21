package atomicutil

import (
	"encoding/json"
	"sync/atomic"
)

type Bool struct {
	atomic.Bool
}

func (b *Bool) MarshalJSON() ([]byte, error) {
	if b.Load() {
		return []byte("true"), nil
	}

	return []byte("false"), nil
}

func (b *Bool) UnmarshalJSON(data []byte) error {
	var v bool
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	b.Store(v)

	return nil
}
