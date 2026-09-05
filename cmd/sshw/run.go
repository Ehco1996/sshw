package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yinheli/sshw"
)

func runExec(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "output summary in JSON format")
	dryRun := fs.Bool("dry-run", false, "list matched targets without executing")

	var concurrency int
	fs.IntVar(&concurrency, "c", 8, "max concurrent connections")
	fs.IntVar(&concurrency, "concurrency", 8, "max concurrent connections")

	var timeout time.Duration
	fs.DurationVar(&timeout, "t", 30*time.Second, "per-host timeout")
	fs.DurationVar(&timeout, "timeout", 30*time.Second, "per-host timeout")

	var force bool
	fs.BoolVar(&force, "f", false, "allow execution of dangerous commands")
	fs.BoolVar(&force, "force", false, "allow execution of dangerous commands")

	useSsh := fs.Bool("s", false, "use local ssh config '~/.ssh/config'")
	invURL := fs.String("i", "", "dynamic inventory url")
	invKey := fs.String("k", "", "API key for dynamic inventory")

	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: sshw run [flags] <target> <command...>")
		fmt.Fprintln(os.Stderr, "Flags:")
		fs.PrintDefaults()
		os.Exit(1)
	}

	targetPattern := fs.Arg(0)

	s := *S || *useSsh
	i := *I
	if *invURL != "" {
		i = *invURL
	}
	k := *K
	if *invKey != "" {
		k = *invKey
	}

	if err := loadInventory(s, i, k); err != nil {
		fmt.Fprintf(os.Stderr, "error loading inventory: %v\n", err)
		os.Exit(1)
	}

	nodes := sshw.GetConfig()
	targets := sshw.MatchTargets(nodes, targetPattern)
	if len(targets) == 0 {
		fmt.Fprintf(os.Stderr, "no targets matched %q\n", targetPattern)
		os.Exit(1)
	}

	if *dryRun {
		if *asJSON {
			infos := make([]sshw.NodeInfo, len(targets))
			for idx, t := range targets {
				infos[idx] = sshw.BuildNodeInfo(t, nil)
			}
			data, _ := json.MarshalIndent(infos, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Printf("Matched %d target(s) for %q:\n", len(targets), targetPattern)
			for _, t := range targets {
				fmt.Printf("  - %s (%s@%s:%d)\n", t.Name, t.EffectiveUser(), t.Host, t.SSHPort())
			}
		}
		return
	}

	if fs.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "error: command is required")
		fmt.Fprintln(os.Stderr, "Usage: sshw run [flags] <target> <command...>")
		os.Exit(1)
	}

	cmdStr := strings.Join(fs.Args()[1:], " ")

	opts := sshw.BatchOptions{
		Concurrency: concurrency,
		Timeout:     timeout,
		ForceDanger: force,
	}

	summary, err := sshw.RunBatch(context.Background(), targets, cmdStr, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "execution aborted: %v\n", err)
		os.Exit(1)
	}

	if *asJSON {
		data, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error encoding JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
		if summary.Failed > 0 {
			os.Exit(1)
		}
		return
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
		return
	}

	// Multi-target: formatted summary
	for _, r := range summary.Results {
		badge := "✓"
		if r.Error != "" || r.ExitCode != 0 {
			badge = "✗"
		}
		fmt.Printf("%s  %s (%s) [exit=%d, %dms]\n", badge, r.Name, r.Host, r.ExitCode, r.DurationMs)
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
}
