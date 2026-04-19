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

func findAlias(nodes []*sshw.Node, nodeAlias string) *sshw.Node {
	for _, node := range nodes {
		if node.Alias == nodeAlias {
			return node
		}
		if len(node.Children) > 0 {
			return findAlias(node.Children, nodeAlias)
		}
	}
	return nil
}

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

	// login by alias
	if len(os.Args) > 1 {
		var nodeAlias = os.Args[1]
		var nodes = sshw.GetConfig()
		var node = findAlias(nodes, nodeAlias)
		if node != nil {
			client := sshw.NewClient(node)
			client.Login()
			return
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
