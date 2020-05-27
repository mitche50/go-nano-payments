package nanostructs

//AccountHistoryReturn contains the Account, a list of blocks and the hash of the previous transaction
type AccountHistoryReturn struct {
	Account           string         `json:"account"`
	HistoryCollection []HistoryBlock `json:"history"`
	Previous          string         `json:"previous"`
}

//AccountHistoryReturnRaw contains the Account, a list of blocks and the hash of the previous transaction
type AccountHistoryReturnRaw struct {
	Account           string            `json:"account"`
	HistoryCollection []HistoryBlockRaw `json:"history"`
	Previous          string            `json:"previous"`
}

//HistoryBlock contains the information for blocks in an account's history return
type HistoryBlock struct {
	Type           string `json:"type"`
	Account        string `json:"account"`
	Amount         string `json:"amount"`
	LocalTimestamp string `json:"local_timestamp"`
	Height         string `json:"height"`
	Hash           string `json:"hash"`
}

//HistoryBlockRaw contains the information in an account's history in raw form
type HistoryBlockRaw struct {
	Account        string `json:"account"`
	Amount         string `json:"amount"`
	LocalTimestamp string `json:"local_timestamp"`
	Height         string `json:"height"`
	Hash           string `json:"hash"`
	Work           string `json:"work"`
	Subtype        string `json:"subtype"`
	Representative string `json:"representative"`
	Link           string `json:"link"`
	Balance        string `json:"balance"`
	Previous       string `json:"previous"`
	Signature      string `json:"signature"`
	Type           string `json:"type"`
}

//Pending contains a list of pending blocks for an account
type Pending struct {
	Blocks []string `json:"blocks"`
}

//Block contains details on the block
type Block struct {
	Type           string `json:"type"`
	Account        string `json:"account"`
	Previous       string `json:"previous"`
	Representative string `json:"representative"`
	Balance        string `json:"balance"`
	Link           string `json:"link"`
	LinkAsAccount  string `json:"link_as_account"`
	Signature      string `json:"signature"`
	Work           string `json:"work"`
}

//BlockInfo contains the details for a block info return
type BlockInfo struct {
	BlockAccount   string `json:"block_account"`
	Amount         string `json:"amount"`
	Balance        string `json:"balance"`
	Height         string `json:"height"`
	LocalTimestamp string `json:"local_timestamp"`
	Confirmed      string `json:"confirmed"`
	Contents       Block  `json:"contents"`
	Subtype        string `json:"subtype"`
}

//NanoRPC contains the port and host information for the Nano node
type NanoRPC struct {
	Host string
	Port string
}
