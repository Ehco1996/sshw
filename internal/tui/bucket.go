package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yinheli/sshw"
)

// bucket groups hosts that produced functionally identical output.
// `key` is the hash-stable identity used for grouping; `klass` and
// `exemplar` drive rendering of the bucket's row.
type bucket struct {
	key      string
	klass    string         // "exit=0" / "exit=1" / "timeout" / ...
	hosts    []*sshw.Node   // display-order subset of targets that landed in this bucket
	exemplar sshw.RunResult // result of the first host (representative content)
}

// errClass returns a short tag for connection-class errors so timeouts on
// 5 hosts can coalesce into one bucket. Mirrors renderResultBadge.
func errClass(r sshw.RunResult) string {
	if r.Err == nil {
		return ""
	}
	msg := r.Err.Error()
	switch {
	case strings.Contains(msg, "context deadline exceeded"),
		strings.Contains(msg, "timeout"),
		strings.Contains(msg, "deadline"):
		return "timeout"
	case strings.Contains(msg, "refused"):
		return "refused"
	case strings.Contains(msg, "no route"):
		return "no-route"
	case strings.Contains(msg, "auth"),
		strings.Contains(msg, "unable to authenticate"):
		return "auth"
	}
	return "error"
}

// bucketKlass returns the human-readable label that drives bucket-row badge
// rendering. Errors come first so all-timeout buckets coalesce regardless
// of exit-code value.
func bucketKlass(r sshw.RunResult) string {
	if r.Err != nil && r.ExitCode == -1 {
		return errClass(r)
	}
	return fmt.Sprintf("exit=%d", r.ExitCode)
}

// bucketKey is the identity used for grouping. Two results with the same
// key are guaranteed to render the same way; differences in surface noise
// (ANSI, trailing whitespace) are normalized away.
func bucketKey(r sshw.RunResult) string {
	return strings.Join([]string{
		bucketKlass(r),
		"\x00stdout\x00",
		normalizeOutput(string(r.Stdout)),
		"\x00stderr\x00",
		normalizeOutput(string(r.Stderr)),
	}, "")
}

// computeBuckets groups completed results by output. Pending hosts (those
// without a done result) are silently skipped — grouping is meaningful
// only post-completion. Buckets sort largest-first, ties broken by class
// ascending so exit=0 tends to sort before failures within equal counts.
func computeBuckets(targets []*sshw.Node, results map[*sshw.Node]*batchTargetResult) []bucket {
	indexByKey := make(map[string]int, len(targets))
	out := make([]bucket, 0, len(targets))
	for _, n := range targets {
		r := results[n]
		if r == nil || !r.done {
			continue
		}
		k := bucketKey(r.res)
		if i, ok := indexByKey[k]; ok {
			out[i].hosts = append(out[i].hosts, n)
			continue
		}
		indexByKey[k] = len(out)
		out = append(out, bucket{
			key:      k,
			klass:    bucketKlass(r.res),
			hosts:    []*sshw.Node{n},
			exemplar: r.res,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if len(out[i].hosts) != len(out[j].hosts) {
			return len(out[i].hosts) > len(out[j].hosts)
		}
		return out[i].klass < out[j].klass
	})
	return out
}

// hostListLabel renders a compact, fixed-width list of host names from a
// bucket: "a, b, c" up to maxNames, then "+N more". Used in the detail
// view header so users can see which hosts coalesced.
func hostListLabel(hosts []*sshw.Node, maxNames int) string {
	if len(hosts) == 0 {
		return ""
	}
	if len(hosts) <= maxNames {
		names := make([]string, len(hosts))
		for i, h := range hosts {
			names[i] = h.Name
		}
		return strings.Join(names, ", ")
	}
	names := make([]string, maxNames)
	for i := 0; i < maxNames; i++ {
		names[i] = hosts[i].Name
	}
	return strings.Join(names, ", ") + fmt.Sprintf(", +%d more", len(hosts)-maxNames)
}
