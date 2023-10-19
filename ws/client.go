package ws

import (
	"crypto/tls"
	"errors"
	"log"

	"github.com/gorilla/websocket"

	"net/http"
	"net/url"

	"sync"
	"time"

	"github.com/XDagger/xdagpool/util"
)

// https://github.com/sacOO7/GoWebsocket
type Empty struct {
}

type Socket struct {
	Conn              *websocket.Conn
	WebsocketDialer   *websocket.Dialer
	Url               string
	ConnectionOptions ConnectionOptions
	RequestHeader     http.Header
	OnConnected       func(socket Socket)
	OnTextMessage     func(message string, socket Socket)
	OnBinaryMessage   func(data []byte, socket Socket)
	OnConnectError    func(err error, socket Socket)
	OnDisconnected    func(err error, socket Socket)
	OnPingReceived    func(data string, socket Socket)
	OnPongReceived    func(data string, socket Socket)
	IsConnected       bool
	Timeout           time.Duration
	sendMu            *sync.Mutex // Prevent "concurrent write to websocket connection"
	receiveMu         *sync.Mutex
}

type ConnectionOptions struct {
	UseCompression bool
	UseSSL         bool
	Proxy          func(*http.Request) (*url.URL, error)
	Subprotocols   []string
}

// todo Yet to be done
type ReconnectionOptions struct {
}

func New(url string) *Socket {
	return &Socket{
		Url:           url,
		RequestHeader: http.Header{},
		ConnectionOptions: ConnectionOptions{
			UseCompression: false,
			UseSSL:         true,
		},
		WebsocketDialer: &websocket.Dialer{},
		Timeout:         0,
		sendMu:          &sync.Mutex{},
		receiveMu:       &sync.Mutex{},
	}
}

func (socket *Socket) setConnectionOptions() {
	socket.WebsocketDialer.EnableCompression = socket.ConnectionOptions.UseCompression
	socket.WebsocketDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: socket.ConnectionOptions.UseSSL}
	socket.WebsocketDialer.Proxy = socket.ConnectionOptions.Proxy
	socket.WebsocketDialer.Subprotocols = socket.ConnectionOptions.Subprotocols
}

func (socket *Socket) Connect() {
	var err error
	var resp *http.Response
	socket.setConnectionOptions()

	socket.Conn, resp, err = socket.WebsocketDialer.Dial(socket.Url, socket.RequestHeader)

	if err != nil {
		util.Error.Println("Error while connecting to server ", err)
		if resp != nil {
			util.Error.Printf("HTTP Response %d status: %s", resp.StatusCode, resp.Status)
		}
		socket.IsConnected = false
		if socket.OnConnectError != nil {
			socket.OnConnectError(err, *socket)
		}
		return
	}

	util.Info.Println("Connected to server")

	if socket.OnConnected != nil {
		socket.IsConnected = true
		socket.OnConnected(*socket)
	}

	defaultPingHandler := socket.Conn.PingHandler()
	socket.Conn.SetPingHandler(func(appData string) error {
		util.Info.Println("Received PING from server")
		if socket.OnPingReceived != nil {
			socket.OnPingReceived(appData, *socket)
		}
		return defaultPingHandler(appData)
	})

	defaultPongHandler := socket.Conn.PongHandler()
	socket.Conn.SetPongHandler(func(appData string) error {
		util.Info.Println("Received PONG from server")
		if socket.OnPongReceived != nil {
			socket.OnPongReceived(appData, *socket)
		}
		return defaultPongHandler(appData)
	})

	defaultCloseHandler := socket.Conn.CloseHandler()
	socket.Conn.SetCloseHandler(func(code int, text string) error {
		result := defaultCloseHandler(code, text)
		util.Warn.Println("Disconnected from server ", result)
		if socket.OnDisconnected != nil {
			socket.IsConnected = false
			socket.OnDisconnected(errors.New(text), *socket)
		}
		return result
	})

	go func() {
		for {
			socket.receiveMu.Lock()
			if socket.Timeout != 0 {
				socket.Conn.SetReadDeadline(time.Now().Add(socket.Timeout))
			}
			messageType, message, err := socket.Conn.ReadMessage()
			socket.receiveMu.Unlock()
			if err != nil {
				util.Error.Println("read:", err)
				if socket.OnDisconnected != nil {
					socket.IsConnected = false
					socket.OnDisconnected(err, *socket)
				}
				return
			}
			util.Info.Printf("recv: %s", message)

			switch messageType {
			case websocket.TextMessage:
				if socket.OnTextMessage != nil {
					socket.OnTextMessage(string(message), *socket)
				}
			case websocket.BinaryMessage:
				if socket.OnBinaryMessage != nil {
					socket.OnBinaryMessage(message, *socket)
				}
			}
		}
	}()
}

func (socket *Socket) SendText(message string) {
	err := socket.send(websocket.TextMessage, []byte(message))
	if err != nil {
		util.Error.Println("write:", err)
		return
	}
}

func (socket *Socket) SendBinary(data []byte) {
	err := socket.send(websocket.BinaryMessage, data)
	if err != nil {
		util.Error.Println("write:", err)
		return
	}
}

func (socket *Socket) send(messageType int, data []byte) error {
	socket.sendMu.Lock()
	err := socket.Conn.WriteMessage(messageType, data)
	socket.sendMu.Unlock()
	return err
}

func (socket *Socket) Close() {
	err := socket.send(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		util.Error.Println("write close:", err)
	}
	socket.Conn.Close()
	if socket.OnDisconnected != nil {
		socket.IsConnected = false
		socket.OnDisconnected(err, *socket)
	}
}

func BuildProxy(Url string) func(*http.Request) (*url.URL, error) {
	uProxy, err := url.Parse(Url)
	if err != nil {
		log.Fatal("Error while parsing url ", err)
	}
	return http.ProxyURL(uProxy)
}
