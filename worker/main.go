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
	// Define the HTTP route
	http.HandleFunc("/", handleRequest)

	// Load certificate and key
	// cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	// if err != nil {
	// 	log.Fatal("Failed to load certificate:", err)
	// }
	//
	// // Configure TLS
	// tlsConfig := &tls.Config{
	// 	Certificates: []tls.Certificate{cert},
	// 	MinVersion:   tls.VersionTLS12, // Enforce TLS 1.2 or higher
	// 	CipherSuites: []uint16{
	// 		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	// 		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	// 	},
	// }

	server := &http.Server{
		Addr: ":8080",
		// TLSConfig: tlsConfig,
	}

	log.Fatal(server.ListenAndServe())
}

// handleRequest processes incoming HTTP requests
func handleRequest(w http.ResponseWriter, r *http.Request) {
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

// toggleDevice enables or disables the specified device
func toggleDevice(device string, state string) error {
	var command string

	// Determine the command based on the device and state
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

	// Execute the command
	cmd := exec.Command("cmd", "/C", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %s, output: %s", err, string(output))
	}

	return nil
}
