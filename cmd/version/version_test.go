package version

import (
	"io"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_Version(t *testing.T) {
	cmd := NewCmd()

	// capture stdout
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("error running version command: %s", err)
	}

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = rescueStdout

	str := string(out)
	diff := cmp.Diff("(devel)\n", str)
	if diff != "" {
		t.Fatalf("Failed. expected: (devel) got: %s. diff: %s", str, diff)
	}
}
