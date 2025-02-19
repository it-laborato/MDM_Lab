package allmulti

import (
	"context"

	"github.com/it-laborato/MDM_Lab/server/mdm/nanomdm/mdm"
	"github.com/it-laborato/MDM_Lab/server/mdm/nanomdm/storage"
)

func (ms *MultiAllStorage) RetrievePushInfo(ctx context.Context, ids []string) (map[string]*mdm.Push, error) {
	val, err := ms.execStores(ctx, func(s storage.AllStorage) (interface{}, error) {
		return s.RetrievePushInfo(ctx, ids)
	})
	return val.(map[string]*mdm.Push), err
}
