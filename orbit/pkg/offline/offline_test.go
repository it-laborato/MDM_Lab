package offline

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheAndLoadPolicies(t *testing.T) {
	// Создаём временный каталог для кэша.
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, "policies.json")

	// Формируем пример политик.
	policies := &Policies{
		Config: map[string]interface{}{
			"policy1": "value1",
			"policy2": float64(42),
		},
		LastUpdated: time.Now(),
	}

	// Кэшируем политики.
	err := CachePolicies(cacheFile, policies)
	require.NoError(t, err, "CachePolicies не должен возвращать ошибку")

	// Загружаем политики из кэша.
	loaded, err := LoadCachedPolicies(cacheFile)
	require.NoError(t, err, "LoadCachedPolicies не должен возвращать ошибку")
	assert.Equal(t, policies.Config, loaded.Config, "Загруженные данные должны совпадать с кэшированными")
	assert.WithinDuration(t, policies.LastUpdated, loaded.LastUpdated, time.Second, "Временные метки должны быть близки")
}

func TestCheckServerOnline(t *testing.T) {
	// Создаём тестовый HTTP-сервер, возвращающий 200 OK.
	tsOK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer tsOK.Close()

	online := CheckServerOnline(tsOK.URL, 2*time.Second)
	assert.True(t, online, "Сервер должен считаться доступным")

	// Сервер, возвращающий ошибку (например, 500 Internal Server Error)
	tsErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer tsErr.Close()

	online = CheckServerOnline(tsErr.URL, 2*time.Second)
	assert.False(t, online, "Сервер с кодом ошибки должен считаться недоступным")

	// Проверка для невалидного URL.
	online = CheckServerOnline("http://invalid.local", 2*time.Second)
	assert.False(t, online, "Невалидный URL должен возвращать false")
}

func TestGetPoliciesFromServer_Success(t *testing.T) {
	// Формируем ожидаемые политики.
	expectedPolicies := &Policies{
		Config: map[string]interface{}{
			"policyA": "enabled",
		},
	}
	data, err := json.Marshal(expectedPolicies)
	require.NoError(t, err)

	// Создаём тестовый HTTP-сервер, который по пути "/api/latest/mdmlab/policies" возвращает корректный JSON.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/latest/mdmlab/policies" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))
	defer ts.Close()

	// Вызываем функцию получения политик.
	policies, err := GetPoliciesFromServer(ts.URL)
	require.NoError(t, err, "Получение политик не должно завершаться ошибкой")
	// LastUpdated устанавливается автоматически, поэтому проверяем только Config.
	assert.Equal(t, expectedPolicies.Config, policies.Config, "Конфигурация должна совпадать")
}

func TestGetPoliciesFromServer_NonOK(t *testing.T) {
	// Создаём сервер, возвращающий не-200 статус.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	_, err := GetPoliciesFromServer(ts.URL)
	require.Error(t, err, "Функция должна возвращать ошибку при неверном статусе")
}

func TestGetPoliciesFromServer_InvalidJSON(t *testing.T) {
	// Создаём сервер, возвращающий некорректный JSON.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer ts.Close()

	_, err := GetPoliciesFromServer(ts.URL)
	require.Error(t, err, "Функция должна возвращать ошибку при невалидном JSON")
}
