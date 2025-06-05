// +build goexecutor

package tests

import (
	"testing"
	"time"
)

func TestPassing1(t *testing.T) {
	time.Sleep(time.Millisecond * 20)
}

func TestPassing2(t *testing.T) {
	time.Sleep(time.Millisecond * 60)
}

func TestPassing3(t *testing.T) {
	t.Run("SubTest1", func (t *testing.T) {
		time.Sleep(time.Millisecond * 10)
	})

	t.Run("SubTest2", func (t *testing.T) {
		time.Sleep(time.Millisecond * 10)
	})
}