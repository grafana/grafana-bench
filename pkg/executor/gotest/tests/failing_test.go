// +build goexecutor

package tests

import (
	"testing"
	"time"
)

func TestFailing(t *testing.T) {
	time.Sleep(time.Millisecond * 100)
	t.Fail()
}

