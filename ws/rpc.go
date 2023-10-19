package ws

import (
	"github.com/XDagger/xdagpool/util"
)

var WsClient *Socket

func NewRpcClient(url string) {
	WsClient = New(url)

	WsClient.ConnectionOptions = ConnectionOptions{
		//Proxy: gowebsocket.BuildProxy("http://example.com"),
		UseSSL:         false,
		UseCompression: false,
	}

	WsClient.RequestHeader.Set("Accept-Encoding", "gzip, deflate, sdch")
	WsClient.RequestHeader.Set("Accept-Language", "en-US,en;q=0.8")
	WsClient.RequestHeader.Set("Pragma", "no-cache")
	WsClient.RequestHeader.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/49.0.2623.87 Safari/537.36")

	WsClient.OnConnectError = func(err error, socket Socket) {
		util.Error.Println("Recieved connect error ", err)
	}
	WsClient.OnConnected = func(socket Socket) {
		util.Info.Println("Connected to server")

	}
	WsClient.OnTextMessage = func(message string, socket Socket) {
		util.Info.Println("Recieved message  " + message)
	}
	WsClient.OnPingReceived = func(data string, socket Socket) {
		util.Info.Println("Recieved ping " + data)
	}
	WsClient.OnDisconnected = func(err error, socket Socket) {
		util.Info.Println("Disconnected from server ", err)

		go WsClient.Connect() //reconnection
	}
	WsClient.Connect()
}
