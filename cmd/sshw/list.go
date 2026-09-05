package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/yinheli/sshw"
)

func runList(nodes []*sshw.Node, args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "output in JSON format")
	_ = fs.Parse(args)

	infos := sshw.ListNodeInfos(nodes)

	if *asJSON {
		data, err := json.MarshalIndent(infos, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error encoding JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tALIAS\tTARGET\tJUMP\tPATH")
	for _, info := range infos {
		target := fmt.Sprintf("%s@%s:%d", info.User, info.Host, info.Port)
		jump := info.Jump
		if jump == "" {
			jump = "-"
		}
		alias := info.Alias
		if alias == "" {
			alias = "-"
		}
		pathStr := "-"
		if len(info.Path) > 0 {
			pathStr = strings.Join(info.Path, " ❯ ")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", info.Name, alias, target, jump, pathStr)
	}
	_ = w.Flush()
}
