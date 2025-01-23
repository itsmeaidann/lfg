package hpl

import (
	"encoding/binary"
	"fmt"
	"lfg/pkg/utils"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/vmihailenco/msgpack/v5"
)

const VERIFYING_CONTRACT = "0x0000000000000000000000000000000000000000"

var nonceCounter int64

func getNonce() int64 {
	nonce := atomic.AddInt64(&nonceCounter, 1) + time.Now().UnixMilli()
	return nonce
}

func (e *HplExchange) getRequestSignature(action any, vaultAddress string, nonce int64) (RsvSignature, error) {
	hash, err := hashAction(action, vaultAddress, uint64(nonce))
	if err != nil {
		return RsvSignature{}, err
	}
	message := e.buildMessage(hash.Bytes())
	v, r, s, err := e.SignInner(message)
	if err != nil {
		return RsvSignature{}, err
	}
	return getRsvSignature(r, s, v), nil
}

func (e *HplExchange) buildMessage(hash []byte) apitypes.TypedDataMessage {
	source := e.getSource()
	return apitypes.TypedDataMessage{
		"source":       source,
		"connectionId": hash,
	}
}

func (e *HplExchange) getSource() string {
	if e.IsMainnet {
		return "a"
	} else {
		return "b"
	}
}

func hashAction(action any, vaultAddress string, nonce uint64) (common.Hash, error) {
	data, err := msgpack.Marshal(action)
	if err != nil {
		return common.Hash{}, fmt.Errorf("fail to pack the data: %v: %v", action, err)
	}

	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, nonce)
	data = append(data, nonceBytes...)
	if vaultAddress == "" {
		data = append(data, []byte("\x00")...)
	} else {
		data = append(data, []byte("\x01")...)
		vaultAddressBytes, err := utils.HexToBytes(vaultAddress)
		if err != nil {
			return common.Hash{}, err
		}
		data = append(data, vaultAddressBytes...)
	}
	hash := crypto.Keccak256Hash(data)
	return hash, nil
}

func (e *HplExchange) SignInner(message apitypes.TypedDataMessage) (byte, [32]byte, [32]byte, error) {
	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"Agent": []apitypes.Type{
				{
					Name: "source",
					Type: "string",
				},
				{
					Name: "connectionId",
					Type: "bytes32",
				},
			},
			"EIP712Domain": {
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
		},
		PrimaryType: "Agent",
		Domain: apitypes.TypedDataDomain{
			Name:              "Exchange",
			Version:           "1",
			ChainId:           math.NewHexOrDecimal256(1337), // HPL use 1337 as chainId regardless of testnet/mainnet
			VerifyingContract: VERIFYING_CONTRACT,
		},
		Message: message,
	}

	bytes, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		return 0, [32]byte{}, [32]byte{}, err
	}

	sig, err := crypto.Sign(bytes, e.AccountPrivKey)
	if err != nil {
		return 0, [32]byte{}, [32]byte{}, err
	}
	v, r, s := utils.SignatureToVRS(sig)
	return v, r, s, nil
}
