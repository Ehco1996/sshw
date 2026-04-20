package sshw

import (
	"log"
	"os"
	"strings"
	"testing"
)

func TestDefaultLogger_prefixes(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	old := stdlog
	stdlog = log.New(w, "[sshw] ", 0)
	defer func() { stdlog = old }()

	done := make(chan struct{})
	var out strings.Builder
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				out.Write(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	lg := &logger{}
	lg.Info("a", "b")
	lg.Infof("x=%s", "y")
	lg.Error("oops")
	lg.Errorf("bad %d", 1)
	_ = w.Close()
	<-done

	s := out.String()
	if !strings.Contains(s, "[sshw]") || !strings.Contains(s, "[info]") {
		t.Fatalf("expected sshw+info prefix in %q", s)
	}
	if !strings.Contains(s, "[error]") {
		t.Fatalf("expected [error] in %q", s)
	}
	if strings.Contains(s, "[level]") {
		t.Fatalf("unexpected legacy Errorf tag [level] in %q", s)
	}
}
