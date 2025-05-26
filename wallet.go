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

func NewWallet(prvHex, rpc string) (*Wallet, error) {
	signer, err := NewSigner(prvHex)
	if err != nil {
		return nil, err
	}

	client := ethrpc.New(rpc)

	version, err := client.NetVersion()
	if err != nil {
		return nil, err
	}
	chainID, ok := new(big.Int).SetString(version, 10)
	if !ok {
		return nil, fmt.Errorf("wrong chainID: %s", version)
	}

	return &Wallet{
		Address: signer.Address,
		ChainID: chainID,

		Signer: signer,
		Client: client,
	}, nil
}

func NewWalletFromPath(prvPath, rpc string) (*Wallet, error) {
	b, err := os.ReadFile(prvPath)
	if err != nil {
		return nil, err
	}

	return NewWallet(strings.TrimSpace(string(b)), rpc)
}

func (w *Wallet) SendTx(to common.Address, amount *big.Int, data []byte, opts *TxOpts) (txHash string, err error) {
	opts, err = w.InitTxOpts(to, amount, data, opts)
	if err != nil {
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
		return
	}

	raw, err := tx.MarshalBinary()
	if err != nil {
		return
	}

	return w.Client.EthSendRawTransaction(hexutil.Encode(raw))
}

func (w *Wallet) SendLegacyTx(to common.Address, amount *big.Int, data []byte, opts *TxOpts) (txHash string, err error) {
	opts, err = w.InitTxOpts(to, amount, data, opts)
	if err != nil {
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
		return
	}

	raw, err := tx.MarshalBinary()
	if err != nil {
		return
	}

	return w.Client.EthSendRawTransaction(hexutil.Encode(raw))
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
