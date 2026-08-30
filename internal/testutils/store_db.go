package testutils

import (
	"sync"
	"testing"

	"github.com/arlogy/deltapilot/internal/dbclient"
)

var (
	testDBHandle *dbclient.DBHandle
	testDBMu     sync.Mutex
)

func InitDB(t *testing.T) *dbclient.DBHandle {
	t.Helper()

	if testDBHandle == nil {
		testDBMu.Lock()
		defer testDBMu.Unlock()

		handle, err := dbclient.ConnectFromEnv()
		if err != nil {
			t.Fatal(err)
		}

		if err := handle.ConfigureDBPool(10, 10); err != nil {
			t.Fatal(err)
		}

		testDBHandle = handle
	}

	return testDBHandle
}
