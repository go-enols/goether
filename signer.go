package goether

import (
	"crypto/ecdsa"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/go-enols/go-log"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

type Signer struct {
	Address common.Address
	key     *ecdsa.PrivateKey
}

func NewSigner(prvHex string) (*Signer, error) {
	log.Debug("Creating new signer from private key")
	k, err := crypto.HexToECDSA(prvHex)
	if err != nil {
		log.Error("Failed to parse private key", "error", err)
		return nil, err
	}

	address := crypto.PubkeyToAddress(k.PublicKey)
	log.Debug("Signer created successfully", "address", address.Hex())
	return &Signer{
		key:     k,
		Address: address,
	}, nil
}

func NewSignerFromPath(prvPath string) (*Signer, error) {
	log.Debug("Creating signer from file", "path", prvPath)
	b, err := os.ReadFile(prvPath)
	if err != nil {
		log.Error("Failed to read private key file", "path", prvPath, "error", err)
		return nil, err
	}

	log.Debug("Private key file read successfully", "path", prvPath)
	return NewSigner(strings.TrimSpace(string(b)))
}

func (s Signer) GetPrivateKey() *ecdsa.PrivateKey {
	return s.key
}

func (s Signer) GetPublicKey() []byte {
	return crypto.FromECDSAPub(&s.key.PublicKey)
}

func (s Signer) GetPublicKeyHex() string {
	return hexutil.Encode(s.GetPublicKey())
}

// SignTx DynamicFeeTx
func (s *Signer) SignTx(
	nonce int, to common.Address, amount *big.Int,
	gasLimit int, gasTipCap *big.Int, gasFeeCap *big.Int,
	data []byte, chainID *big.Int,
) (tx *types.Transaction, err error) {
	log.Debug("Signing dynamic fee transaction",
		"from", s.Address.Hex(),
		"to", to.Hex(),
		"nonce", nonce,
		"amount", amount.String(),
		"gasLimit", gasLimit,
		"gasTipCap", gasTipCap.String(),
		"gasFeeCap", gasFeeCap.String(),
		"chainID", chainID.String())

	baseTx := &types.DynamicFeeTx{
		Nonce:     uint64(nonce),
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       uint64(gasLimit),
		To:        &to,
		Value:     amount,
		Data:      data,
	}

	tx, err = types.SignNewTx(s.key, types.LatestSignerForChainID(chainID), baseTx)
	if err != nil {
		log.Error("Failed to sign dynamic fee transaction", "error", err)
		return nil, err
	}

	log.Debug("Dynamic fee transaction signed successfully", "txHash", tx.Hash().Hex())
	return tx, nil
}

func (s *Signer) SignLegacyTx(
	nonce int, to common.Address, amount *big.Int,
	gasLimit int, gasPrice *big.Int,
	data []byte, chainID *big.Int,
) (tx *types.Transaction, err error) {
	log.Debug("Signing legacy transaction",
		"from", s.Address.Hex(),
		"to", to.Hex(),
		"nonce", nonce,
		"amount", amount.String(),
		"gasLimit", gasLimit,
		"gasPrice", gasPrice.String(),
		"chainID", chainID.String())

	tx, err = types.SignTx(
		types.NewTransaction(
			uint64(nonce), to, amount,
			uint64(gasLimit), gasPrice, data),
		types.NewEIP155Signer(chainID),
		s.key,
	)
	if err != nil {
		log.Error("Failed to sign legacy transaction", "error", err)
		return nil, err
	}

	log.Debug("Legacy transaction signed successfully", "txHash", tx.Hash().Hex())
	return tx, nil
}

func (s Signer) SignMsg(msg []byte) (sig []byte, err error) {
	log.Debug("Signing message", "signer", s.Address.Hex(), "msgLength", len(msg))
	hash := accounts.TextHash(msg)
	sig, err = crypto.Sign(hash, s.key)
	if err != nil {
		log.Error("Failed to sign message", "error", err)
		return
	}

	sig[64] += 27
	log.Debug("Message signed successfully", "signature", hexutil.Encode(sig))
	return
}

func (s Signer) SignTypedData(typedData apitypes.TypedData) (sig []byte, err error) {
	log.Debug("Signing typed data", "signer", s.Address.Hex(), "domain", typedData.Domain.Name)
	hash, err := EIP712Hash(typedData)
	if err != nil {
		log.Error("Failed to generate EIP712 hash", "error", err)
		return
	}

	sig, err = crypto.Sign(hash, s.key)
	if err != nil {
		log.Error("Failed to sign typed data", "error", err)
		return
	}

	sig[64] += 27
	log.Debug("Typed data signed successfully", "signature", hexutil.Encode(sig))
	return
}

// Decrypt decrypt
func (s Signer) Decrypt(ct []byte) ([]byte, error) {
	eciesPriv := ecies.ImportECDSA(s.key)
	return eciesPriv.Decrypt(ct, nil, nil)
}
