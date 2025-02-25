package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/lunnik9/rdp/rdp"
	"github.com/lunnik9/rdp/rdp/pdu"
)

const (
	wsReadBufferSize  = 8192
	wsWriteBufferSize = 8192 * 2
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  wsReadBufferSize,
	WriteBufferSize: wsWriteBufferSize,
	// Разрешаем все источники – для разработки. В продакшене нужно настроить проверку.
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	// Если React-приложение развёрнуто отдельно,
	// здесь нам нужен только endpoint для WebSocket-соединения.
	http.HandleFunc("/connect", connectHandler)

	log.Println("Backend-сервер запущен на :4000")
	if err := http.ListenAndServe(":4000", nil); err != nil {
		log.Fatal(err)
	}
}

// connectHandler устанавливает WebSocket-соединение и подключается к RDP-серверу.
func connectHandler(w http.ResponseWriter, r *http.Request) {
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Ошибка апгрейда websocket: %v", err)
		return
	}
	defer wsConn.Close()

	// Создаем контекст для контроля жизненного цикла соединения
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Читаем параметры подключения из URL-запроса
	width, err := strconv.Atoi(r.URL.Query().Get("width"))
	if err != nil {
		log.Printf("Ошибка получения width: %v", err)
		return
	}
	height, err := strconv.Atoi(r.URL.Query().Get("height"))
	if err != nil {
		log.Printf("Ошибка получения height: %v", err)
		return
	}
	host := r.URL.Query().Get("host")
	user := r.URL.Query().Get("user")
	password := r.URL.Query().Get("password")
	if host == "" || user == "" || password == "" {
		log.Println("Не заданы параметры подключения (host, user, password)")
		return
	}
	// Если порт не указан, добавляем стандартный порт 3389
	if !strings.Contains(host, ":") {
		host += ":3389"
	}

	// Создаем RDP-клиента
	rdpClient, err := rdp.NewClient(host, user, password, width, height)
	if err != nil {
		log.Printf("Ошибка создания RDP клиента: %v", err)
		return
	}
	defer rdpClient.Close()

	// Подключаемся к RDP-серверу
	if err := rdpClient.Connect(); err != nil {
		log.Printf("Ошибка подключения к RDP серверу: %v", err)
		return
	}
	log.Println("RDP-соединение установлено")

	// Запускаем две горутины:
	// 1. Для передачи сообщений от WS к RDP
	go wsToRdp(ctx, wsConn, rdpClient, cancel)
	// 2. Для пересылки обновлений от RDP к WS
	rdpToWs(ctx, wsConn, rdpClient)
}

// wsToRdp читает сообщения от React-клиента и отправляет их в RDP.
func wsToRdp(ctx context.Context, wsConn *websocket.Conn, rdpClient *rdp.Client, cancel context.CancelFunc) {
	defer func() {
		log.Println("wsToRdp завершен")
		cancel()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, data, err := wsConn.ReadMessage()
			if err != nil {
				// Если ошибка связана с нормальным закрытием соединения – просто завершаем горутину.
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					log.Println("WebSocket закрыт (нормальное закрытие):", err)
				} else {
					log.Println(fmt.Errorf("Ошибка чтения из WS: %w", err))
				}
				return
			}
			if err := rdpClient.SendInputEvent(data); err != nil {
				log.Printf("Ошибка отправки ввода в RDP: %v", err)
				return
			}
		}
	}
}

// rdpToWs получает обновления от RDP и пересылает их в React-клиент через WS.
func rdpToWs(ctx context.Context, wsConn *websocket.Conn, rdpClient *rdp.Client) {
	defer func() {
		log.Println("rdpToWs завершен")
	}()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			upd, err := rdpClient.GetUpdate()
			switch {
			case err == nil:
				// все ок
			case errors.Is(err, pdu.ErrDeactiateAll):
				log.Println("RDP: деактивация обновлений")
				return
			default:
				log.Printf("Ошибка получения обновления от RDP: %v", err)
				return
			}
			if upd == nil {
				continue
			}
			if err := wsConn.WriteMessage(websocket.BinaryMessage, upd.Data); err != nil {
				if err == websocket.ErrCloseSent {
					log.Println("WebSocket закрыт, отправка невозможна")
					return
				}
				log.Printf("Ошибка отправки сообщения в WS: %v", err)
				return
			}
		}
	}
}
