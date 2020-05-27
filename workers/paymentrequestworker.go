package workers

import (
	"encoding/json"
	"fmt"
	"math/big"
	bb "nano-pp/block_broadcaster"
	br "nano-pp/block_recorder"
	nano "nano-pp/nanocurrency"
	"nano-pp/nanocurrency/nanostructs"
	nanostruct "nano-pp/nanocurrency/nanostructs"
	structs "nano-pp/paymentstructs"
	"strconv"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
)

func getKnownBlocks(pool *redis.Pool, destinationAddress string) []string {
	//Known hashes include the past 1000 blocks and any pending blocks (including active)
	br.BlockRecorder(pool, destinationAddress)
	hashC := pool.Get()
	hashReturn, membersErr := hashC.Do("SMEMBERS", fmt.Sprintf("known_pending/%s", destinationAddress))
	if membersErr != nil {
		fmt.Println("error retrieving known/pending hashes from redis:", membersErr)
	}

	hashes, _ := redis.Strings(hashReturn, nil)

	return hashes
}

func getPendingBlocks(rpc nanostructs.NanoRPC, destinationAddress string) nanostruct.Pending {
	//getPendingBlocks is used to check for any new pending blocks periodically to resubmit for
	//confirmation.
	fmt.Println("checking for pending from pollPending for address:", destinationAddress)
	optionalPending := map[string]string{"include_active": "true"}
	pendingResponse, pendingErr := nano.Pending(rpc, destinationAddress, optionalPending)
	if pendingErr != nil {
		fmt.Printf("Error retrieving pending blocks: %v\n", pendingErr)
	}

	var pending nanostruct.Pending
	d := json.NewDecoder(strings.NewReader(string(pendingResponse)))
	d.Decode(&pending)

	return pending
}

func setPendingHashMap(hashes []string) map[string]bool {
	//setPendingHashMap sets a map with the existing hashes for ease of reference.
	hashCheck := make(map[string]bool)

	for _, hash := range hashes {
		hashCheck[hash] = true
	}

	return hashCheck
}

func getConfirmationHeight(rpc nanostructs.NanoRPC, hash string) string {
	//getConfirmationHeight will return the confirmation height for a provided hash
	var blockInfo nanostruct.BlockInfo
	blockReturn, err := nano.BlockInfo(rpc, hash)
	if err != nil {
		fmt.Println("Error getting info for confirmation height:", err)
	}
	d := json.NewDecoder(strings.NewReader(string(blockReturn)))
	d.Decode(&blockInfo)

	return blockInfo.Height
}

func convertPaymentAmounts(expected string, received string) (*big.Int, *big.Int) {
	//convertPaymentAmounts will convert strings into big.Ints so there is no data loss in raw
	expectedInt := new(big.Int)
	receivedInt := new(big.Int)
	if _, ok := expectedInt.SetString(expected, 10); !ok {
		fmt.Println("error setting big int for expected")
	}
	if _, ok := receivedInt.SetString(received, 10); !ok {
		fmt.Println("error setting big int for received")
	}

	return expectedInt, receivedInt
}

func calcDifference(amountComparison int, expected string, received string) *big.Int {
	//calcDifference returns the difference of the expected payment and the received payment
	//This return is always positive.
	expectedInt, receivedInt := convertPaymentAmounts(expected, received)
	var difference *big.Int
	if amountComparison == -1 {
		difference = big.NewInt(0).Sub(receivedInt, expectedInt)
	} else {
		difference = big.NewInt(0).Sub(expectedInt, receivedInt)
	}
	return difference
}

func sendConfirmation(confirming structs.Payment, destinationAddress string, pool *redis.Pool) {
	//sendConfirmation converts a payment to a JSON string and publishes over redis.
	confirmJSON, confirmErr := json.Marshal(confirming)
	if confirmErr != nil {
		fmt.Println("Error converting confirmation to JSON:", confirmErr)
	}
	confirmC := pool.Get()
	_, err := confirmC.Do("PUBLISH", fmt.Sprintf("payment.%s", destinationAddress), string(confirmJSON))
	if err != nil {
		fmt.Println("Error posting payment confirmation", err)
	}
	confirmC.Close()
}

func parseWebhookMessage(message []byte) bb.WebsocketMessage {
	//parseWebhookMessage converts the webhook message sent from the Nano node to a WebsocketMessage struct.
	var websocketJSON bb.WebsocketMessage
	d := json.NewDecoder(strings.NewReader(string(message)))
	d.Decode(&websocketJSON)

	return websocketJSON
}

