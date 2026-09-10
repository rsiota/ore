package git

import "errors"

// IsNotRepository reports whether err wraps ErrNotRepository.
func IsNotRepository(err error) bool {
	return errors.Is(err, ErrNotRepository)
}
