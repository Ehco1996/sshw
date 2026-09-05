package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yinheli/sshw"
)

var (
	runAsJSON   bool
	dryRun      bool
	concurrency int
	timeout     time.Duration
	force       bool
)

var runCmd = &cobra.Command{
	Use:     "run <target> <command...>",
	Aliases: []string{"exec"},
	Short:   "Run command on target host(s)",
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetPattern := args[0]
		nodes, err := loadNodes()
		if err != nil {
			return err
		}

		targets := sshw.MatchTargets(nodes, targetPattern)
		if len(targets) == 0 {
			return fmt.Errorf("no targets matched %q", targetPattern)
		}

		if dryRun {
			if runAsJSON {
				infos := sshw.FilterNodeInfos(nodes, targetPattern)
				data, _ := json.MarshalIndent(infos, "", "  ")
				fmt.Println(string(data))
			} else {
				fmt.Printf("Matched %d target(s) for %q:\n", len(targets), targetPattern)
				for _, t := range targets {
					aliasPart := ""
					if t.Alias != "" {
						aliasPart = fmt.Sprintf(" [alias: %s]", t.Alias)
					}
					fmt.Printf("  - %s%s (%s@%s:%d)\n", t.Name, aliasPart, t.EffectiveUser(), t.Host, t.SSHPort())
				}
			}
			return nil
		}

		if len(args) < 2 {
			return errors.New("command is required")
		}

		cmdStr := strings.Join(args[1:], " ")
		opts := sshw.BatchOptions{
			Concurrency: concurrency,
			Timeout:     timeout,
			ForceDanger: force,
		}

		summary, err := sshw.RunBatch(cmd.Context(), targets, cmdStr, opts)
		if err != nil {
			return fmt.Errorf("execution aborted: %w", err)
		}

		if runAsJSON {
			data, err := json.MarshalIndent(summary, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			if summary.Failed > 0 {
				os.Exit(1)
			}
			return nil
		}

		// Single target: direct pass-through experience
		if len(targets) == 1 {
			res := summary.Results[0]
			if res.Stdout != "" {
				fmt.Print(res.Stdout)
			}
			if res.Stderr != "" {
				fmt.Fprint(os.Stderr, res.Stderr)
			}
			if res.Error != "" {
				fmt.Fprintf(os.Stderr, "error: %s\n", res.Error)
			}
			if res.ExitCode != 0 {
				if res.ExitCode > 0 {
					os.Exit(res.ExitCode)
				}
				os.Exit(1)
			}
			return nil
		}

		// Multi-target: formatted summary
		for _, r := range summary.Results {
			badge := "✓"
			if r.Error != "" || r.ExitCode != 0 {
				badge = "✗"
			}
			aliasPart := ""
			if r.Alias != "" {
				aliasPart = fmt.Sprintf(" [%s]", r.Alias)
			}
			fmt.Printf("%s  %s%s (%s) [exit=%d, %dms]\n", badge, r.Name, aliasPart, r.Host, r.ExitCode, r.DurationMs)
			if r.Stdout != "" {
				lines := strings.Split(strings.TrimRight(r.Stdout, "\n"), "\n")
				for _, l := range lines {
					fmt.Printf("   %s\n", l)
				}
			}
			if r.Stderr != "" {
				lines := strings.Split(strings.TrimRight(r.Stderr, "\n"), "\n")
				for _, l := range lines {
					fmt.Printf("   [stderr] %s\n", l)
				}
			}
			if r.Error != "" {
				fmt.Printf("   [error] %s\n", r.Error)
			}
		}

		fmt.Printf("\nCompleted in %dms: %d total, %d ok, %d failed\n",
			summary.DurationMs, summary.Total, summary.Ok, summary.Failed)
		if summary.LogDir != "" {
			fmt.Printf("Audit log: %s\n", summary.LogDir)
		}

		if summary.Failed > 0 {
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	runCmd.Flags().BoolVar(&runAsJSON, "json", false, "output summary in JSON format")
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "list matched targets without executing")
	runCmd.Flags().IntVarP(&concurrency, "concurrency", "c", sshw.DefaultBatchConcurrency, "max concurrent connections")
	runCmd.Flags().DurationVarP(&timeout, "timeout", "t", sshw.DefaultBatchTimeout, "per-host timeout")
	runCmd.Flags().BoolVarP(&force, "force", "f", false, "allow execution of dangerous commands")
}
