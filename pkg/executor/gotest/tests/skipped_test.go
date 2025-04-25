// +build goexecutor

package tests

import "testing"

func TestSkipped(t *testing.T) {
	t.Skip("skipping")
}