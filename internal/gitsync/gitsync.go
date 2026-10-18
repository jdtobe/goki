// Package gitsync records wiki changes as git commits using the git CLI.
package gitsync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Committer commits changes in a working tree. The zero value is unusable;
// construct one with New.
type Committer struct {
	dir    string
	name   string
	email  string
	mu     sync.Mutex
	runGit func(ctx context.Context, args ...string) (string, error)
}

// New prepares a Committer for the repository containing dir, initializing a
// new repository there if none exists.
func New(dir, authorName, authorEmail string) (*Committer, bool, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return nil, false, errors.New("git auto-commit needs git on your PATH")
	}
	c := &Committer{dir: dir, name: authorName, email: authorEmail}
	c.runGit = c.exec
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	initialized := false
	if _, err := c.runGit(ctx, "rev-parse", "--is-inside-work-tree"); err != nil {
		if _, err := c.runGit(ctx, "init", "--quiet"); err != nil {
			return nil, false, err
		}
		initialized = true
	}
	return c, initialized, nil
}

func (c *Committer) exec(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = c.dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(out.String()))
	}
	return out.String(), nil
}

// Commit stages the given paths (relative to the wiki root, including
// deleted ones) and commits them with message. It is a no-op when nothing
// changed.
func (c *Committer) Commit(message string, paths ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	add := append([]string{"add", "--all", "--"}, paths...)
	if _, err := c.runGit(ctx, add...); err != nil {
		return err
	}
	diff := append([]string{"diff", "--cached", "--quiet", "--"}, paths...)
	if _, err := c.runGit(ctx, diff...); err == nil {
		return nil // nothing staged
	}
	args := []string{}
	if c.name != "" {
		args = append(args, "-c", "user.name="+c.name)
	}
	if c.email != "" {
		args = append(args, "-c", "user.email="+c.email)
	}
	args = append(args, "commit", "--quiet", "-m", message, "--")
	args = append(args, paths...)
	_, err := c.runGit(ctx, args...)
	return err
}
