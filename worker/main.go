package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
)

// RequestBody defines the structure of the JSON request
type RequestBody struct {
	Button string `json:"button"` // "camera", "microphone", or "usb"
	State  string `json:"state"`  // "on" or "off"
}

func main() {
	// Define the HTTP route with CORS middleware
	http.HandleFunc("/", enableCORS(handleRequest))

	// Configure TLS parameters

	server := &http.Server{
		Addr: "0.0.0.0:8080",
	}

	log.Println("Starting server...")

	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// enableCORS adds CORS headers and handles preflight OPTIONS requests
func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// handleRequest processes incoming HTTP requests
func handleRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Println("got request")
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Decode the JSON request body
	var reqBody RequestBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Handle the button and state
	switch reqBody.Button {
	case "camera":
		err = toggleDevice("camera", reqBody.State)
	case "microphone":
		err = toggleDevice("microphone", reqBody.State)
	case "usb":
		err = toggleDevice("usb", reqBody.State)
	default:
		http.Error(w, "Invalid button", http.StatusBadRequest)
		return
	}

	// Check for errors
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to toggle device: %v", err), http.StatusInternalServerError)
		return
	}

	// Respond with success
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("%s turned %s", reqBody.Button, reqBody.State)))
}

// toggleDevice enables or disables the specified device (unchanged)
func toggleDevice(device string, state string) error {
	var command string

	switch device {
	case "camera":
		if state == "on" {
			command = "pnputil /enable-device \"Camera\""
		} else {
			command = "pnputil /disable-device \"Camera\""
		}
	case "microphone":
		if state == "on" {
			command = "pnputil /enable-device \"Microphone\""
		} else {
			command = "pnputil /disable-device \"Microphone\""
		}
	case "usb":
		if state == "on" {
			command = "powershell -Command Enable-PnpDevice -InstanceId (Get-PnpDevice -Class USB | Where-Object {$_.FriendlyName -like '*USB Root Hub*'}).InstanceId -Confirm:$false"
		} else {
			command = "powershell -Command Disable-PnpDevice -InstanceId (Get-PnpDevice -Class USB | Where-Object {$_.FriendlyName -like '*USB Root Hub*'}).InstanceId -Confirm:$false"
		}
	default:
		return fmt.Errorf("invalid device: %s", device)
	}

	cmd := exec.Command("cmd", "/C", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %s, output: %s", err, string(output))
	}

	return nil
}
