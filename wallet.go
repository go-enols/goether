package goether

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/go-enols/ethrpc"
	"github.com/go-enols/go-log"
)

type TxOpts struct {
	Nonce     *int
	GasLimit  *int
	GasPrice  *big.Int
	GasTipCap *big.Int
	GasFeeCap *big.Int
}

// GetOldFee 计算出本次如果使用旧版交易时最大消耗Gas手续费
func (t *TxOpts) GetOldFee() (*big.Int, error) {
	if t.GasPrice != nil && t.GasLimit != nil {
		// 旧版费用计算：GasPrice * GasLimit
		fee := new(big.Int)
		fee.Mul(t.GasPrice, big.NewInt(int64(*t.GasLimit)))
		return fee, nil
	}
	return nil, errors.New("未设置基础参数")
}

// GetNewFee 计算出本次如果使用新版交易时最大消耗Gas手续费
func (t *TxOpts) GetNewFee() (*big.Int, error) {
	if t.GasTipCap != nil && t.GasFeeCap != nil && t.GasLimit != nil {
		// 新版费用计算：(GasTipCap + GasFeeCap) * GasLimit
		totalCap := new(big.Int)
		totalCap.Add(new(big.Int).Mul(t.GasTipCap, big.NewInt(2)), t.GasFeeCap)

		fee := new(big.Int)
		fee.Mul(totalCap, big.NewInt(int64(*t.GasLimit)))
		return fee, nil
	}
	return nil, errors.New("未设置基础参数")
}

type Wallet struct {
	Address common.Address
	ChainID *big.Int

	Signer *Signer
	Client *ethrpc.EthRPC
}

