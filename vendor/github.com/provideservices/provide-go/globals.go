package provide

import (
	"os"

	"github.com/kthomas/go-logger"
)

// Log global
var Log = logger.NewLogger("provide-go", getLogLevel(), true)

func getLogLevel() string {
	lvl := os.Getenv("LOG_LEVEL")
	if lvl == "" {
		lvl = "debug"
	}
	return lvl
}

func stringOrNil(str string) *string {
	if str == "" {
		return nil
	}
	return &str
}
