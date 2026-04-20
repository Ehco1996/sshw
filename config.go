package sshw

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"time"

	"github.com/atrox/homedir"
	"github.com/kevinburke/ssh_config"
	"gopkg.in/yaml.v2"
)

type Node struct {
	Name           string           `yaml:"name"`
	Alias          string           `yaml:"alias"`
	Host           string           `yaml:"host"`
	User           string           `yaml:"user"`
	Port           int              `yaml:"port"`
	KeyPath        string           `yaml:"keypath"`
	AgentPath      string           `yaml:"agentpath"`
	Passphrase     string           `yaml:"passphrase"`
	Password       string           `yaml:"password"`
	CallbackShells []*CallbackShell `yaml:"callback-shells"`
	Children       []*Node          `yaml:"children"`
	Jump           []*Node          `yaml:"jump"`
}

type CallbackShell struct {
	Cmd   string        `yaml:"cmd"`
	Delay time.Duration `yaml:"delay"`
}

func (n *Node) String() string {
	return n.Name
}

// Connectable reports whether this node is a leaf host entry (same rule as the TUI: Enter connects).
func (n *Node) Connectable() bool {
	return n.Host != "" && len(n.Children) == 0
}

// FindConnectableByNameOrAlias returns all connectable nodes whose Name or Alias equals token (exact match).
func FindConnectableByNameOrAlias(nodes []*Node, token string) []*Node {
	var out []*Node
	var walk func([]*Node)
	walk = func(ns []*Node) {
		for _, n := range ns {
			if n.Connectable() && (n.Name == token || (n.Alias != "" && n.Alias == token)) {
				out = append(out, n)
			}
			if len(n.Children) > 0 {
				walk(n.Children)
			}
		}
	}
	walk(nodes)
	return out
}

func (n *Node) user() string {
	if n.User == "" {
		return "root"
	}
	return n.User
}

func (n *Node) port() int {
	if n.Port <= 0 {
		return 22
	}
	return n.Port
}

// EffectiveUser returns User, or "root" when empty (same rule as SSH client config).
func (n *Node) EffectiveUser() string {
	return n.user()
}

// SSHPort returns Port, or 22 when unset or non-positive.
func (n *Node) SSHPort() int {
	return n.port()
}

// JumpLabel returns the first jump hop's Name, or Host if Name is empty; empty if there is no jump.
func (n *Node) JumpLabel() string {
	if len(n.Jump) == 0 {
		return ""
	}
	j := n.Jump[0]
	if j.Name != "" {
		return j.Name
	}
	return j.Host
}

var (
	config []*Node
)

func GetConfig() []*Node {
	return config
}

func LoadConfig() error {
	var paths []string
	if envPath := os.Getenv("SSHW_CONFIG_PATH"); envPath != "" {
		paths = []string{envPath}
	}
	paths = append(paths, ".sshw", ".sshw.yml", ".sshw.yaml")
	b, err := LoadConfigBytes(paths...)
	if err != nil {
		return err
	}
	var c []*Node
	err = yaml.Unmarshal(b, &c)
	if err != nil {
		return err
	}

	config = c

	return nil
}

func LoadSshConfig() error {
	var p string
	if ep := os.Getenv("SSHW_SSH_CONFIG_PATH"); ep != "" {
		p = ep
	} else {
		u, err := user.Current()
		if err != nil {
			l.Error(err)
			return err
		}
		p = filepath.Join(u.HomeDir, ".ssh", "config")
	}
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			config = nil
			return nil
		}
		return err
	}
	defer f.Close()

	cfg, err := ssh_config.Decode(f)
	if err != nil {
		return err
	}
	var nc []*Node
	for _, host := range cfg.Hosts {
		alias := fmt.Sprintf("%s", host.Patterns[0])
		hostName, err := cfg.Get(alias, "HostName")
		if err != nil {
			return err
		}
		if hostName != "" {
			port, _ := cfg.Get(alias, "Port")
			if port == "" {
				port = "22"
			}
			var c = new(Node)
			c.Name = alias
			c.Alias = alias
			c.Host = hostName
			c.User, _ = cfg.Get(alias, "User")
			c.Port, _ = strconv.Atoi(port)
			keyPath, _ := cfg.Get(alias, "IdentityFile")
			c.KeyPath, _ = homedir.Expand(keyPath)
			agentPath, _ := cfg.Get(alias, "IdentityAgent")
			c.AgentPath, _ = homedir.Expand(agentPath)
			nc = append(nc, c)
			// fmt.Println(c.Alias, c.Host, c.User, c.Port, c.KeyPath)
		}
	}
	config = nc
	return nil
}

func LoadConfigBytes(names ...string) ([]byte, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	var lastErr error
	// homedir
	for i := range names {
		sshw, err := os.ReadFile(filepath.Join(u.HomeDir, names[i]))
		if err == nil {
			return sshw, nil
		}
		lastErr = err
	}
	// relative
	for i := range names {
		sshw, err := os.ReadFile(names[i])
		if err == nil {
			return sshw, nil
		}
		lastErr = err
	}
	return nil, lastErr
}
