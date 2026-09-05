package sshw

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

var ErrDangerousCommand = errors.New("dangerous command detected")

const (
	DefaultBatchConcurrency = 8
	DefaultBatchTimeout     = 30 * time.Second
)

// BatchOptions configures batch execution across nodes.
type BatchOptions struct {
	Concurrency int                     // Max concurrent connections; defaults to DefaultBatchConcurrency
	Timeout     time.Duration           // Per-host execution timeout; defaults to DefaultBatchTimeout
	ForceDanger bool                    // Allow commands flagged as dangerous
	NoAudit     bool                    // Skip writing to audit log (e.g. in tests)
	RunnerFunc  func(node *Node) Runner // Factory for Runner; defaults to NewRunner
}

// TargetExecutionResult represents the execution outcome on a single host.
type TargetExecutionResult struct {
	Node       *Node  `json:"-"`
	Name       string `json:"name"`
	Alias      string `json:"alias,omitempty"`
	Host       string `json:"host"`
	ExitCode   int    `json:"exit_code"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	DurationMs int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

// BatchSummary summarizes the result of running a command across multiple targets.
type BatchSummary struct {
	Cmd        string                  `json:"cmd"`
	Started    time.Time               `json:"started"`
	Finished   time.Time               `json:"finished"`
	DurationMs int64                   `json:"duration_ms"`
	Total      int                     `json:"total"`
	Ok         int                     `json:"ok"`
	Failed     int                     `json:"failed"`
	LogDir     string                  `json:"log_dir,omitempty"`
	Results    []TargetExecutionResult `json:"results"`
}

// RunBatch executes cmd concurrently across targets with concurrency and timeout limits.
func RunBatch(ctx context.Context, targets []*Node, cmd string, opts BatchOptions) (*BatchSummary, error) {
	if len(targets) == 0 {
		return nil, errors.New("no targets specified")
	}

	if match, ok := DangerousMatch(cmd); ok && !opts.ForceDanger {
		return nil, fmt.Errorf("%w: matched %q (use --force to override)", ErrDangerousCommand, match)
	}

	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = DefaultBatchConcurrency
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultBatchTimeout
	}
	runnerFunc := opts.RunnerFunc
	if runnerFunc == nil {
		runnerFunc = NewRunner
	}

	started := time.Now()
	total := len(targets)
	results := make([]TargetExecutionResult, total)
	rawMap := make(map[*Node]RunResult, total)
	var rawMapMu sync.Mutex

	var g errgroup.Group
	g.SetLimit(concurrency)

	for i, t := range targets {
		idx := i
		node := t
		g.Go(func() error {
			if ctx.Err() != nil {
				results[idx] = TargetExecutionResult{
					Node:     node,
					Name:     node.Name,
					Alias:    node.Alias,
					Host:     node.Host,
					ExitCode: -1,
					Error:    ctx.Err().Error(),
				}
				rawMapMu.Lock()
				rawMap[node] = RunResult{ExitCode: -1, Err: ctx.Err()}
				rawMapMu.Unlock()
				return nil
			}

			subCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			runner := runnerFunc(node)
			res := runner.RunCommand(subCtx, cmd)

			errStr := ""
			if res.Err != nil {
				errStr = res.Err.Error()
			}

			results[idx] = TargetExecutionResult{
				Node:       node,
				Name:       node.Name,
				Alias:      node.Alias,
				Host:       node.Host,
				ExitCode:   res.ExitCode,
				Stdout:     string(res.Stdout),
				Stderr:     string(res.Stderr),
				DurationMs: res.Duration.Milliseconds(),
				Error:      errStr,
			}

			rawMapMu.Lock()
			rawMap[node] = res
			rawMapMu.Unlock()
			return nil
		})
	}

	_ = g.Wait()
	finished := time.Now()

	var okCount, failCount int
	for _, r := range results {
		if r.Error == "" && r.ExitCode == 0 {
			okCount++
		} else {
			failCount++
		}
	}

	summary := &BatchSummary{
		Cmd:        cmd,
		Started:    started,
		Finished:   finished,
		DurationMs: finished.Sub(started).Milliseconds(),
		Total:      total,
		Ok:         okCount,
		Failed:     failCount,
		Results:    results,
	}

	if !opts.NoAudit {
		rec := RunRecord{
			Cmd:      cmd,
			Targets:  targets,
			Results:  rawMap,
			Started:  started,
			Finished: finished,
		}
		if _, logDir, err := WriteRun(rec); err == nil {
			summary.LogDir = logDir
		}
	}

	return summary, nil
}
