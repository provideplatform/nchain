package dbconf

import (
	"testing"
)

type Values struct{}

func TestDatabaseConnection(t *testing.T) {
	values := Values{}
	DatabaseConnection().Select("SELECT 1", &values)
}