func markConfirming(pool *redis.Pool, hash string, destinationAddress string) {
	//markConfirming sets the key of the worker to "confirming".  Used to prevent prematurely closing payment's
	//status as failed while a transaction is still pending.
	confC := pool.Get()
	_, confErr := confC.Do("SET", fmt.Sprintf("confirming/%s", destinationAddress), "confirming")
	if confErr != nil {
		fmt.Println("Error marking the block as confirming.")
	}
}

func setWorkerStatus(status string, workerID string, pool *redis.Pool) {
	//setWorkerStatus sets a status in redis to allow for status checks of a specific worker.
	statusC := pool.Get()
	_, delErr := statusC.Do("DEL", fmt.Sprintf("status/%s", workerID))
	if delErr != nil {
		fmt.Println("Error clearing the worker status:", delErr)
	}
	_, statusErr := statusC.Do("SET", fmt.Sprintf("status/%s", workerID), status)
	if statusErr != nil {
		fmt.Println("Error updating the worker status:", statusErr)
	}
	fmt.Printf("Set the worker status for worker %s to %s\n", workerID, status)
}

func pollPending(paymentRequest structs.PaymentRequest, rpc nanostructs.NanoRPC, hashCheck map[string]bool, pool *redis.Pool, workerID string) {
	//pollPending will periodically poll the RPC for new pending blocks for a provided account.  If there is a completed
	//transaction in the meantime, it will cancel.
	pendingC := pool.Get()
	fmt.Printf("subscribing to cancel/%s\n", workerID)
	psc := redis.PubSubConn{Conn: pendingC}
	if err := psc.Subscribe(fmt.Sprintf("cancel/%s", workerID)); err != nil {
		fmt.Println("Error subscribing to cancellations:", err)
	}
	if err := psc.Subscribe(fmt.Sprintf("resetPending/%s", workerID)); err != nil {
		fmt.Println("Error subscribing to resetPending:", err)
	}

	cancelChan := make(chan bool)

	go pendingTimerCheck(pool, paymentRequest, hashCheck, cancelChan, workerID, rpc)

	for {
		switch v := psc.ReceiveWithTimeout(time.Duration(20 * time.Second)).(type) {
		case redis.Message:
			if v.Channel == fmt.Sprintf("cancel/%s", workerID) {
				cancelChan <- true
				return
			}
		}
	}
}

func pendingTimerCheck(pool *redis.Pool, paymentRequest structs.PaymentRequest, hashCheck map[string]bool, cancelChan chan bool, workerID string, rpc nanostructs.NanoRPC) {
	//pendingTimerCheck is a non-blocking goroutine which will periodically check for new pending hashes for a provided
	//account.
	var confirming structs.Payment

	timerC := pool.Get()
	pendingTimer := time.NewTicker(5 * time.Second)
	created := time.Now()
	for {
		select {
		case <-cancelChan:
			fmt.Println("Cancelling")
			pendingTimer.Stop()
			return
		case <-pendingTimer.C:
			_, err := timerC.Do("PUBLISH", fmt.Sprintf("resetPending/%s", workerID), true)
			if err != nil {
				fmt.Println("Error publishing cancel event:", err)
			}
			pending := getPendingBlocks(rpc, paymentRequest.DestinationAddress)

			for _, b := range pending.Blocks {
				if _, ok := hashCheck[b]; !ok {
					fmt.Println("Found new pending block:", b)

					confirming.DestinationAddress = paymentRequest.DestinationAddress
					confirming.Status = "confirming"
					confirming.Hash = b
					confirming.WorkerID = workerID
					sendConfirmation(confirming, paymentRequest.DestinationAddress, pool)
					setWorkerStatus("confirming", paymentRequest.WorkerID, pool)

					nano.BlockConfirm(rpc, b)
					hashCheck[b] = true
					markConfirming(pool, b, paymentRequest.DestinationAddress)
					go PaymentConfirmationWorker(pool, b, paymentRequest, workerID)
					return
				}
			}
			now := time.Now()
			if now.Sub(created) >= (20 * time.Second) {
				return
			}
		}

	}

}

func compareAmounts(expected string, received string) int {
	//compareAmounts converts strings to bigInts and returns which is larger.
	expectedInt := new(big.Int)
	receivedInt := new(big.Int)
	if _, ok := expectedInt.SetString(expected, 10); !ok {
		fmt.Println("error setting big int for expected")
	}
	if _, ok := receivedInt.SetString(received, 10); !ok {
		fmt.Println("error setting big int for received")
	}

	comparison := expectedInt.Cmp(receivedInt)

	return comparison
}

