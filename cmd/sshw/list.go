package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/yinheli/sshw"
)

var listAsJSON bool

var listCmd = &cobra.Command{
	Use:   "list [target]",
	Short: "List configured hosts",
	RunE: func(cmd *cobra.Command, args []string) error {
		nodes, err := loadNodes()
		if err != nil {
			return err
		}

		pattern := ""
		if len(args) > 0 {
			pattern = args[0]
		}
		infos := sshw.FilterNodeInfos(nodes, pattern)

		if listAsJSON {
			data, err := json.MarshalIndent(infos, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
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
		return w.Flush()
	},
}

func init() {
	listCmd.Flags().BoolVar(&listAsJSON, "json", false, "output in JSON format")
}
