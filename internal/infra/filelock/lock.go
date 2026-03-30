package filelock

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
)

var ErrLocked = errors.New("file lock already held")

var (
	locksMu sync.Mutex
	locks   = map[string]struct{}{}
)

type Locker struct {
	path   string
	locked bool
}

func New(path string) *Locker {
	return &Locker{path: path}
}

func (l *Locker) Lock() error {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}

	locksMu.Lock()
	defer locksMu.Unlock()
	if _, ok := locks[l.path]; ok {
		return ErrLocked
	}
	locks[l.path] = struct{}{}
	l.locked = true
	return nil
}

func (l *Locker) Unlock() error {
	locksMu.Lock()
	defer locksMu.Unlock()
	if l.locked {
		delete(locks, l.path)
		l.locked = false
	}
	return nil
}
