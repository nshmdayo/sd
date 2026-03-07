package selector

import (
	"fmt"

	"github.com/ktr0731/go-fuzzyfinder"
)

// InternalSelector uses go-fuzzyfinder as a pure-Go fallback UI.
type InternalSelector struct{}

func (s *InternalSelector) Select(candidates []string, _ string) (string, error) {
	if len(candidates) == 0 {
		return "", fmt.Errorf("no candidates")
	}

	idx, err := fuzzyfinder.Find(candidates, func(i int) string {
		return candidates[i]
	})
	if err != nil {
		if err == fuzzyfinder.ErrAbort {
			return "", ErrCancelled
		}
		return "", err
	}
	return candidates[idx], nil
}
