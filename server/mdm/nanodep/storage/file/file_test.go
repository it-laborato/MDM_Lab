package file

import (
	"testing"

	"github.com:it-laborato/MDM_Lab/server/mdm/nanodep/storage"
	"github.com:it-laborato/MDM_Lab/server/mdm/nanodep/storage/storagetest"
)

func TestFileStorage(t *testing.T) {
	storagetest.Run(t, func(t *testing.T) storage.AllDEPStorage {
		s, err := New(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		return s
	})
}
