package main

import (
	"reflect"
	"testing"
)

func TestPrepareArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "empty",
			args: []string{},
			want: []string{},
		},
		{
			name: "subcommand list",
			args: []string{"list"},
			want: []string{"list"},
		},
		{
			name: "subcommand list with json flag",
			args: []string{"list", "--json"},
			want: []string{"list", "--json"},
		},
		{
			name: "subcommand run with target and cmd",
			args: []string{"run", "dev", "uptime"},
			want: []string{"run", "dev", "uptime"},
		},
		{
			name: "boolean flag with subcommand",
			args: []string{"-s", "list", "--json"},
			want: []string{"-s", "list", "--json"},
		},
		{
			name: "explicit config flag with space and subcommand",
			args: []string{"--config", "testdata/mixed.yml", "list", "--json"},
			want: []string{"--config", "testdata/mixed.yml", "list", "--json"},
		},
		{
			name: "config flag with equal sign and subcommand",
			args: []string{"--config=testdata/mixed.yml", "list"},
			want: []string{"--config=testdata/mixed.yml", "list"},
		},
		{
			name: "host target rewritten to connect",
			args: []string{"my-host"},
			want: []string{"connect", "my-host"},
		},
		{
			name: "flags before host target",
			args: []string{"--config", "testdata/mixed.yml", "my-host"},
			want: []string{"--config", "testdata/mixed.yml", "connect", "my-host"},
		},
		{
			name: "ssh-config flag before host target",
			args: []string{"-s", "my-host"},
			want: []string{"-s", "connect", "my-host"},
		},
		{
			name: "inventory flags before host target",
			args: []string{"-i", "https://api.example.com", "-k", "secret", "my-host"},
			want: []string{"-i", "https://api.example.com", "-k", "secret", "connect", "my-host"},
		},
		{
			name: "double dash stops parsing",
			args: []string{"--", "my-host"},
			want: []string{"--", "my-host"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := prepareArgs(rootCmd, tt.args)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("prepareArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}
