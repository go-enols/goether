package goether

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/go-enols/go-log"
)

func EthToBN(amount float64) (bn *big.Int) {
	log.Debug("Converting ETH to big number", "amount", amount)
	bf := new(big.Float).Mul(big.NewFloat(amount), big.NewFloat(1000000000000000000))
	bn, _ = bf.Int(bn)
	log.Debug("ETH converted to big number", "amount", amount, "result", bn.String())
	return bn
}

func GweiToBN(amount float64) (bn *big.Int) {
	log.Debug("Converting Gwei to big number", "amount", amount)
	bf := new(big.Float).Mul(big.NewFloat(amount), big.NewFloat(1000000000))
	bn, _ = bf.Int(bn)
	log.Debug("Gwei converted to big number", "amount", amount, "result", bn.String())
	return bn
}

func EIP712Hash(typedData apitypes.TypedData) (hash []byte, err error) {
	log.Debug("Generating EIP712 hash", "primaryType", typedData.PrimaryType, "domain", typedData.Domain.Name)
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		log.Error("Failed to hash EIP712 domain", "error", err)
		return
	}
	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		log.Error("Failed to hash typed data", "primaryType", typedData.PrimaryType, "error", err)
		return
	}
	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	hash = crypto.Keccak256(rawData)
	log.Debug("EIP712 hash generated successfully", "hashLength", len(hash))
	return
}

func Ecrecover(hash, signature []byte) (publicBy []byte, address common.Address, err error) {
	log.Debug("Recovering public key from signature", "hashLength", len(hash), "signatureLength", len(signature))
	sig := make([]byte, len(signature))
	copy(sig, signature)
	if len(sig) != 65 {
		err = fmt.Errorf("invalid length of signture: %d", len(sig))
		log.Error("Invalid signature length", "length", len(sig))
		return
	}

	if sig[64] != 27 && sig[64] != 28 && sig[64] != 1 && sig[64] != 0 {
		err = fmt.Errorf("invalid signature type")
		log.Error("Invalid signature type", "type", sig[64])
		return
	}
	if sig[64] >= 27 {
		sig[64] -= 27
	}

	publicBy, err = crypto.Ecrecover(hash, sig)
	if err != nil {
		err = fmt.Errorf("can not ecrecover: %v", err)
		log.Error("Failed to recover public key", "error", err)
		return
	}

	address = common.BytesToAddress(crypto.Keccak256(publicBy[1:])[12:])
	log.Debug("Public key recovered successfully", "address", address.Hex())
	return
}

// Encrypt encrypt
func Encrypt(publicKey string, message []byte) ([]byte, error) {
	log.Debug("Encrypting message", "publicKey", publicKey, "messageLength", len(message))
	pub := common.FromHex(publicKey)
	pubKey, err := crypto.UnmarshalPubkey(pub)
	if err != nil {
		log.Error("Failed to unmarshal public key", "publicKey", publicKey, "error", err)
		return nil, err
	}
	eciesPub := ecies.ImportECDSAPublic(pubKey)
	result, err := ecies.Encrypt(rand.Reader, eciesPub, message, nil, nil)
	if err != nil {
		log.Error("Failed to encrypt message", "error", err)
		return nil, err
	}
	log.Debug("Message encrypted successfully", "resultLength", len(result))
	return result, nil
}
