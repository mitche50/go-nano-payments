package bb

import (
	"encoding/json"
	"fmt"
	"log"
	structs "nano-pp/paymentstructs"
	"os"
	"os/signal"

	"github.com/gomodule/redigo/redis"
	"github.com/sacOO7/gowebsocket"
)

//WebsocketMessage contains the Topic, Time and Message sent to the websocket client
type WebsocketMessage struct {
	Topic   string      `json:"topic"`
	Time    string      `json:"time"`
	Message NodeMessage `json:"message"`
}

//NodeMessage is the contents of the message sent from the Nano node.
type NodeMessage struct {
	Account string `json:"account"`
	Amount  string `json:"amount"`
	Hash    string `json:"hash"`
	Block   Block  `json:"block"`
}

//Block is the information for the provided block
type Block struct {
	Type           string `json:"state"`
	Account        string `json:"account"`
	Previous       string `json:"previous"`
	Representative string `json:"representative"`
	Balance        string `json:"balance"`
	Link           string `json:"link"`
	LinkAsAccount  string `json:"link_as_account"`
	Work           string `json:"work"`
	Subtype        string `json:"subtype"`
}

//BlockBroadcaster listens to a webhook and broadcasts the blocks
func BlockBroadcaster() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	config := structs.LoadConfig()
	fmt.Println("websocket host:", config.NanoWebsocketHost)
	fmt.Println("websocket port:", config.NanoWebsocketPort)

	socket := gowebsocket.New(fmt.Sprintf("%s:%s", config.NanoWebsocketHost, config.NanoWebsocketPort))
	c, err := redis.Dial("tcp", fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort))
	if err != nil {
		fmt.Println("Error connecting to redis for Block Broadcaster:", err)
	}
	defer c.Close()

	socket.OnConnected = func(socket gowebsocket.Socket) {
		log.Println("Connected to nano websocket")
		data := map[string]string{"action": "subscribe", "topic": "confirmation"}
		dataJSON, _ := json.Marshal(data)
		socket.SendBinary(dataJSON)
	}

	socket.OnTextMessage = func(message string, socket gowebsocket.Socket) {
		_, err := c.Do("PUBLISH", "nano-websocket-confirmations", string(message))
		if err != nil {
			log.Println("error in publishing:", err)
		}
		log.Println("block published to nano-websocket-confirmations")
	}

	socket.OnConnectError = func(err error, socket gowebsocket.Socket) {
		fmt.Println("Error connecting to websocket:", err)
	}

	socket.Connect()

	for {
		select {
		case <-interrupt:
			log.Println("Disconnecting from websocket.")
			return
		}
	}
}
