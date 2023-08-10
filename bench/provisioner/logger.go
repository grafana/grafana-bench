package provisioner

import (
	"log/slog"
	"os"
)

var log = slog.New(slog.NewTextHandler(os.Stderr, nil))
