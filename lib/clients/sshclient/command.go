package sshclient

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"golang.org/x/crypto/ssh"
)

// ExecResult holds the output of a remote command execution.
type ExecResult struct {
	// Stdout is the standard output captured from the command.
	Stdout []byte
	// Stderr is the standard error captured from the command.
	Stderr []byte
}

// Exec executes a command on the remote host and returns the captured
// stdout and stderr. If the command exits with a non-zero status, an
// error is returned but ExecResult still contains whatever output was
// captured before the failure.
func (c *Client) Exec(ctx context.Context, cmd string) (*ExecResult, error) {
	session, err := c.newSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	// Use a channel to detect command completion so we can respect ctx.
	done := make(chan error, 1)
	go func() {
		done <- session.Run(cmd)
	}()

	select {
	case <-ctx.Done():
		// Signal the remote process. Best-effort; the session close
		// in the defer will clean up regardless.
		_ = session.Signal(ssh.SIGKILL)
		return &ExecResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, ctx.Err()
	case err := <-done:
		return &ExecResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, err
	}
}

// ExecWithSudo executes a command on the remote host with sudo
// privileges. If password is non-empty, it is fed to sudo via stdin
// using `sudo -S -p ”` (silent prompt, read password from stdin).
// If password is empty, plain `sudo` is used (suitable for NOPASSWD
// sudoers entries).
func (c *Client) ExecWithSudo(ctx context.Context, cmd string, password string) (*ExecResult, error) {
	session, err := c.newSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	var sudoCmd string
	if password != "" {
		// -S: read password from stdin
		// -p '': suppress the password prompt to avoid it appearing in output
		sudoCmd = fmt.Sprintf("sudo -S -p '' %s", cmd)

		stdin, err := session.StdinPipe()
		if err != nil {
			return nil, fmt.Errorf("sshclient: failed to create stdin pipe: %w", err)
		}

		// Write the password followed by a newline so sudo can read it.
		go func() {
			defer stdin.Close()
			_, _ = io.WriteString(stdin, password+"\n")
		}()
	} else {
		sudoCmd = fmt.Sprintf("sudo %s", cmd)
	}

	done := make(chan error, 1)
	go func() {
		done <- session.Run(sudoCmd)
	}()

	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGKILL)
		return &ExecResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, ctx.Err()
	case err := <-done:
		return &ExecResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, err
	}
}

// ExecStream executes a command on the remote host, streaming stdout
// and stderr to the provided writers in real time. It blocks until the
// command completes or the context is cancelled.
func (c *Client) ExecStream(ctx context.Context, cmd string, stdoutW, stderrW io.Writer) error {
	session, err := c.newSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdout = stdoutW
	session.Stderr = stderrW

	done := make(chan error, 1)
	go func() {
		done <- session.Run(cmd)
	}()

	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGKILL)
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// newSession creates a new SSH session, returning an error if the
// client is not connected.
func (c *Client) newSession() (*ssh.Session, error) {
	c.mu.Lock()
	sshClient := c.client
	c.mu.Unlock()

	if sshClient == nil {
		return nil, fmt.Errorf("sshclient: not connected")
	}

	session, err := sshClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("sshclient: failed to create session: %w", err)
	}
	return session, nil
}
