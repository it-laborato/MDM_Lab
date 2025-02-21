package osquery_utils

import (
	"context"
	"fmt"

	"github.com/go-kit/log"
	"github.com/it-laborato/MDM_Lab/orbit/pkg/gps"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
)

// EmptyToZero Sometimes osquery gives us empty string where we expect an integer.
// We change the to "0" so it can be handled by the appropriate string to
// integer conversion function, as these will err on ""
func EmptyToZero(val string) string {
	if val == "" {
		return "0"
	}
	return val
}

// directIngestGPSModule получает данные с GPS и обновляет информацию в host.
func directIngestGPSModule(ctx context.Context, logger log.Logger, host *mdmlab.Host, ds mdmlab.Datastore, rows []map[string]string) error {
	// Здесь можно игнорировать rows, т.к. мы собираем данные напрямую через COM.
	gpsData, err := gps.GetGPSCoordinates()
	if err != nil {
		logger.Log("error", fmt.Sprintf("failed to get GPS data: %v", err))
		return err
	}

	host.GPSModule = &mdmlab.HostGPSModule{
		Latitude:  gpsData.Latitude,
		Longitude: gpsData.Longitude,
	}

	// Сохраните информацию о GPS через Datastore.
	return ds.SetOrUpdateHostGPSModule(ctx, host.ID, host.GPSModule)
}
