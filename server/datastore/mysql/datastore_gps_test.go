package mysql

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-kit/log"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestSetOrUpdateHostGPSModule_Update(t *testing.T) {
	// Создаём sqlmock DB
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Оборачиваем sql.DB в sqlx.DB
	sqlxDB := sqlx.NewDb(db, "mysql")

	// Создаём экземпляр Datastore с основным соединением
	ds := &Datastore{
		primary: sqlxDB,
		logger:  log.NewNopLogger(),
	}

	ctx := context.Background()
	hostID := uint(42)
	gpsModule := &mdmlab.HostGPSModule{
		Latitude:  55.7558, // например, Москва
		Longitude: 37.6173,
	}

	// Ожидаем выполнение запроса для обновления данных GPS:
	// "UPDATE hosts SET gps_latitude = ?, gps_longitude = ? WHERE id = ?"
	mock.ExpectExec(`UPDATE hosts SET gps_latitude = \?, gps_longitude = \? WHERE id = \?`).
		WithArgs(gpsModule.Latitude, gpsModule.Longitude, hostID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = ds.SetOrUpdateHostGPSModule(ctx, hostID, gpsModule)
	require.NoError(t, err)

	// Проверяем, что все ожидания sqlmock выполнены
	err = mock.ExpectationsWereMet()
	require.NoError(t, err)
}

func TestSetOrUpdateHostGPSModule_Clear(t *testing.T) {
	// Создаём sqlmock DB
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Оборачиваем sql.DB в sqlx.DB
	sqlxDB := sqlx.NewDb(db, "mysql")

	// Создаём экземпляр Datastore
	ds := &Datastore{
		primary: sqlxDB,
		logger:  log.NewNopLogger(),
	}

	ctx := context.Background()
	hostID := uint(42)

	// Ожидаем выполнение запроса для сброса GPS (module == nil):
	// "UPDATE hosts SET gps_latitude = NULL, gps_longitude = NULL WHERE id = ?"
	mock.ExpectExec(`UPDATE hosts SET gps_latitude = NULL, gps_longitude = NULL WHERE id = \?`).
		WithArgs(hostID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = ds.SetOrUpdateHostGPSModule(ctx, hostID, nil)
	require.NoError(t, err)

	// Проверяем, что все ожидания sqlmock выполнены
	err = mock.ExpectationsWereMet()
	require.NoError(t, err)
}
