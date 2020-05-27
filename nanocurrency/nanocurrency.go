package nanocurrency

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"nano-pp/nanocurrency/nanostructs"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//NodePost formats and posts to the Nano Node and returns the response
//formatted as a map[string]interface{}.
func NodePost(host string, port string, data *map[string]string) (map[string]interface{}, error) {
	//Convert the map to JSON
	dataJSON, _ := json.Marshal(data)

	//Declare the HTTP client and set a timeout
	var netClient = &http.Client{
		Timeout: time.Second * 3,
	}

	//Post to Nano node
	r, responseError := netClient.Post(host+":"+port, "application/json", bytes.NewBuffer(dataJSON))
	if responseError != nil {
		return nil, responseError
	}

	//Parse to back to map
	body, _ := ioutil.ReadAll(r.Body)
	responseJSON := make(map[string]interface{})
	json.Unmarshal(body, &responseJSON)

	//If there is an error in the Node, capture it and return an error
	if val, ok := responseJSON["error"]; ok {
		return nil, fmt.Errorf(val.(string))
	}

	return responseJSON, nil
}

//RawNodePost returns the raw JSON value for formatting later
func RawNodePost(host string, port string, data *map[string]string) ([]byte, error) {
	//Convert the map to JSON
	dataJSON, _ := json.Marshal(data)

	//Declare the HTTP client and set a timeout
	var netClient = &http.Client{
		Timeout: time.Second * 3,
	}

	//Post to Nano node
	r, responseError := netClient.Post(host+":"+port, "application/json", bytes.NewBuffer(dataJSON))
	if responseError != nil {
		return nil, responseError
	}

	//Parse to back to map
	body, _ := ioutil.ReadAll(r.Body)

	return body, nil
}

//BlockCount returns a map with the checked and unchecked blocks of the node running on the provided host / port combo.
func BlockCount(rpc nanostructs.NanoRPC) (map[string]interface{}, error) {
	data := map[string]string{"action": "block_count"}

	response, blockError := NodePost(rpc.Host, rpc.Port, &data)
	if blockError != nil {
		return nil, blockError
	}

	return response, nil
}

//AccountBalance returns a map with the corresponding pending and balance of the provided account.
func AccountBalance(rpc nanostructs.NanoRPC, account string) (map[string]interface{}, error) {
	data := map[string]string{"action": "account_balance", "account": account}

	response, accountError := NodePost(rpc.Host, rpc.Port, &data)
	if accountError != nil {
		return nil, accountError
	}

	return response, nil
}

//AccountBlocks returns a map with the total amount of blocks associated with the provided account.
func AccountBlocks(rpc nanostructs.NanoRPC, account string) (map[string]interface{}, error) {
	data := map[string]string{"action": "account_block_count", "account": account}

	response, accountError := NodePost(rpc.Host, rpc.Port, &data)
	if accountError != nil {
		return nil, accountError
	}

	return response, nil
}

//AccountInformation returns a map with the frontier, open block, change representative block, balance,
//last modified timestamp from local database, account version & block count for the provided account. Only works for
//accounts that have an entry on the ledger, will return "Account not found" otherwise.  Optional arguments include
//representative, weight and pending to return the representative, voting weight and pending balance for the account.
//Optional arguments should be included in a map.
func AccountInformation(rpc nanostructs.NanoRPC, account string, optional map[string]string) (map[string]interface{}, error) {
	data := map[string]string{"action": "account_info", "account": account}

	//If the length of optional arguments is > 0, iterate over them and make sure the arguments are valid.
	//If they're valid, add to the data map, if not return an error.
	if len(optional) > 0 {
		for k, v := range optional {
			switch strings.ToLower(k) {
			case "representative":
				if _, ok := data[k]; ok {
					return nil, fmt.Errorf("Duplicate key provided: %s", k)
				}
				_, err := strconv.ParseBool(v)
				if err != nil {
					return nil, fmt.Errorf("Invalid optional argument value: %s: %s", k, v)
				}
				data[k] = v

			case "weight":
				if _, ok := data[k]; ok {
					return nil, fmt.Errorf("Duplicate key provided: %s", k)
				}
				_, err := strconv.ParseBool(v)
				if err != nil {
					return nil, fmt.Errorf("Invalid optional argument value: %s: %s", k, v)
				}
				data[k] = v

			case "pending":
				if _, ok := data[k]; ok {
					return nil, fmt.Errorf("Duplicate key provided: %s", k)
				}
				_, err := strconv.ParseBool(v)
				if err != nil {
					return nil, fmt.Errorf("Invalid optional argument value: %s: %s", k, v)
				}
				data[k] = v

			default:
				return nil, fmt.Errorf("Invalid optional argument key: %s: %s", k, v)
			}
		}
	}

	response, accountError := NodePost(rpc.Host, rpc.Port, &data)
	if accountError != nil {
		return nil, accountError
	}

	return response, nil
}