//PaymentRequestWorker will monitor the websocket for confirmation messages.
//If a confirmation comes in with the destination address, it will double check confirmation status
//and ensure the amount is the same as the expected amount.  If the transaction is pending, it will
//return a confirming status and start a paymentconfirmationworker to process.
func PaymentRequestWorker(pool *redis.Pool, paymentRequest structs.PaymentRequest, workerID string) {
	config := structs.LoadConfig()

	rpc := nanostructs.NanoRPC{Host: config.RPCHost, Port: config.RPCPort}
	c := pool.Get()
	psc := redis.PubSubConn{Conn: c}

	setWorkerStatus("pending", workerID, pool)

	if err := psc.Subscribe("nano-websocket-confirmations"); err != nil {
		fmt.Println("Error subscribing to nano-websocket-confirmations:", err)
	}

	// We record the known blocks for the account to prevent false credit for payments
	hashes := getKnownBlocks(pool, paymentRequest.DestinationAddress)
	hashCheck := setPendingHashMap(hashes)

	// Set a poll to check to see if there are any active pending blocks that don't pass through the websocket
	go pollPending(paymentRequest, rpc, hashCheck, pool, workerID)

	for {
		switch v := psc.ReceiveWithTimeout(time.Duration(20 * time.Second)).(type) {
		case redis.Message:
			if v.Channel == "nano-websocket-confirmations" {
				websocketJSON := parseWebhookMessage(v.Data)

				// Check if the block is a send to the destination account
				if websocketJSON.Message.Block.Subtype == "send" && websocketJSON.Message.Block.LinkAsAccount == paymentRequest.DestinationAddress {
					confBlockReturn := getConfirmationHeight(rpc, websocketJSON.Message.Hash)
					// We retrieve the confirmation height of the sending account to see if the received block is old.
					// It must be in the most recent 5% of blocks to be accepted.
					confAccountReturn, countErr := nano.AccountInformation(rpc, websocketJSON.Message.Account, nil)
					if countErr != nil {
						fmt.Println("Error getting the block count to invalidate old blocks:", countErr)
					}

					confHeightBlock, heightErr := strconv.Atoi(confBlockReturn)
					if heightErr != nil {
						fmt.Println("Error converting confirmation height:", heightErr)
					}
					confHeightAccount, bcErr := strconv.Atoi(string(confAccountReturn["confirmation_height"].(string)))
					if bcErr != nil {
						fmt.Println("Error converting confirmation height:", bcErr)
					}

					blockAgeCheck := float64(confHeightAccount) * .95

					if _, ok := hashCheck[websocketJSON.Message.Hash]; !ok && float64(confHeightBlock) >= blockAgeCheck {
						fmt.Printf("Hash %s didn't exist in pending or account history\n", websocketJSON.Message.Hash)
						fmt.Println("received amount:", websocketJSON.Message.Amount)
						fmt.Println("expected amount:", paymentRequest.Amount)
						processPaymentMessage(pool, paymentRequest, websocketJSON.Message.Amount, websocketJSON.Message.Hash, websocketJSON.Message.Account, workerID)
					} else {
						fmt.Printf("Hash %s existed\n", websocketJSON.Message.Hash)
					}
					cancelC := pool.Get()
					cancelC.Do("PUBLISH", fmt.Sprintf("cancel/%s", workerID), true)
					return
				}
				if v.Channel == fmt.Sprintf("cancel/%s", workerID) {
					return
				}
			}
		case error:
			confC := pool.Get()
			defer confC.Close()

			confReturn, confErr := redis.String(confC.Do("GET", fmt.Sprintf("confirming/%s", paymentRequest.DestinationAddress)))
			if confErr != nil {
				fmt.Println("Error retrieving data from redis:", confErr)
			}
			fmt.Println("confReturn:", confReturn)
			if confReturn != "confirming" {
				var payment structs.Payment

				payment.Status = "error"
				payment.ErrorCode = 0
				payment.ErrorMessage = "Payment Request reached time limit with no payment."
				payment.DestinationAddress = paymentRequest.DestinationAddress
				payment.ExpectedAmount = paymentRequest.Amount

				sendConfirmation(payment, paymentRequest.DestinationAddress, pool)
				setWorkerStatus("timeout", workerID, pool)

				fmt.Printf("Timer expired, publishing to cancel/%s\n", workerID)
				cancelC := pool.Get()
				_, timeoutErr := cancelC.Do("PUBLISH", fmt.Sprintf("cancel/%s", workerID), true)
				if timeoutErr != nil {
					fmt.Println("Error sending timeout cancellation", timeoutErr)
				}
				return
			}

			fmt.Println("There's currently a block confirming.")
			confC.Do("DEL", fmt.Sprintf("confirming/%s", paymentRequest.DestinationAddress))
			cancelC := pool.Get()
			cancelC.Do("PUBLISH", fmt.Sprintf("cancel/%s", workerID), true)
			return
		}
	}
}
