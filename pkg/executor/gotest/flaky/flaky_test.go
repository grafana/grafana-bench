
// +build goexecutor

package tests

import (
	"flag"
	"os"
	"testing"
)

var markFile string

func TestMain(m *testing.M) {
	flag.StringVar(&markFile, "flaky-mark-file","","path to flaky test mark file")
	flag.Parse()

	m.Run()
}


// TestFlaky passes if the flaky mark file does not exists.
// When it fails removes the file, so next execution will pass
func TestFlaky(t *testing.T) {
	if markFile == "" {
		t.Fatalf("flaky mark file should be specified")
	}

	_, err := os.Stat(markFile)
	if os.IsNotExist(err) {
		return
	}

	_ = os.Remove(markFile)

	t.Fatal()
}
