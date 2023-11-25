package ws

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/XDagger/xdagpool/pool"
	"github.com/XDagger/xdagpool/util"
)

var Client *Socket

func NewClient(url string, ssl bool, msgChan chan pool.Message) *Socket {
	Client = New(url)

	Client.ConnectionOptions = ConnectionOptions{
		//Proxy: gowebsocket.BuildProxy("http://example.com"),
		UseSSL:         ssl,
		UseCompression: false,
	}

	Client.RequestHeader.Set("Accept-Encoding", "gzip, deflate, sdch")
	Client.RequestHeader.Set("Accept-Language", "en-US,en;q=0.8")
	Client.RequestHeader.Set("Pragma", "no-cache")
	Client.RequestHeader.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/49.0.2623.87 Safari/537.36")

	Client.OnConnectError = func(err error, socket Socket) {
		util.Error.Println("Recieved connect error ", err)
		go func() {
			time.Sleep(1000 * time.Millisecond)
			Client.Connect() //reconnection
		}()
	}
	Client.OnConnected = func(socket Socket) {
		util.Info.Println("Connected to server")

	}
	Client.OnTextMessage = func(message string, socket Socket) {
		var msg pool.Message
		err := json.Unmarshal([]byte(message), &msg)
		if err != nil {
			util.Error.Println("unmarshal message error", err)
		} else {
			msgChan <- msg
		}
		util.Info.Println("Recieved text message  " + message)
	}

	Client.OnBinaryMessage = func(message []byte, socket Socket) {
		var msg pool.Message
		err := json.Unmarshal(message, &msg)
		if err != nil {
			util.Error.Println("unmarshal message error", err)
		} else {
			msgChan <- msg
		}
		util.Info.Println("Recieved binary message  " + string(message))
	}

	Client.OnPingReceived = func(data string, socket Socket) {
		util.Info.Println("Recieved ping " + data)
	}
	Client.OnDisconnected = func(err error, socket Socket) {
		util.Info.Println("Disconnected from server ", err)

		go func() {
			time.After(1000 * time.Millisecond)
			Client.Connect() //reconnection
		}()
	}
	Client.Connect()

	return Client
}

type submitShare struct {
	Share string `json:"share"`
	Hash  string `json:"hash"`
	Index int    `json:"taskIndex"`
}

func Submit(jobHash, share string, taskIndex int) error {
	if Client.IsConnected {
		submit := submitShare{
			Share: share,
			Hash:  jobHash,
			Index: taskIndex,
		}
		data, _ := json.Marshal(submit)

		msg := pool.Message{
			MsgType:    2,
			MsgContent: data,
		}
		wsData, _ := json.Marshal(msg)
		Client.SendBinary(wsData)
		return nil
	}
	return errors.New("ws disconnected")
}
