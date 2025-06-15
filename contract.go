package goether

import (
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/go-enols/ethrpc"
	"github.com/go-enols/go-log"
)

type Contract struct {
	Address common.Address
	ABI     abi.ABI

	Wallet *Wallet
	Client *ethrpc.EthRPC
}

func NewContract(address common.Address, abiStr, rpc string, wallet *Wallet) (*Contract, error) {
	log.Debug("Creating new contract instance",
		"address", address.Hex(),
		"rpc", rpc,
		"walletAddress", wallet.Address.Hex())

	Abi, err := abi.JSON(strings.NewReader(abiStr))
	if err != nil {
		log.Error("Failed to parse contract ABI", "error", err)
		return nil, err
	}

	log.Debug("Contract instance created successfully", "address", address.Hex())
	return &Contract{
		Address: address,
		ABI:     Abi,
		Wallet:  wallet,
		Client:  wallet.Client,
	}, nil
}

// CallMethod Only read contract status
// tag:
//
//		HEX String - an integer block number
//	 String "earliest" for the earliest/genesis block
//	 String "latest" - for the latest mined block
//	 String "pending" - for the pending state/transactions
func (c *Contract) CallMethod(methodName, tag string, args ...interface{}) (res string, err error) {
	log.Debug("Calling contract read method",
		"contract", c.Address.Hex(),
		"method", methodName,
		"tag", tag,
		"argsCount", len(args))

	data, err := c.EncodeData(methodName, args...)
	if err != nil {
		log.Error("Failed to encode method data for call", "method", methodName, "error", err)
		return
	}

	res, err = c.Client.EthCall(ethrpc.T{
		Data: hexutil.Encode(data),
		To:   c.Address.String(),
		From: c.Address.String(),
	}, tag)
	if err != nil {
		log.Error("Failed to call contract method", "method", methodName, "error", err)
		return
	}

	log.Debug("Contract method called successfully", "method", methodName, "result", res)
	return res, nil
}

// ExecMethod Execute tx
func (c *Contract) ExecMethod(methodName string, opts *TxOpts, args ...interface{}) (txHash string, err error) {
	log.Debug("Executing contract method",
		"contract", c.Address.Hex(),
		"method", methodName,
		"argsCount", len(args))

	if c.Wallet == nil {
		err = errors.New("wallet is nil")
		log.Error("Cannot execute contract method: wallet is nil", "method", methodName)
		return
	}

	data, err := c.EncodeData(methodName, args...)
	if err != nil {
		log.Error("Failed to encode method data for execution", "method", methodName, "error", err)
		return
	}

	txHash, err = c.Wallet.SendTx(c.Address, big.NewInt(0), data, opts)
	if err != nil {
		log.Error("Failed to execute contract method", "method", methodName, "error", err)
		return
	}

	log.Debug("Contract method executed successfully", "method", methodName, "txHash", txHash)
	return txHash, nil
}

func (c *Contract) GetAddress() string {
	return c.Address.String()
}

func (c *Contract) EncodeData(methodName string, args ...interface{}) ([]byte, error) {
	log.Debug("Encoding contract method data", "method", methodName, "argsCount", len(args))
	data, err := c.ABI.Pack(methodName, args...)
	if err != nil {
		log.Error("Failed to encode method data", "method", methodName, "error", err)
		return nil, err
	}
	log.Debug("Method data encoded successfully", "method", methodName, "dataLength", len(data))
	return data, nil
}

func (c *Contract) EncodeDataHex(methodName string, args ...interface{}) (hex string, err error) {
	log.Debug("Encoding contract method data to hex", "method", methodName, "argsCount", len(args))
	by, err := c.EncodeData(methodName, args...)
	if err != nil {
		log.Error("Failed to encode method data to hex", "method", methodName, "error", err)
		return
	}

	hex = hexutil.Encode(by)
	log.Debug("Method data encoded to hex successfully", "method", methodName, "hex", hex)
	return hex, nil
}

func (c *Contract) DecodeData(data []byte) (methodName string, params map[string]interface{}, err error) {
	log.Debug("Decoding contract method data", "dataLength", len(data))
	if len(data) < 4 {
		err = errors.New("data is too short")
		log.Error("Cannot decode data: too short", "dataLength", len(data))
		return
	}

	method, err := c.ABI.MethodById(data[:4])
	if err != nil {
		log.Error("Failed to find method by ID", "error", err)
		return
	}
	methodName = method.Name

	params = make(map[string]interface{})
	err = method.Inputs.UnpackIntoMap(params, data[4:])
	if err != nil {
		log.Error("Failed to unpack method parameters", "method", methodName, "error", err)
		return
	}

	log.Debug("Method data decoded successfully", "method", methodName, "paramsCount", len(params))
	return
}

func (c *Contract) DecodeDataHex(dataHex string) (methodName string, params map[string]interface{}, err error) {
	log.Debug("Decoding contract method data from hex", "dataHex", dataHex)
	data := common.FromHex(dataHex)
	return c.DecodeData(data)
}

func (c *Contract) DecodeEvent(topics []common.Hash, data []byte) (eventName string, values map[string]interface{}, err error) {
	log.Debug("Decoding contract event", "topicsCount", len(topics), "dataLength", len(data))
	if len(topics) < 1 {
		err = errors.New("no topics found")
		log.Error("Cannot decode event: no topics found")
		return
	}

	event, err := c.ABI.EventByID(topics[0])
	if err != nil {
		log.Error("Failed to find event by ID", "error", err)
		return
	}
	eventName = event.Name

	values = make(map[string]interface{})
	// parse topics
	var indexed abi.Arguments
	for _, arg := range event.Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	indexTopics := []common.Hash{}
	for _, topic := range topics[1:] {
		indexTopics = append(indexTopics, topic)
	}
	if err = abi.ParseTopicsIntoMap(values, indexed, indexTopics); err != nil {
		log.Error("Failed to parse event topics", "event", eventName, "error", err)
		return
	}

	// parse data
	err = event.Inputs.UnpackIntoMap(values, data)
	if err != nil {
		log.Error("Failed to unpack event data", "event", eventName, "error", err)
		return
	}

	log.Debug("Event decoded successfully", "event", eventName, "valuesCount", len(values))
	return
}

func (c *Contract) DecodeEventHex(topicsHex []string, dataHex string) (eventName string, values map[string]interface{}, err error) {

	topics := []common.Hash{}
	for _, topicHex := range topicsHex {
		topics = append(topics, common.HexToHash(topicHex))
	}
	return c.DecodeEvent(topics, common.FromHex(dataHex))
}

func (c *Contract) DecodeFromMethod(method string, output any, results *[]any) error {

	if results == nil {
		results = new([]any)
	}

	var data []byte

	if value, ok := output.(string); ok {
		d, err := hexutil.Decode(value)
		if err != nil {
			return err
		}
		data = d
	} else if value, ok := output.([]byte); ok {
		data = value
	} else {
		return errors.New("output 无效的类型")
	}
	if len(data) == 0 {
		*results = make([]interface{}, 0)
		return nil
	}

	if len(*results) == 0 {
		res, err := c.ABI.Unpack(method, data)
		*results = res
		return err
	}
	res := *results
	return c.ABI.UnpackIntoInterface(res[0], method, data)
}
