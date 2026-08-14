package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/dittofleet/whatagain/internal/xdg"
)

const (
	lockTimeout  = 5 * time.Second
	lockPollWait = 20 * time.Millisecond
)

// lockPath is a file of its own rather than the store itself, because
// saving replaces the store through a rename and would strand a lock
// held on the old inode.
func lockPath() string {
	return filepath.Join(xdg.ConfigDir(xdg.App), ".todo.lock")
}

// acquire takes an exclusive advisory lock, waiting a short while for
// whoever holds it. Polling rather than blocking outright means a stuck
// process cannot hang the CLI indefinitely.
func acquire() (release func(), err error) {
	path := lockPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}

	deadline := time.Now().Add(lockTimeout)
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return func() {
				_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
				_ = f.Close()
			}, nil
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) {
			f.Close()
			return nil, fmt.Errorf("lock %s: %w", path, err)
		}
		if time.Now().After(deadline) {
			f.Close()
			return nil, fmt.Errorf("timed out waiting for another whatagain to finish writing (%s)", path)
		}
		time.Sleep(lockPollWait)
	}
}

// Update applies fn to the store and saves the result, holding a lock for
// the whole read-modify-write. Without it two commands racing (say, an
// agent per worktree) would each save from the copy they read and one
// would silently lose its item. Every mutating command goes through here,
// and fn returning an error leaves the store untouched.
func Update(fn func(*Store) error) error {
	release, err := acquire()
	if err != nil {
		return err
	}
	defer release()

	s, err := Load()
	if err != nil {
		return err
	}
	if err := fn(s); err != nil {
		return err
	}
	return s.Save()
}
