// +build goexecutor

package tests

import "testing"

func TestFailing(t *testing.T) {
	t.Fail()
}

