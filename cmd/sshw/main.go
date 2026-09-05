package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/yinheli/sshw"
	"github.com/yinheli/sshw/internal/tui"
)

var (
	Build = "devel"
	log   = sshw.GetLogger()
)

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: sshw [flags] [subcommand | host]\n\n")
	fmt.Fprintf(os.Stderr, "Subcommands:\n")
	fmt.Fprintf(os.Stderr, "  list [--json]              List configured hosts\n")
	fmt.Fprintf(os.Stderr, "  run [flags] <target> <cmd> Run command on target host(s)\n")
	fmt.Fprintf(os.Stderr, "  exec [flags] <target> <cmd> Alias for run\n\n")
	fmt.Fprintf(os.Stderr, "Global Flags:\n")
	fmt.Fprintf(os.Stderr, "  -s                         use local ssh config '~/.ssh/config'\n")
	fmt.Fprintf(os.Stderr, "  -i string                  dynamic inventory URL\n")
	fmt.Fprintf(os.Stderr, "  -k string                  API key for dynamic inventory\n")
	fmt.Fprintf(os.Stderr, "  -version                   show version\n")
	fmt.Fprintf(os.Stderr, "  -help                      show help\n")
}

func parseGlobalFlags(args []string) (opts sshw.InventoryOptions, showHelp, showVersion bool, rest []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help" || arg == "-help":
			showHelp = true
		case arg == "-v" || arg == "--version" || arg == "-version":
			showVersion = true
		case arg == "-s" || arg == "--s":
			opts.UseSSHConfig = true
		case arg == "-i" || arg == "--i":
			if i+1 < len(args) {
				opts.DynamicURL = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "-i="):
			opts.DynamicURL = strings.TrimPrefix(arg, "-i=")
		case arg == "-k" || arg == "--k":
			if i+1 < len(args) {
				opts.DynamicKey = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "-k="):
			opts.DynamicKey = strings.TrimPrefix(arg, "-k=")
		default:
			rest = append(rest, arg)
		}
	}
	return
}

func main() {
	opts, showHelp, showVersion, rest := parseGlobalFlags(os.Args[1:])

	if showHelp {
		printUsage()
		return
	}

	if showVersion {
		fmt.Println("sshw - ssh client wrapper for automatic login")
		fmt.Println("  git version:", Build)
		fmt.Println("  go version :", runtime.Version())
		return
	}

	nodes, err := sshw.LoadInventory(opts)
	if err != nil {
		log.Error("load inventory error", err)
		os.Exit(1)
	}

	if len(rest) > 0 {
		subcmd := rest[0]
		switch subcmd {
		case "list":
			runList(nodes, rest[1:])
			return
		case "run", "exec":
			runExec(nodes, rest[1:])
			return
		}

		// Single host interactive login by Name or Alias (SSOT via FindConnectableByNameOrAlias)
		matches := sshw.FindConnectableByNameOrAlias(nodes, subcmd)
		switch len(matches) {
		case 1:
			client := sshw.NewClient(matches[0])
			client.Login()
			return
		case 0:
			// fall through to TUI
		default:
			fmt.Fprintf(os.Stderr, "ambiguous name or alias %q (%d matches):\n", subcmd, len(matches))
			for i, n := range matches {
				fmt.Fprintf(os.Stderr, "  %d) %s @ %s (%s)\n", i+1, n.EffectiveUser(), n.Host, n.Name)
			}
			os.Exit(1)
		}
	}

	node, err := tui.Run(nodes, !opts.UseSSHConfig)
	if err != nil {
		log.Error("tui error", err)
		os.Exit(1)
	}
	if node == nil {
		return
	}

	client := sshw.NewClient(node)
	client.Login()
}
