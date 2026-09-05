package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/yinheli/sshw"
	"github.com/yinheli/sshw/internal/tui"
)

var (
	Build = "devel"

	explicitConfigPath string
	useSSHConfig       bool
	dynamicURL         string
	dynamicKey         string

	rootCmd = &cobra.Command{
		Use:          "sshw [flags] [subcommand | host]",
		Short:        "ssh client wrapper for automatic login and fleet orchestration",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			nodes, err := loadNodes()
			if err != nil {
				return err
			}
			node, err := tui.Run(nodes, !useSSHConfig)
			if err != nil {
				return err
			}
			if node == nil {
				return nil
			}
			client := sshw.NewClient(node)
			client.Login()
			return nil
		},
	}

	connectCmd = &cobra.Command{
		Use:    "connect <host>",
		Short:  "Connect to a host by name or alias",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nodes, err := loadNodes()
			if err != nil {
				return err
			}
			token := args[0]
			matches := sshw.FindConnectableByNameOrAlias(nodes, token)
			switch len(matches) {
			case 1:
				client := sshw.NewClient(matches[0])
				client.Login()
				return nil
			case 0:
				return fmt.Errorf("no host matched name or alias %q", token)
			default:
				fmt.Fprintf(os.Stderr, "ambiguous name or alias %q (%d matches):\n", token, len(matches))
				for i, n := range matches {
					fmt.Fprintf(os.Stderr, "  %d) %s @ %s (%s)\n", i+1, n.EffectiveUser(), n.Host, n.Name)
				}
				os.Exit(1)
				return nil
			}
		},
	}
)

func loadNodes() ([]*sshw.Node, error) {
	return sshw.LoadInventory(sshw.InventoryOptions{
		ConfigPath:   explicitConfigPath,
		UseSSHConfig: useSSHConfig,
		DynamicURL:   dynamicURL,
		DynamicKey:   dynamicKey,
	})
}

func init() {
	rootCmd.Version = Build
	rootCmd.PersistentFlags().StringVar(&explicitConfigPath, "config", "", "path to YAML config file")
	rootCmd.PersistentFlags().BoolVarP(&useSSHConfig, "ssh-config", "s", false, "use local ssh config '~/.ssh/config'")
	rootCmd.PersistentFlags().StringVarP(&dynamicURL, "inventory-url", "i", "", "dynamic inventory URL")
	rootCmd.PersistentFlags().StringVarP(&dynamicKey, "inventory-key", "k", "", "API key for dynamic inventory")

	rootCmd.AddCommand(connectCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(runCmd)
}

func lookupFlag(cmd *cobra.Command, name string, isShorthand bool) *pflag.Flag {
	if isShorthand {
		if f := cmd.Flags().ShorthandLookup(name); f != nil {
			return f
		}
		return cmd.PersistentFlags().ShorthandLookup(name)
	}
	return cmd.Flag(name)
}

// prepareArgs rewrites args so that arbitrary host targets (e.g. `sshw dev`) route to `connectCmd`.
func prepareArgs(cmd *cobra.Command, args []string) []string {
	if len(args) == 0 {
		return args
	}

	i := 0
	for i < len(args) {
		arg := args[i]
		if arg == "--" {
			break
		}
		if strings.HasPrefix(arg, "--") {
			flagPart := strings.TrimPrefix(arg, "--")
			if strings.Contains(flagPart, "=") {
				i++
				continue
			}
			f := lookupFlag(cmd, flagPart, false)
			if f != nil && f.NoOptDefVal == "" {
				i += 2
				continue
			}
			i++
			continue
		} else if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			flagPart := strings.TrimPrefix(arg, "-")
			if strings.Contains(flagPart, "=") {
				i++
				continue
			}
			f := lookupFlag(cmd, flagPart, true)
			if f != nil && f.NoOptDefVal == "" {
				i += 2
				continue
			}
			i++
			continue
		}

		foundCmd, _, _ := cmd.Find([]string{arg})
		if foundCmd != nil && foundCmd != cmd {
			return args
		}
		newArgs := make([]string, 0, len(args)+1)
		newArgs = append(newArgs, args[:i]...)
		newArgs = append(newArgs, "connect")
		newArgs = append(newArgs, args[i:]...)
		return newArgs
	}
	return args
}

// Execute runs the root command.
func Execute() error {
	rootCmd.SetArgs(prepareArgs(rootCmd, os.Args[1:]))
	return rootCmd.Execute()
}
