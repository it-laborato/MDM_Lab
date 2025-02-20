package offline

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"time"
)

// Policies представляет структуру политик, полученных с сервера.
type Policies struct {
	// Здесь можно определить конкретные поля политик.
	// Например:
	Config map[string]interface{} `json:"config"`
	// Время последнего обновления.
	LastUpdated time.Time `json:"last_updated"`
}

// CachePolicies сохраняет политики в указанный файл.
func CachePolicies(filePath string, policies *Policies) error {
	data, err := json.MarshalIndent(policies, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filePath, data, 0644)
}

// LoadCachedPolicies загружает политики из файла.
func LoadCachedPolicies(filePath string) (*Policies, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var policies Policies
	if err := json.Unmarshal(data, &policies); err != nil {
		return nil, err
	}
	return &policies, nil
}

// CheckServerOnline пытается выполнить GET-запрос к серверу и возвращает true,
// если сервер доступен (HTTP 200), и false в противном случае.
func CheckServerOnline(serverURL string, timeout time.Duration) bool {
	client := http.Client{Timeout: timeout}
	resp, err := client.Get(serverURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// GetPoliciesFromServer – пример функции для получения политик с сервера.
// В реальном проекте здесь будет вызываться API вашего сервера.
func GetPoliciesFromServer(serverURL string) (*Policies, error) {
	// Пример: выполняем GET-запрос и парсим JSON.
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(serverURL + "/api/latest/mdmlab/policies")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("server returned non-OK status")
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var policies Policies
	if err := json.Unmarshal(data, &policies); err != nil {
		return nil, err
	}
	policies.LastUpdated = time.Now()
	return &policies, nil
}
