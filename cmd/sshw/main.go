package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/yinheli/sshw"
	"github.com/yinheli/sshw/internal/tui"
)

var (
	Build = "devel"
	V     = flag.Bool("version", false, "show version")
	H     = flag.Bool("help", false, "show help")
	S     = flag.Bool("s", false, "use local ssh config '~/.ssh/config'")
	I     = flag.String("i", "", "use dynamic inventory url (e.g., https://.../api/inventory/)")
	K     = flag.String("k", "", "API key for dynamic inventory (sent as X-API-KEY header)")

	log = sshw.GetLogger()
)

func loadInventory(s bool, i, k string) error {
	if i != "" {
		if k == "" {
			return fmt.Errorf("-k (API key) is required when using -i")
		}
		return sshw.LoadDynamicConfig(i, k)
	}
	if s {
		return sshw.LoadSshConfig()
	}
	return sshw.LoadConfig()
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: sshw [flags] [subcommand | host]\n\n")
		fmt.Fprintf(os.Stderr, "Subcommands:\n")
		fmt.Fprintf(os.Stderr, "  list [--json]              List configured hosts\n")
		fmt.Fprintf(os.Stderr, "  run [flags] <target> <cmd> Run command on target host(s)\n")
		fmt.Fprintf(os.Stderr, "  exec [flags] <target> <cmd> Alias for run\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()
	if !flag.Parsed() {
		flag.Usage()
		return
	}

	if *H {
		flag.Usage()
		return
	}

	if *V {
		fmt.Println("sshw - ssh client wrapper for automatic login")
		fmt.Println("  git version:", Build)
		fmt.Println("  go version :", runtime.Version())
		return
	}

	if flag.NArg() > 0 {
		subcmd := flag.Arg(0)
		switch subcmd {
		case "list":
			runList(flag.Args()[1:])
			return
		case "run", "exec":
			runExec(flag.Args()[1:])
			return
		}
	}

	if err := loadInventory(*S, *I, *K); err != nil {
		log.Error("load config error", err)
		os.Exit(1)
	}

	token := ""
	if flag.NArg() > 0 {
		token = flag.Arg(0)
	}

	if token != "" {
		nodes := sshw.GetConfig()
		matches := sshw.FindConnectableByNameOrAlias(nodes, token)
		switch len(matches) {
		case 1:
			client := sshw.NewClient(matches[0])
			client.Login()
			return
		case 0:
			// fall through to TUI
		default:
			fmt.Fprintf(os.Stderr, "ambiguous name or alias %q (%d matches):\n", token, len(matches))
			for i, n := range matches {
				fmt.Fprintf(os.Stderr, "  %d) %s @ %s (%s)\n", i+1, n.EffectiveUser(), n.Host, n.Name)
			}
			os.Exit(1)
		}
	}

	node, err := tui.Run(sshw.GetConfig(), !*S)
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
