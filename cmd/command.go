package cmd

import (
	"context"
)

// Command defines the interface for executing commands
type Command interface {
	Exec(context.Context) error
}