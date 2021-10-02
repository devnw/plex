package plex

import (
	"errors"
	"fmt"
)

func recoverErr(r interface{}) error {
	switch v := r.(type) {
	case nil:
		return nil
	case string:
		return errors.New(v)
	case error:
		return v
	default:
		// Fallback err (per specs, error strings
		// should be lowercase w/o punctuation
		return fmt.Errorf("panic: %v", r)
	}
}
