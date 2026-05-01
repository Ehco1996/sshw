# sshw

![GitHub](https://img.shields.io/github/license/yinheli/sshw) ![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/yinheli/sshw)

ssh client wrapper for automatic login.

![usage](./assets/sshw-demo.gif)

## install

use `go get`

```
go install github.com/yinheli/sshw/cmd/sshw@latest
```

or download binary from [releases](//github.com/yinheli/sshw/releases).

## config

If `SSHW_CONFIG_PATH` is set to a file path, only that file is tried for the YAML config (see `LoadConfig` in code). Otherwise the config file load order is:

- `~/.sshw`
- `~/.sshw.yml`
- `~/.sshw.yaml`
- `./.sshw`
- `./.sshw.yml`
- `./.sshw.yaml`

When using the `-s` flag (read OpenSSH `~/.ssh/config` instead of YAML), the path defaults to `~/.ssh/config`. You can override it with **`SSHW_SSH_CONFIG_PATH`**, for example to point at a non-standard location or a copy for testing:

```bash
SSHW_SSH_CONFIG_PATH=/path/to/ssh_config sshw -s
```

config example:

<!-- prettier-ignore -->
```yaml
- { name: dev server fully configured, user: appuser, host: 192.168.8.35, port: 22, password: 123456 }
- { name: dev server with key path, user: appuser, host: 192.168.8.35, port: 22, keypath: /root/.ssh/id_rsa }
- { name: dev server with passphrase key, user: appuser, host: 192.168.8.35, port: 22, keypath: /root/.ssh/id_rsa, passphrase: abcdefghijklmn}
- { name: dev server without port, user: appuser, host: 192.168.8.35 }
- { name: dev server without user, host: 192.168.8.35 }
- { name: dev server without password, host: 192.168.8.35 }
- { name: ⚡️ server with emoji name, host: 192.168.8.35 }
- { name: server with alias, alias: dev, host: 192.168.8.35 }
- name: server with jump
  user: appuser
  host: 192.168.8.35
  port: 22
  password: 123456
  jump:
  - user: appuser
    host: 192.168.8.36
    port: 2222


# server group 1
- name: server group 1
  children:
  - { name: server 1, user: root, host: 192.168.1.2 }
  - { name: server 2, user: root, host: 192.168.1.3 }
  - { name: server 3, user: root, host: 192.168.1.4 }

# server group 2
- name: server group 2
  children:
  - { name: server 1, user: root, host: 192.168.2.2 }
  - { name: server 2, user: root, host: 192.168.3.3 }
  - { name: server 3, user: root, host: 192.168.4.4 }
```

# keys

The TUI supports the following keys:

- `↑` / `↓` — move the cursor; `enter` opens a group or connects to a leaf host
- `esc` / `backspace` — leave the current view (group, palette, batch flow); pressing it from the root quits
- `/` — filter by name or alias (multi-keyword: type space-separated terms to AND them)
- `ctrl+k` — toggle the global palette (flat list of every leaf with its breadcrumb)
- `ctrl+h` — TCP healthcheck against every visible host (5 s timeout each)
- `space` — mark/unmark the host under the cursor (used by batch run; only visible after the first mark)
- `ctrl+x` — batch run: type a command, confirm with `y` (or type `yes I am sure` for destructive commands), watch ✓/✗ per host. In the results view: `g` group identical output, `f` filter to failed only, `r` rerun all / `R` rerun failed only, `enter` drills into full output. Every run is written to `~/.local/state/sshw/runs/`. See [batch run](#batch-run) below.
- `q` — quit

# batch run

`ctrl+x` runs one command against multiple hosts in parallel:

- **Targets**: marked hosts (`space`) when there are any, otherwise every connectable host visible in the current view (so entering a group and pressing `ctrl+x` targets that whole group; in `ctrl+k` global mode it targets every host). Healthchecks and batch runs share the same target rules.
- **Concurrency**: at most 8 SSH sessions in flight at a time. Each host has a 30 s wall-clock timeout.
- **Output**: results land in a summary view (`✓/✗  name  user@host  exit=N  Δt  <first-line>`); `enter` opens a scrollable detail viewer with full stdout / stderr / meta.
- **Safety**: there is always an explicit confirmation prompt (`Run on N hosts? [y/N]`). The command string is passed unparsed to the remote shell, so `&&`, pipes, and quoting work the way the remote shell evaluates them.
- **Destructive-command guard**: when the command matches a built-in pattern of destructive shapes (`rm -rf`, `dd if=`, `mkfs`, `shutdown`/`reboot`/`poweroff`/`halt`, `init 0|6`, fork bombs, redirects to `/dev/sd*`, `chmod 000`, `find / … -delete`), the `[y/N]` prompt is replaced with a typed-confirm step requiring you to type `yes I am sure`. Press `esc` to return to the prompt with your command preserved for editing.
- **Limits**: this is intentionally non-interactive — no PTY is requested, so anything that needs a TTY (interactive `sudo`, `vim`, `less`) will not work. Output is captured per-host on completion (no streaming).

## results-view keys

In the post-run summary:

- `↑` / `↓` — move the cursor (works on flat rows or buckets)
- `enter` — drill into the highlighted host's (or bucket's) full output
- `g` — toggle **grouped view** (dshbak-style): hosts are bucketed by identical output, largest bucket first. Drilling into a bucket shows the exemplar output and lists every host in it
- `f` — toggle **failed-only filter**: hide ✓ hosts so the ✗ ones are easy to find
- `r` — rerun the same command on the **full** target set
- `R` — rerun the same command on the **failed subset** only (Ansible-style `--limit @retry`)
- `esc` — back to the host list (clears selection)

In the detail viewer: `↑↓ / pgup / pgdn / g / G` to scroll, `esc` to return to results.

## audit log

Every completed batch run is persisted under `~/.local/state/sshw/runs/` (or `$XDG_STATE_HOME/sshw/runs/` if set, or `$SSHW_RUN_LOG_DIR` for an explicit override):

- `runs.jsonl` — append-only index, one JSON line per run with `ts`, `cmd`, `hosts`, `ok`/`fail`/`total`, `duration_ms`, `log_dir`. Greppable.
- `runs/<run_id>/<host>.log` — per-host plain-text log with stdout, stderr, exit code, and metadata.

Passwords and passphrases are never written to disk. The results view shows the directory path on the second line so you can `tail` or `grep` it from another shell.

# callback

<!-- prettier-ignore -->
```yaml
- name: dev server fully configured
  user: appuser
  host: 192.168.8.35
  port: 22
  password: 123456
  callback-shells:
    - { cmd: 2 }
    - { delay: 1500, cmd: 0 }
    - { cmd: "echo 1" }
```
