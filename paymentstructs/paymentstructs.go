package structs

import (
	"fmt"
	"os"
	"strconv"
)

//PaymentRequest contains the data for validating a payment
type PaymentRequest struct {
	// Optional: the hash of the send transaction to confirm
	ValidationHash string `json:"validation_hash,omitempty"`
	// The Nano address where the payment is expected
	DestinationAddress string `json:"destination_address"`
	// The amount expected at the Destination Address
	Amount string `json:"amount"`
	// Worker ID for status reference
	WorkerID string `json:"worker_id"`
}

//Payment contains data on the payment during confirmation
type Payment struct {
	// Status for the payment: "confirming", "success", "error"
	Status string `json:"status"`
	// Hash of the transaction that completed the payment
	Hash string `json:"hash,omitempty"`
	// Error code if there is an error with the payment
	ErrorCode int `json:"error_code,omitempty"`
	// Error message if there is an error with the payment
	ErrorMessage string `json:"error_message,omitempty"`
	// Destination of the payment
	DestinationAddress string `json:"destination_address"`
	// Address that made the payment
	SendingAddress string `json:"sending_address"`
	// Expected amount of payment
	ExpectedAmount string `json:"expected_amount"`
	// Amount validated on transaction hash
	ValidatedAmount string `json:"validated_amount,omitempty"`
	// Worker ID for status reference
	WorkerID string `json:"worker_id"`
}

//Ack is sent to confirm receipt of the payment request
type Ack struct {
	// Destination account for the payment
	DestinationAddress string `json:"destination_address"`
	// Expected amount of payment
	ExpectedAmount string `json:"expected_amount"`
	// Worker ID for status reference
	WorkerID string
}

//Config retrieves the configuration variables from the config.json file
type Config struct {
	RPCHost           string
	RPCPort           string
	RedisHost         string
	RedisPort         string
	TimeoutDuration   int
	NanoWebsocketHost string
	NanoWebsocketPort string
}

func configEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

//LoadConfig returns a configuration structure populated with the ENV values provided.  If no ENV
//values are present, defaults are used.
func LoadConfig() Config {
	var configuration Config

	configuration.RPCHost = configEnv("RPCHOST", "http://[::1]")
	configuration.RPCPort = configEnv("RPCPORT", "55000")
	configuration.RedisHost = configEnv("REDISHOST", "localhost")
	configuration.RedisPort = configEnv("REDISPORT", "22000")
	var timeoutErr error
	configuration.TimeoutDuration, timeoutErr = strconv.Atoi(configEnv("TIMEOUTDURATION", "60"))
	if timeoutErr != nil {
		fmt.Println("Error converting timeout duration to int:", timeoutErr)
	}
	configuration.NanoWebsocketHost = configEnv("NANOWEBSOCKETHOST", "ws://[::1]")
	configuration.NanoWebsocketPort = configEnv("NANOWEBSOCKETPORT", "57000")

	return configuration
}
