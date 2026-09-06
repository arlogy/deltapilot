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

	testDBMu.Lock()
	defer testDBMu.Unlock()

	if testDBHandle == nil {
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
