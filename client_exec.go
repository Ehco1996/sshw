package sshw

import (
	"bytes"
	"context"
	"time"

	"golang.org/x/crypto/ssh"
)

// RunResult is the outcome of a non-interactive RunCommand against a single host.
//
// ExitCode and Err encode different failure modes:
//   - Err != nil: the command did not run normally (dial / auth / session
//     setup failed, or ctx was cancelled). ExitCode is -1 in this case.
//   - Err == nil and ExitCode != 0: the command ran and exited non-zero;
//     Stdout/Stderr hold whatever it produced before exiting.
//   - Err == nil and ExitCode == 0: success.
type RunResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	Err      error
	Duration time.Duration
}

// Runner executes a single command against a host configured by a Node.
type Runner interface {
	RunCommand(ctx context.Context, cmd string) RunResult
}

// NewRunner builds a Runner that uses the same auth + jump-host plumbing as
// the interactive Login() path.
func NewRunner(node *Node) Runner {
	return genSSHConfig(node)
}

func (c *defaultClient) RunCommand(ctx context.Context, cmd string) RunResult {
	defer c.close()
	start := time.Now()

	client, err := c.dial(false)
	if err != nil {
		return RunResult{ExitCode: -1, Err: err, Duration: time.Since(start)}
	}
	defer client.Close()

	sess, err := client.NewSession()
	if err != nil {
		return RunResult{ExitCode: -1, Err: err, Duration: time.Since(start)}
	}
	defer sess.Close()

	var stdout, stderr bytes.Buffer
	sess.Stdout = &stdout
	sess.Stderr = &stderr

	done := make(chan error, 1)
	go func() { done <- sess.Run(cmd) }()

	select {
	case <-ctx.Done():
		// Best-effort kill the remote process; ignore errors since the session
		// will be torn down by the deferred Close anyway.
		_ = sess.Signal(ssh.SIGKILL)
		_ = sess.Close()
		return RunResult{
			Stdout:   stdout.Bytes(),
			Stderr:   stderr.Bytes(),
			ExitCode: -1,
			Err:      ctx.Err(),
			Duration: time.Since(start),
		}
	case runErr := <-done:
		res := RunResult{
			Stdout:   stdout.Bytes(),
			Stderr:   stderr.Bytes(),
			Duration: time.Since(start),
		}
		if runErr == nil {
			res.ExitCode = 0
			return res
		}
		if ee, ok := runErr.(*ssh.ExitError); ok {
			res.ExitCode = ee.ExitStatus()
			return res
		}
		res.ExitCode = -1
		res.Err = runErr
		return res
	}
}
