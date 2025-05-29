package commands

import (
	"os"

	"nos3/pkg/logger"
)

func ExitOnError(err error) {
	logger.Error("nos3 error", "err", err.Error())
	os.Exit(1)
}