//AccountCreate creates a new account for the provided wallet and inserts it into the next index.
//Optional parameters include index (the index of which acconut to create) and work (indicate whether work should be
//generated after creating the account)
//Optional paramters should be included in a map.
func AccountCreate(rpc nanostructs.NanoRPC, wallet string, optional map[string]string) (map[string]interface{}, error) {
	data := map[string]string{"action": "account_create", "wallet": wallet}

	//If the length of optional arguments is > 0, iterate over them and make sure the arguments are valid.
	//If they're valid, add to the data map, if not return an error.
	if len(optional) > 0 {
		for k, v := range optional {
			switch strings.ToLower(k) {
			case "index":
				if _, ok := data[k]; ok {
					return nil, fmt.Errorf("Duplicate key provided: %s", k)
				}
				_, err := strconv.Atoi(v)
				if err != nil {
					return nil, fmt.Errorf("Invalid optional argument value: %s: %s", k, v)
				}
				data[k] = v

			case "work":
				if _, ok := data[k]; ok {
					return nil, fmt.Errorf("Duplicate key provided: %s", k)
				}
				_, err := strconv.ParseBool(v)
				if err != nil {
					return nil, fmt.Errorf("Invalid optional argument value: %s: %s", k, v)
				}
				data[k] = v

			default:
				return nil, fmt.Errorf("Invalid optional argument key: %s: %s", k, v)
			}
		}
	}
	response, createError := NodePost(rpc.Host, rpc.Port, &data)
	if createError != nil {
		return nil, createError
	}

	return response, nil
}

//AccountGet returns the account number for a provided public key.
func AccountGet(rpc nanostructs.NanoRPC, key string) (map[string]interface{}, error) {
	data := map[string]string{"action": "account_get", "key": key}

	response, getError := NodePost(rpc.Host, rpc.Port, &data)
	if getError != nil {
		return nil, getError
	}

	return response, nil
}

//AccountHistory reports send/receive information for an account. Returns only send & receive
//blocks by default (unless raw is set to true - see optional parameters below): change, state
//change & state epoch blocks are skipped, open & state open blocks will appear as receive, state
//receive/send blocks will appear as receive/send entries. Response will start with the latest
//block for the account (the frontier), and will list all blocks back to the open block of this
//account when "count" is set to "-1". Note: "local_timestamp" returned since version 18.0,
//"height" field returned since version 19.0
//Optional Parameters:
//raw (bool) [default: "false"] - if "true", returns all parameters for the block
//head(64 hexadecimal digits string, 256 bit) - Specific head block to start the history at.
//offset (decimal integer) - Amount of blocks to start after the specified head.
//reverse (bool) [default: "false"] - if "true" start from open block of the account.
//Parameter "previous" will change to "next"
func AccountHistory(rpc nanostructs.NanoRPC, account string, count string, optional map[string]string) ([]byte, error) {
	//Check to see if count is an integer.  If not, return an error.
	data := make(map[string]string)

	if count == "" {
		data["action"] = "account_history"
		data["account"] = strings.ToLower(account)
	} else {
		_, err := strconv.Atoi(count)
		if err != nil {
			return nil, fmt.Errorf("Count must be an integer: %s", count)
		}

		data["action"] = "account_history"
		data["account"] = strings.ToLower(account)
		data["count"] = count
	}

	//If the length of optional arguments is > 0, iterate over them and make sure the arguments are valid.
	//If they're valid, add to the data map, if not return an error.
	if len(optional) > 0 {
		for k, v := range optional {
			switch strings.ToLower(k) {
			case "raw":
				if _, ok := data[k]; ok {
					return nil, fmt.Errorf("Duplicate key provided: %s", k)
				}
				_, err := strconv.ParseBool(v)
				if err != nil {
					return nil, fmt.Errorf("Invalid optional argument value: %s: %s", k, v)
				}
				data[k] = v

			case "head":
				if _, ok := data[k]; ok {
					return nil, fmt.Errorf("Duplicate key provided: %s", k)
				}
				data[k] = v

			case "offset":
				if _, ok := data[k]; ok {
					return nil, fmt.Errorf("Duplicate key provided: %s", k)
				}
				_, err := strconv.Atoi(v)
				if err != nil {
					return nil, fmt.Errorf("Invalid optional argument value: %s: %s", k, v)
				}
				data[k] = v

			case "reverse":
				if _, ok := data[k]; ok {
					return nil, fmt.Errorf("Duplicate key provided: %s", k)
				}
				_, err := strconv.ParseBool(v)
				if err != nil {
					return nil, fmt.Errorf("Invalid optional argument value: %s: %s", k, v)
				}
				data[k] = v
			default:
				return nil, fmt.Errorf("Invalid optional argument key: %s: %s", k, v)
			}
		}
	}

	response, historyError := RawNodePost(rpc.Host, rpc.Port, &data)
	if historyError != nil {
		return nil, historyError
	}

	return response, nil
}

