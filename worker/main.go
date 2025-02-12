package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
)

type RequestBody struct {
	Button string `json:"button"` // "camera", "microphone", "usb", or "reboot"
	State  string `json:"state"`  // "on", "off", or "" (for reboot)
}

func main() {
	http.HandleFunc("/", enableCORS(handleRequest))

	server := &http.Server{
		Addr: "0.0.0.0:8080",
	}

	log.Println("Starting admin device control server on :8080...")
	log.Fatal(server.ListenAndServe())
}

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

func handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody RequestBody
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validateRequest(reqBody); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if reqBody.Button == "reboot" {
		if err := rebootSystem(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if err := toggleDevice(reqBody.Button, reqBody.State); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Operation successful: %s", reqBody.Button)
}

func validateRequest(req RequestBody) error {
	validButtons := map[string]bool{"camera": true, "microphone": true, "usb": true, "reboot": true}
	validStates := map[string]bool{"on": true, "off": true}

	if !validButtons[req.Button] {
		return fmt.Errorf("invalid device: %s", req.Button)
	}

	if req.Button != "reboot" && !validStates[req.State] {
		return fmt.Errorf("invalid state: %s", req.State)
	}

	return nil
}

func toggleDevice(device, state string) error {
	var cmd *exec.Cmd

	switch device {
	case "camera":
		action := "Disable"
		if state == "on" {
			action = "Enable"
		}
		cmd = exec.Command("powershell", "-Command",
			fmt.Sprintf("Get-PnpDevice -Class Camera | %s-PnpDevice -Confirm:$false", action))

	case "microphone":
		action := "Disable"
		if state == "on" {
			action = "Enable"
		}
		cmd = exec.Command("powershell", "-Command",
			fmt.Sprintf("Get-PnpDevice -Class Media | %s-PnpDevice -Confirm:$false", action))

	case "usb":
		action := "Disable"
		if state == "on" {
			action = "Enable"
		}
		cmd = exec.Command("powershell", "-Command",
			fmt.Sprintf("Get-PnpDevice -Class USB | %s-PnpDevice -Confirm:$false", action))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s command failed: %v\nOutput: %s", device, err, string(output))
	}
	return nil
}

func rebootSystem() error {
	// Immediate forced reboot command for Windows
	cmd := exec.Command("cmd", "/C", "shutdown /r /f /t 0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("reboot failed: %v\nOutput: %s", err, string(output))
	}
	return nil
}
