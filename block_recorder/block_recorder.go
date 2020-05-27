package br

import (
	"encoding/json"
	"fmt"
	nano "nano-pp/nanocurrency"
	nanostructs "nano-pp/nanocurrency/nanostructs"
	structs "nano-pp/paymentstructs"
	"os"
	"os/signal"
	"strings"

	"github.com/gomodule/redigo/redis"
)

//BlockRecorder will retrieve the most recently confirmed block hashes and pending block hashes for
//a provided account and save them in a redis set for reference
func BlockRecorder(pool *redis.Pool, destinationAccount string) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	config := structs.LoadConfig()

	rpc := nanostructs.NanoRPC{Host: config.RPCHost, Port: config.RPCPort}

	c := pool.Get()

	c.Do("DEL", fmt.Sprintf("known_pending/%s", destinationAccount))

	optionalHistory := map[string]string{"raw": "true"}
	accountHistoryresponse, accountErr := nano.AccountHistory(rpc, destinationAccount, "1000", optionalHistory)
	if accountErr != nil {
		fmt.Printf("Error retrieving account history: %v", accountErr)
	}

	optionalPending := map[string]string{"include_active": "true"}
	pendingResponse, pendingErr := nano.Pending(rpc, destinationAccount, optionalPending)
	if pendingErr != nil {
		fmt.Printf("Error retrieving pending blocks: %v", pendingErr)
	}

	var accountHistory nanostructs.AccountHistoryReturnRaw
	var pending nanostructs.Pending

	d := json.NewDecoder(strings.NewReader(string(accountHistoryresponse)))
	d.Decode(&accountHistory)
	d = json.NewDecoder(strings.NewReader(string(pendingResponse)))
	d.Decode(&pending)

	for _, v := range accountHistory.HistoryCollection {
		if v.Subtype == "receive" {
			// For receive blocks, use the Link field as this is the hash
			// of the send transaction to match what we are tracking on
			// incoming blocks
			c.Do("SADD", fmt.Sprintf("known_pending/%s", destinationAccount), v.Link)
		} else {
			c.Do("SADD", fmt.Sprintf("known_pending/%s", destinationAccount), v.Hash)
		}
	}
	for _, v := range pending.Blocks {
		// Pending blocks will always be of type Send, so we don't need to use the Link field
		_, saddErr := c.Do("SADD", fmt.Sprintf("known_pending/%s", destinationAccount), v)
		if saddErr != nil {
			fmt.Println("error adding to redis:", saddErr)
		}
	}
}