//Pending returns the pending blocks for a provided account.
//Optional Parameters:
//count (string) - If provided, limits the amount of pending blocks to the count provided.  If not, returns all
//threshold (string) - Filters the list of pending blocks to greater than or equal to the threshold in raw
//source (string) - If "true" returns the source account for each pending block
//include_active (string) - If "true" will include active blocks without finished confirmations
//sorting (string) - If "true" sorts blocks by their amounts in descending order
//include_only_confirmed (string) - If "true" returns only blocks which have their confirmation height set
// or are going through confirmation height processing
func Pending(rpc nanostructs.NanoRPC, account string, optional map[string]string) ([]byte, error) {
	data := map[string]string{"action": "pending", "account": account}

	if len(optional) > 0 {
		for k, v := range optional {
			switch strings.ToLower(k) {
			case "count":
				if _, ok := data[k]; ok {
					return nil, fmt.Errorf("Duplicate key provided: %s", k)
				}
				_, err := strconv.Atoi(v)
				if err != nil {
					return nil, fmt.Errorf("Count must be an integer: %s", v)
				}
			case "threshold":
				if _, ok := data[k]; ok {
					return nil, fmt.Errorf("Duplicate key provided: %s", k)
				}
				bi := big.NewInt(0)
				if _, ok := bi.SetString(v, 10); !ok {
					return nil, fmt.Errorf("Threshold must be an integer: %s", v)
				}
				data[k] = v
			case "source":
				// if _, ok := data[k]; ok {
				// 	return nil, fmt.Errorf("Duplicate key provided: %s", k)
				// }
				// _, err := strconv.ParseBool(v)
				// if err != nil {
				// 	return nil, fmt.Errorf("Invalid optional argument value: %s: %s", k, v)
				// }
				// data[k] = v
				return nil, fmt.Errorf("Source is not supported by golang currently")
			case "include_active":
				if _, ok := data[k]; ok {
					return nil, fmt.Errorf("Duplicate key provided: %s", k)
				}
				_, err := strconv.ParseBool(v)
				if err != nil {
					return nil, fmt.Errorf("Invalid optional argument value: %s: %s", k, v)
				}
				data[k] = v
			case "sorting":
				if _, ok := data[k]; ok {
					return nil, fmt.Errorf("Duplicate key provided: %s", k)
				}
				_, err := strconv.ParseBool(v)
				if err != nil {
					return nil, fmt.Errorf("Invalid optional argument value: %s: %s", k, v)
				}
				data[k] = v
			case "include_only_confirmed":
				if _, ok := data[k]; ok {
					return nil, fmt.Errorf("Duplicate key provided: %s", k)
				}
				_, err := strconv.ParseBool(v)
				if err != nil {
					return nil, fmt.Errorf("Invalid optional argument value: %s: %s", k, v)
				}
				data[k] = v
			default:
				return nil, fmt.Errorf("Invalid optional argument key: %s: %s", k, v)
			}
		}
	}

	response, getError := RawNodePost(rpc.Host, rpc.Port, &data)
	if getError != nil {
		return nil, getError
	}

	return response, nil
}

//BlockConfirm submits the provided block for voting
func BlockConfirm(rpc nanostructs.NanoRPC, hash string) (string, error) {
	data := map[string]string{"action": "block_confirm", "hash": hash}

	_, getError := NodePost(rpc.Host, rpc.Port, &data)
	if getError != nil {
		return "", getError
	}

	return "success", nil
}

//BlockInfo returns information on the provided block
func BlockInfo(rpc nanostructs.NanoRPC, hash string) ([]byte, error) {
	data := map[string]string{"action": "block_info", "hash": hash, "json_block": "true"}

	response, getError := RawNodePost(rpc.Host, rpc.Port, &data)
	if getError != nil {
		return nil, getError
	}

	return response, nil
}
