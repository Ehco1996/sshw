package tui

import (
	"errors"
	"testing"

	"github.com/yinheli/sshw"
)

func mkRes(stdout, stderr string, exit int, err error) *batchTargetResult {
	return &batchTargetResult{
		done: true,
		res: sshw.RunResult{
			Stdout:   []byte(stdout),
			Stderr:   []byte(stderr),
			ExitCode: exit,
			Err:      err,
		},
	}
}

func TestComputeBuckets_AllIdentical(t *testing.T) {
	t.Parallel()
	a := &sshw.Node{Name: "a"}
	b := &sshw.Node{Name: "b"}
	c := &sshw.Node{Name: "c"}
	results := map[*sshw.Node]*batchTargetResult{
		a: mkRes("ok\n", "", 0, nil),
		b: mkRes("ok\n", "", 0, nil),
		c: mkRes("ok\n", "", 0, nil),
	}
	got := computeBuckets([]*sshw.Node{a, b, c}, results)
	if len(got) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(got))
	}
	if len(got[0].hosts) != 3 {
		t.Errorf("bucket should hold 3 hosts, got %d", len(got[0].hosts))
	}
	if got[0].klass != "exit=0" {
		t.Errorf("klass = %q, want exit=0", got[0].klass)
	}
}

func TestComputeBuckets_SplitOnExit(t *testing.T) {
	t.Parallel()
	a := &sshw.Node{Name: "a"}
	b := &sshw.Node{Name: "b"}
	c := &sshw.Node{Name: "c"}
	results := map[*sshw.Node]*batchTargetResult{
		a: mkRes("ok\n", "", 0, nil),
		b: mkRes("ok\n", "", 0, nil),
		c: mkRes("ok\n", "", 1, nil), // same stdout but different exit → distinct bucket
	}
	got := computeBuckets([]*sshw.Node{a, b, c}, results)
	if len(got) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(got))
	}
	// Largest first.
	if len(got[0].hosts) != 2 || got[0].klass != "exit=0" {
		t.Errorf("bucket[0] wrong: hosts=%d klass=%q", len(got[0].hosts), got[0].klass)
	}
	if len(got[1].hosts) != 1 || got[1].klass != "exit=1" {
		t.Errorf("bucket[1] wrong: hosts=%d klass=%q", len(got[1].hosts), got[1].klass)
	}
}

func TestComputeBuckets_NormalizeWhitespaceAndAnsi(t *testing.T) {
	t.Parallel()
	a := &sshw.Node{Name: "a"}
	b := &sshw.Node{Name: "b"}
	c := &sshw.Node{Name: "c"}
	results := map[*sshw.Node]*batchTargetResult{
		a: mkRes("hello\nworld\n", "", 0, nil),
		b: mkRes("hello   \nworld\n\n\n", "", 0, nil),         // trailing ws + extra trailing newlines
		c: mkRes("\x1b[32mhello\x1b[0m\nworld\n", "", 0, nil), // colored
	}
	got := computeBuckets([]*sshw.Node{a, b, c}, results)
	if len(got) != 1 {
		t.Fatalf("normalization failed; got %d buckets", len(got))
	}
}

func TestComputeBuckets_TimeoutCoalesce(t *testing.T) {
	t.Parallel()
	a := &sshw.Node{Name: "a"}
	b := &sshw.Node{Name: "b"}
	c := &sshw.Node{Name: "c"}
	to := errors.New("dial: context deadline exceeded")
	results := map[*sshw.Node]*batchTargetResult{
		a: mkRes("", "", -1, to),
		b: mkRes("", "", -1, errors.New("read: i/o timeout")),
		c: mkRes("ok\n", "", 0, nil),
	}
	got := computeBuckets([]*sshw.Node{a, b, c}, results)
	if len(got) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(got))
	}
	// Two timeouts should coalesce.
	if len(got[0].hosts) != 2 || got[0].klass != "timeout" {
		t.Errorf("largest bucket wrong: %+v", got[0])
	}
}

func TestComputeBuckets_SkipsPending(t *testing.T) {
	t.Parallel()
	a := &sshw.Node{Name: "a"}
	b := &sshw.Node{Name: "b"}
	results := map[*sshw.Node]*batchTargetResult{
		a: mkRes("ok\n", "", 0, nil),
		b: {done: false}, // pending
	}
	got := computeBuckets([]*sshw.Node{a, b}, results)
	if len(got) != 1 || len(got[0].hosts) != 1 {
		t.Fatalf("pending host should be skipped; got %+v", got)
	}
}

func TestHostListLabel(t *testing.T) {
	t.Parallel()
	hosts := []*sshw.Node{{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"}}
	if got := hostListLabel(hosts, 5); got != "a, b, c, d" {
		t.Errorf("got %q", got)
	}
	if got := hostListLabel(hosts, 2); got != "a, b, +2 more" {
		t.Errorf("got %q", got)
	}
	if got := hostListLabel(nil, 5); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}
