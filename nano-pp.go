package main

import (
	"encoding/json"
	"fmt"
	"log"
	bb "nano-pp/block_broadcaster"
	"nano-pp/nanoredis"
	structs "nano-pp/paymentstructs"
	workers "nano-pp/workers"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/adjust/rmq"
	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
)

//Consumer reads messages from the message queue
type Consumer struct {
	name   string
	count  int
	before time.Time
}

//readPaymentRequest converts a payload to a PaymentRequest struct
func readPaymentRequest(data string) structs.PaymentRequest {
	var paymentRequest structs.PaymentRequest
	d := json.NewDecoder(strings.NewReader(string(data)))
	d.Decode(&paymentRequest)
	fmt.Println("Received new payment request:", paymentRequest)

	return paymentRequest
}

//acknowledge sends an acknowledgement message to the middleware
func acknowledge(pool *redis.Pool, destinationAddress string, amount string, workerID uuid.UUID) {
	var ack structs.Ack
	ack.DestinationAddress = destinationAddress
	ack.ExpectedAmount = amount
	ack.WorkerID = workerID.String()

	data, err := json.Marshal(ack)
	if err != nil {
		fmt.Println("Error converting json for payment request:", err)
	}
	ackC := pool.Get()
	_, pubErr := ackC.Do("PUBLISH", fmt.Sprintf("ack.%s", destinationAddress), string(data))
	if pubErr != nil {
		fmt.Println("Error publishing acknowledgement:", pubErr)
	}
}

//newConsumer creates a new message consumer for the Redis message queue
func newConsumer(tag int) *Consumer {
	return &Consumer{
		name:   fmt.Sprintf("consumer %d", tag),
		count:  0,
		before: time.Now(),
	}
}

//Consume will pull a message from the request queue and start a new payment worker
func (consumer *Consumer) Consume(delivery rmq.Delivery) {
	consumer.count++
	consumer.before = time.Now()

	fmt.Printf("consumer %s processing request number %v", consumer.name, consumer.count)

	pool := nanoredis.NewPool()

	paymentRequest := readPaymentRequest(delivery.Payload())
	workerID := uuid.New()

	acknowledge(pool, paymentRequest.DestinationAddress, paymentRequest.Amount, workerID)

	go workers.PaymentRequestWorker(pool, paymentRequest, workerID.String())
}

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	config := structs.LoadConfig()

	ppID := uuid.New()

	pool := nanoredis.NewPool()
	defer pool.Close()

	rmqConn := rmq.OpenConnection("PaymentRequests", "tcp", fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort), 1)
	paymentQueue := rmqConn.OpenQueue("PaymentRequestQueue")

	paymentQueue.StartConsuming(10, 500*time.Millisecond)
	for i := 0; i < 3; i++ {
		paymentQueue.AddConsumer(fmt.Sprintf("%s-paymentworker", ppID.String()), newConsumer(i))
	}

	go bb.BlockBroadcaster()

	for {
		select {
		case <-interrupt:
			log.Println("Disconnecting from payment processor.")
			return
		}
	}

}
