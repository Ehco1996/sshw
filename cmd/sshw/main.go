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

	log = sshw.GetLogger()
)

func main() {
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
	if *S {
		err := sshw.LoadSshConfig()
		if err != nil {
			log.Error("load ssh config error", err)
			os.Exit(1)
		}
	} else {
		err := sshw.LoadConfig()
		if err != nil {
			log.Error("load config error", err)
			os.Exit(1)
		}
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

	node, err := tui.Run(sshw.GetConfig())
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
