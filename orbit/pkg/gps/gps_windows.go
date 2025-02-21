//go:build windows

package gps

import (
	"fmt"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// GPSData содержит координаты.
type GPSData struct {
	Latitude  float64
	Longitude float64
}

// GetGPSCoordinates инициализирует COM и получает координаты с первого доступного GPS‑датчика.
func GetGPSCoordinates() (*GPSData, error) {
	// Инициализируем COM.
	if err := ole.CoInitialize(0); err != nil {
		return nil, fmt.Errorf("CoInitialize error: %v", err)
	}
	// Важно вызывать CoUninitialize в конце.
	defer ole.CoUninitialize()

	// Создаём объект менеджера сенсоров.
	unknown, err := oleutil.CreateObject("Sensors.SensorManager")
	if err != nil {
		return nil, fmt.Errorf("CreateObject error: %v", err)
	}
	sensorManager, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, fmt.Errorf("QueryInterface error: %v", err)
	}
	defer sensorManager.Release()

	// GUID для датчика GPS (SENSOR_TYPE_LOCATION_GPS). Обычно он равен:
	const sensorTypeGPS = "{ED4EED84-874F-42E8-A9B0-D6555F9A72FD}"
	sensorsVariant, err := oleutil.CallMethod(sensorManager, "GetSensorsByCategory", sensorTypeGPS)
	if err != nil {
		return nil, fmt.Errorf("CallMethod GetSensorsByCategory error: %v", err)
	}
	sensors := sensorsVariant.ToIDispatch()
	defer sensors.Release()

	// Получаем количество доступных сенсоров.
	countVariant, err := oleutil.GetProperty(sensors, "Count")
	if err != nil {
		return nil, fmt.Errorf("GetProperty Count error: %v", err)
	}
	count := int(countVariant.Val)
	if count == 0 {
		return nil, fmt.Errorf("no GPS sensor found")
	}

	// Берём первый сенсор из коллекции.
	sensorVariant, err := oleutil.CallMethod(sensors, "Item", 0)
	if err != nil {
		return nil, fmt.Errorf("CallMethod Item error: %v", err)
	}
	sensor := sensorVariant.ToIDispatch()
	defer sensor.Release()

	// Получаем объект данных с сенсора.
	dataVariant, err := oleutil.CallMethod(sensor, "GetData")
	if err != nil {
		return nil, fmt.Errorf("CallMethod GetData error: %v", err)
	}
	data := dataVariant.ToIDispatch()
	defer data.Release()

	// Извлекаем свойства Latitude и Longitude.
	latVariant, err := oleutil.GetProperty(data, "Latitude")
	if err != nil {
		return nil, fmt.Errorf("GetProperty Latitude error: %v", err)
	}
	longVariant, err := oleutil.GetProperty(data, "Longitude")
	if err != nil {
		return nil, fmt.Errorf("GetProperty Longitude error: %v", err)
	}

	gpsData := &GPSData{
		Latitude:  float64(latVariant.Val),
		Longitude: float64(longVariant.Val),
	}
	return gpsData, nil
}