// NewWallet 创建一个新的以太坊钱包实例
//
// 该函数是创建钱包的核心方法，支持多种配置选项来定制钱包的行为。
// 钱包实例包含了地址、链ID、签名器和RPC客户端等核心组件。
//
// 参数:
//   - prvHex: 私钥的十六进制字符串表示，用于创建签名器
//   - rpc: 以太坊节点的RPC端点URL
//   - options: 可变参数，支持以下类型的配置选项：
//   - func(rpc *ethrpc.EthRPC): RPC客户端配置函数
//   - *ethrpc.EthRPC: 预先配置的RPC客户端实例
//   - string: 网络版本号，用于确定链ID
//   - *big.Int: 直接指定的链ID
//   - *Wallet: 从现有钱包复制链ID和客户端配置
//
// 返回值:
//   - *Wallet: 创建的钱包实例，包含地址、链ID、签名器和RPC客户端
//   - error: 创建过程中的错误，如私钥无效、RPC连接失败等
//
// 工作流程:
//  1. 解析可变参数，提取各种配置选项
//  2. 使用私钥创建签名器，获取钱包地址
//  3. 如果未提供RPC客户端，则使用RPC URL和配置选项创建新客户端
//  4. 如果未指定网络版本，则通过RPC调用获取
//  5. 如果未指定链ID，则从网络版本解析
//  6. 组装并返回完整的钱包实例
//
// 使用示例:
//
//		// 基本用法
//		wallet, err := NewWallet("0x1234...", "https://mainnet.infura.io/v3/YOUR-PROJECT-ID")
//
//		// 指定链ID
//		chainID := big.NewInt(1) // 主网
//		wallet, err := NewWallet("0x1234...", "https://mainnet.infura.io/v3/YOUR-PROJECT-ID", chainID)
//
//		// 使用自定义RPC客户端
//		client := ethrpc.New("https://mainnet.infura.io/v3/YOUR-PROJECT-ID")
//		wallet, err := NewWallet("0x1234...", "", client)
//
//		// 从现有钱包复制配置
//		newWallet, err := NewWallet("0x5678...", "", existingWallet)
//	    • *Wallet: 从现有钱包复制链ID和客户端配置
//
// 返回值:
//   - *Wallet: 创建的钱包实例，包含地址、链ID、签名器和RPC客户端
//   - error: 创建过程中的错误，如私钥无效、RPC连接失败等
//
// 工作流程:
//  1. 解析可变参数，提取各种配置选项
//  2. 使用私钥创建签名器，获取钱包地址
//  3. 如果未提供RPC客户端，则使用RPC URL和配置选项创建新客户端
//  4. 如果未指定网络版本，则通过RPC调用获取
//  5. 如果未指定链ID，则从网络版本解析
//  6. 组装并返回完整的钱包实例
//
// 使用示例:
//
//	// 基本用法
//	wallet, err := NewWallet("0x1234...", "https://mainnet.infura.io/v3/YOUR-PROJECT-ID")
//
//	// 指定链ID
//	chainID := big.NewInt(1) // 主网
//	wallet, err := NewWallet("0x1234...", "https://mainnet.infura.io/v3/YOUR-PROJECT-ID", chainID)
//
//	// 使用自定义RPC客户端
//	client := ethrpc.New("https://mainnet.infura.io/v3/YOUR-PROJECT-ID")
//	wallet, err := NewWallet("0x1234...", "", client)
//
//	// 从现有钱包复制配置(这个只会复制client信息以及节点信息)
//	newWallet, err := NewWallet("0x5678...", "", existingWallet)
func NewWallet(prvHex, rpc string, options ...any) (*Wallet, error) {
	log.Debug("Creating new wallet", "rpc", rpc, "optionsCount", len(options))

	var clientOptions []func(rpc *ethrpc.EthRPC)
	var client *ethrpc.EthRPC
	var version string
	var chainID *big.Int
	for _, opt := range options {
		switch data := opt.(type) {
		case func(rpc *ethrpc.EthRPC):
			clientOptions = append(clientOptions, data)
			log.Debug("Added RPC client option function")
		case *ethrpc.EthRPC:
			client = data
			log.Debug("Using provided RPC client")
		case string:
			version = data
			log.Debug("Using provided network version", "version", version)
		case *big.Int:
			chainID = data
			version = data.String()
			log.Debug("Using provided chain ID", "chainID", chainID.String())
		case *Wallet:
			chainID = data.ChainID
			client = data.Client
			version = data.ChainID.String()
			log.Debug("Copying configuration from existing wallet", "chainID", chainID.String())
		}
	}
	signer, err := NewSigner(prvHex)
	if err != nil {
		log.Error("Failed to create signer for wallet", "error", err)
		return nil, err
	}

	if client == nil {
		log.Debug("Creating new RPC client", "rpc", rpc)
		client = ethrpc.New(rpc, clientOptions...)
	}

	if version == "" {
		log.Debug("Fetching network version from RPC")
		version, err = client.NetVersion()
		if err != nil {
			log.Error("Failed to get network version", "error", err)
			return nil, err
		}
		log.Debug("Network version retrieved", "version", version)
	}
	if chainID == nil {
		log.Debug("Parsing chain ID from version", "version", version)
		var ok bool
		chainID, ok = new(big.Int).SetString(version, 10)
		if !ok {
			log.Error("Invalid chain ID format", "version", version)
			return nil, fmt.Errorf("wrong chainID: %s", version)
		}
		log.Debug("Chain ID parsed successfully", "chainID", chainID.String())
	}

	log.Debug("Wallet created successfully",
		"address", signer.Address.Hex(),
		"chainID", chainID.String(),
		"rpc", rpc)

	return &Wallet{
		Address: signer.Address,
		ChainID: chainID,

		Signer: signer,
		Client: client,
	}, nil
}

func NewWalletFromPath(prvPath, rpc string) (*Wallet, error) {
	log.Debug("Creating wallet from private key file", "path", prvPath, "rpc", rpc)
	b, err := os.ReadFile(prvPath)
	if err != nil {
		log.Error("Failed to read private key file for wallet", "path", prvPath, "error", err)
		return nil, err
	}

	log.Debug("Private key file read successfully for wallet creation", "path", prvPath)
	return NewWallet(strings.TrimSpace(string(b)), rpc)
}

func (w *Wallet) SendTx(to common.Address, amount *big.Int, data []byte, opts *TxOpts) (txHash string, err error) {
	log.Debug("Sending dynamic fee transaction",
		"from", w.Address.Hex(),
		"to", to.Hex(),
		"amount", amount.String(),
		"dataLength", len(data))

	opts, err = w.InitTxOpts(to, amount, data, opts)
	if err != nil {
		log.Error("Failed to initialize transaction options", "error", err)
		return
	}

	if amount == nil {
		amount = big.NewInt(0)
	}

	tx, err := w.Signer.SignTx(
		*opts.Nonce, to, amount,
		*opts.GasLimit, opts.GasTipCap, opts.GasFeeCap,
		data, w.ChainID)
	if err != nil {
		log.Error("Failed to sign transaction", "error", err)
		return
	}

	raw, err := tx.MarshalBinary()
	if err != nil {
		log.Error("Failed to marshal transaction", "error", err)
		return
	}

	txHash, err = w.Client.EthSendRawTransaction(hexutil.Encode(raw))
	if err != nil {
		log.Error("Failed to send raw transaction", "error", err)
		return
	}

	log.Debug("Dynamic fee transaction sent successfully", "txHash", txHash)
	return txHash, nil
}

