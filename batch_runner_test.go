package sshw

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type dummyRunner struct {
	runFunc func(ctx context.Context, cmd string) RunResult
}

func (d *dummyRunner) RunCommand(ctx context.Context, cmd string) RunResult {
	return d.runFunc(ctx, cmd)
}

func TestRunBatch(t *testing.T) {
	targets := []*Node{
		{Name: "host-1", Host: "10.0.0.1"},
		{Name: "host-2", Host: "10.0.0.2"},
		{Name: "host-3", Host: "10.0.0.3"},
	}

	t.Run("successful batch", func(t *testing.T) {
		opts := BatchOptions{
			NoAudit: true,
			RunnerFunc: func(n *Node) Runner {
				return &dummyRunner{
					runFunc: func(ctx context.Context, cmd string) RunResult {
						return RunResult{
							Stdout:   []byte("ok from " + n.Name),
							ExitCode: 0,
							Duration: 10 * time.Millisecond,
						}
					},
				}
			},
		}

		summary, err := RunBatch(context.Background(), targets, "uptime", opts)
		if err != nil {
			t.Fatalf("RunBatch failed: %v", err)
		}

		if summary.Total != 3 || summary.Ok != 3 || summary.Failed != 0 {
			t.Errorf("unexpected counts: total=%d, ok=%d, failed=%d", summary.Total, summary.Ok, summary.Failed)
		}
		if len(summary.Results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(summary.Results))
		}
		for i, r := range summary.Results {
			if r.Name != targets[i].Name {
				t.Errorf("ordering mismatch at %d: got %s, want %s", i, r.Name, targets[i].Name)
			}
			if r.Stdout != "ok from "+targets[i].Name {
				t.Errorf("unexpected stdout: %s", r.Stdout)
			}
		}
	})

	t.Run("handles failures", func(t *testing.T) {
		opts := BatchOptions{
			NoAudit: true,
			RunnerFunc: func(n *Node) Runner {
				return &dummyRunner{
					runFunc: func(ctx context.Context, cmd string) RunResult {
						if n.Name == "host-2" {
							return RunResult{
								ExitCode: 1,
								Stderr:   []byte("permission denied"),
								Duration: 5 * time.Millisecond,
							}
						}
						return RunResult{
							ExitCode: 0,
							Stdout:   []byte("success"),
							Duration: 5 * time.Millisecond,
						}
					},
				}
			},
		}

		summary, err := RunBatch(context.Background(), targets, "systemctl restart ehco", opts)
		if err != nil {
			t.Fatalf("RunBatch failed: %v", err)
		}
		if summary.Total != 3 || summary.Ok != 2 || summary.Failed != 1 {
			t.Errorf("unexpected counts: total=%d, ok=%d, failed=%d", summary.Total, summary.Ok, summary.Failed)
		}
	})

	t.Run("blocks dangerous command without force", func(t *testing.T) {
		opts := BatchOptions{
			NoAudit:     true,
			ForceDanger: false,
		}
		_, err := RunBatch(context.Background(), targets, "rm -rf /", opts)
		if err == nil {
			t.Fatal("expected dangerous command error, got nil")
		}
		if !errors.Is(err, ErrDangerousCommand) {
			t.Errorf("expected ErrDangerousCommand, got: %v", err)
		}
	})

	t.Run("allows dangerous command with force", func(t *testing.T) {
		opts := BatchOptions{
			NoAudit:     true,
			ForceDanger: true,
			RunnerFunc: func(n *Node) Runner {
				return &dummyRunner{
					runFunc: func(ctx context.Context, cmd string) RunResult {
						return RunResult{ExitCode: 0}
					},
				}
			},
		}
		summary, err := RunBatch(context.Background(), targets, "reboot", opts)
		if err != nil {
			t.Fatalf("expected command to run with ForceDanger, got error: %v", err)
		}
		if summary.Ok != 3 {
			t.Errorf("expected 3 ok, got %d", summary.Ok)
		}
	})

	t.Run("respects concurrency limit", func(t *testing.T) {
		var inFlight int64
		var maxInFlight int64

		opts := BatchOptions{
			Concurrency: 2,
			NoAudit:     true,
			RunnerFunc: func(n *Node) Runner {
				return &dummyRunner{
					runFunc: func(ctx context.Context, cmd string) RunResult {
						curr := atomic.AddInt64(&inFlight, 1)
						for {
							oldMax := atomic.LoadInt64(&maxInFlight)
							if curr <= oldMax || atomic.CompareAndSwapInt64(&maxInFlight, oldMax, curr) {
								break
							}
						}
						time.Sleep(30 * time.Millisecond)
						atomic.AddInt64(&inFlight, -1)
						return RunResult{ExitCode: 0}
					},
				}
			},
		}

		fiveTargets := []*Node{
			{Name: "n1", Host: "10.0.0.1"},
			{Name: "n2", Host: "10.0.0.2"},
			{Name: "n3", Host: "10.0.0.3"},
			{Name: "n4", Host: "10.0.0.4"},
			{Name: "n5", Host: "10.0.0.5"},
		}

		_, err := RunBatch(context.Background(), fiveTargets, "test", opts)
		if err != nil {
			t.Fatalf("RunBatch failed: %v", err)
		}
		if max := atomic.LoadInt64(&maxInFlight); max > 2 {
			t.Errorf("concurrency exceeded limit 2: got %d", max)
		}
	})
}
