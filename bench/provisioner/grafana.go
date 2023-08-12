package provisioner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func GetGrafanaBuildVersion(vm *VMInstance) (string, error) {
	grafanaSession, err := GetGrafanaSession(vm)

	targetURL := vm.HttpsServiceAddress() + "/api/frontend/settings"
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("Failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(grafanaSession)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Request failed: %w", err)
	}
	defer resp.Body.Close()

	//fmt.Println("Response:", responsePayload)

	settings := struct {
		BuildInfo struct {
			Version string `json:"version"`
		} `json:"buildInfo"`
	}{}

	err = json.NewDecoder(resp.Body).Decode(&settings)
	if err != nil {
		return "", fmt.Errorf("Failed to decode response: %w", err)
	}

	return settings.BuildInfo.Version, nil
}

// logs into grafana instance and returns a session cookie
func GetGrafanaSession(vm *VMInstance) (*http.Cookie, error) {
	loginURL := vm.HttpsServiceAddress() + "/login"

	loginPayload := struct {
		User     string `json:"user"`
		Password string `json:"password"`
	}{
		User:     vm.GrafanaUser,
		Password: vm.GrafanaPassword,
	}

	jsonPayload, err := json.Marshal(loginPayload)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal JSON payload: %w", err)
	}

	req, err := http.NewRequest("POST", loginURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("Failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Request failed: %w", err)
	}
	defer resp.Body.Close()

	//fmt.Printf("%#v", resp.Cookies()[0])

	// check response status code
	var responsePayload map[string]interface{}
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error loggin in: Response status code:", resp.StatusCode)
		err = json.NewDecoder(resp.Body).Decode(&responsePayload)
		if err != nil {
			return nil, fmt.Errorf("Failed to decode response: %w", err)
		}
		fmt.Println("Response:", responsePayload)
	}

	// get the build version
	return resp.Cookies()[0], nil
}