func (w *Wallet) SendLegacyTx(to common.Address, amount *big.Int, data []byte, opts *TxOpts) (txHash string, err error) {
	log.Debug("Sending legacy transaction",
		"from", w.Address.Hex(),
		"to", to.Hex(),
		"amount", amount.String(),
		"dataLength", len(data))

	opts, err = w.InitTxOpts(to, amount, data, opts)
	if err != nil {
		log.Error("Failed to initialize legacy transaction options", "error", err)
		return
	}

	if amount == nil {
		amount = big.NewInt(0)
	}
	tx, err := w.Signer.SignLegacyTx(
		*opts.Nonce, to, amount,
		*opts.GasLimit, opts.GasPrice,
		data, w.ChainID)
	if err != nil {
		log.Error("Failed to sign legacy transaction", "error", err)
		return
	}

	raw, err := tx.MarshalBinary()
	if err != nil {
		log.Error("Failed to marshal legacy transaction", "error", err)
		return
	}

	txHash, err = w.Client.EthSendRawTransaction(hexutil.Encode(raw))
	if err != nil {
		log.Error("Failed to send raw legacy transaction", "error", err)
		return
	}

	log.Debug("Legacy transaction sent successfully", "txHash", txHash)
	return txHash, nil
}

func (w *Wallet) InitTxOpts(to common.Address, amount *big.Int, data []byte, opts *TxOpts) (*TxOpts, error) {
	var (
		nonce, gasLimit int
		gasPrice        big.Int
		err             error
	)

	if opts == nil {
		opts = &TxOpts{}
	}

	if opts.Nonce == nil {
		nonce, err = w.GetPendingNonce()
		if err != nil {
			return nil, err
		}
		opts.Nonce = &nonce
	}

	if opts.GasLimit == nil {
		ethrpcTx := ethrpc.T{
			From:  w.Address.String(),
			To:    to.String(),
			Value: amount,
			Data:  hexutil.Encode(data),
		}
		gasLimit, err = w.Client.EthEstimateGas(ethrpcTx)
		if err != nil {
			return nil, err
		}
		opts.GasLimit = &gasLimit
	}

	if opts.GasPrice == nil {
		gasPrice, err = w.Client.EthGasPrice()
		if err != nil {
			return nil, err
		}
		opts.GasPrice = &gasPrice
	}

	if opts.GasTipCap == nil || opts.GasFeeCap == nil {
		opts.GasTipCap = opts.GasPrice
		opts.GasFeeCap = opts.GasPrice
	}

	return opts, nil
}

func (w *Wallet) GetAddress() string {
	return w.Address.String()
}

func (w *Wallet) GetNonce() (nonce int, err error) {
	return w.Client.EthGetTransactionCount(w.GetAddress(), "latest")
}

func (w *Wallet) GetPendingNonce() (nonce int, err error) {
	return w.Client.EthGetTransactionCount(w.GetAddress(), "pending")
}

// GetBalance 获取钱包余额 如果传递了 token 则查询 token 余额
func (w *Wallet) GetBalance(token ...string) (balance big.Int, err error) {
	if len(token) > 0 {
		return w.getTokenBalance(token[0])
	}
	return w.Client.EthGetBalance(w.GetAddress(), "latest")
}

// getTokenBalance 获取 token 代币中本钱包持有的余额
func (w *Wallet) getTokenBalance(token string) (balance big.Int, err error) {
	res, err := w.Client.EthCall(ethrpc.T{
		From: w.GetAddress(),
		To:   token,
		Data: fmt.Sprintf("0x70a08231000000000000000000000000%s", w.GetAddress()[2:]),
	}, "latest")
	if err != nil {
		return
	}
	return ethrpc.ParseBigInt(res)
}
