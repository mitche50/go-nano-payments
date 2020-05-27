package workers

import (
	"encoding/json"
	"fmt"
	nano "nano-pp/nanocurrency"
	"nano-pp/nanocurrency/nanostructs"
	nanostruct "nano-pp/nanocurrency/nanostructs"
	structs "nano-pp/paymentstructs"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
)

//getBlockInfo pulls the block info for a provided hash from the Nano node.
func getBlockInfo(rpc nanostructs.NanoRPC, hash string) nanostruct.BlockInfo {
	var blockInfo nanostruct.BlockInfo
	blockReturn, err := nano.BlockInfo(rpc, hash)
	if err != nil {
		fmt.Println("Error getting info for confirmation height:", err)
	}
	d := json.NewDecoder(strings.NewReader(string(blockReturn)))
	d.Decode(&blockInfo)

	return blockInfo
}

func processPaymentMessage(pool *redis.Pool, paymentRequest structs.PaymentRequest, validatedAmount string, hash string, sendingAddress string, workerID string) {
	confirmationWorkerC := pool.Get()
	amountComparison := compareAmounts(paymentRequest.Amount, validatedAmount)

	var payment structs.Payment
	payment.DestinationAddress = paymentRequest.DestinationAddress
	payment.ExpectedAmount = validatedAmount
	payment.ValidatedAmount = paymentRequest.Amount
	payment.Hash = hash
	payment.SendingAddress = sendingAddress

	if amountComparison == 0 {

		payment.Status = "success"

		sendConfirmation(payment, paymentRequest.DestinationAddress, pool)
		setWorkerStatus("success", workerID, pool)

		fmt.Println("PAYMENT SUCCESS!")

		confirmationWorkerC.Do("DEL", fmt.Sprintf("confirming/%s", paymentRequest.DestinationAddress))
		_, err := confirmationWorkerC.Do("PUBLISH", fmt.Sprintf("cancel/%s", workerID), true)
		if err != nil {
			fmt.Println("Error publishing cancel event:", err)
		}
		return
	} else if amountComparison == -1 {
		var payment structs.Payment

		overpaymentAmount := calcDifference(amountComparison, paymentRequest.Amount, validatedAmount)
		fmt.Println("Overpayment amount:", overpaymentAmount)

		payment.Status = "error"
		payment.ErrorCode = 1
		payment.ErrorMessage = fmt.Sprintf("Overpayment of %s raw received", overpaymentAmount.String())

		sendConfirmation(payment, paymentRequest.DestinationAddress, pool)
		setWorkerStatus("overpayment", workerID, pool)

		fmt.Println("OVERPAYMENT!")
		confirmationWorkerC.Do("DEL", fmt.Sprintf("confirming/%s", paymentRequest.DestinationAddress))
		_, err := confirmationWorkerC.Do("PUBLISH", fmt.Sprintf("cancel/%s", workerID), true)
		if err != nil {
			fmt.Println("Error publishing cancel event:", err)
		}
		return
	} else if amountComparison == 1 {
		var payment structs.Payment

		underpaymentAmount := calcDifference(amountComparison, paymentRequest.Amount, validatedAmount)
		fmt.Println("Underpayment amount:", underpaymentAmount)

		payment.Status = "error"
		payment.ErrorCode = 2
		payment.ErrorMessage = fmt.Sprintf("Underpayment received, remaining balance of %s raw owed.", underpaymentAmount.String())

		sendConfirmation(payment, paymentRequest.DestinationAddress, pool)
		setWorkerStatus("underpayment", workerID, pool)

		fmt.Println("UNDERPAYMENT!")
		confirmationWorkerC.Do("DEL", fmt.Sprintf("confirming/%s", paymentRequest.DestinationAddress))
		_, err := confirmationWorkerC.Do("PUBLISH", fmt.Sprintf("cancel/%s", workerID), true)
		if err != nil {
			fmt.Println("Error publishing cancel event:", err)
		}
		return
	}
}

//PaymentConfirmationWorker checks the confirmation status of a provided hash and
//sends a message when the block is confirmed.
func PaymentConfirmationWorker(pool *redis.Pool, hash string, paymentRequest structs.PaymentRequest, workerID string) {
	config := structs.LoadConfig()
	rpc := nanostructs.NanoRPC{Host: config.RPCHost, Port: config.RPCPort}
	pendingTimer := time.NewTimer(5 * time.Second)

	for {
		select {
		case <-pendingTimer.C:
			blockInfo := getBlockInfo(rpc, hash)

			if blockInfo.Confirmed == "false" {
				fmt.Println("Block still confirming, resubmitting")
				nano.BlockConfirm(rpc, hash)
				pendingTimer.Reset(5 * time.Second)
			} else {
				processPaymentMessage(pool, paymentRequest, blockInfo.Amount, hash, blockInfo.BlockAccount, workerID)
			}
		}

	}
}
