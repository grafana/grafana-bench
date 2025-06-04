// +build goexecutor

package subpkg

import (
	"testing"
	"time"
)

func TestPassing4(t *testing.T) {
	time.Sleep(time.Millisecond * 20)
}

