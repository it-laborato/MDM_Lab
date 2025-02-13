package main

import (
	"github.com/gorilla/websocket"
	"gocv.io/x/gocv"
)

func main() {
	// Windows cameras typically use index 0
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

	// Set camera properties for Windows compatibility
	webcam.Set(gocv.VideoCaptureFrameWidth, 640)
	webcam.Set(gocv.VideoCaptureFrameHeight, 480)

	for {
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
