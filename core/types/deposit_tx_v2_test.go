// Copyright 2021 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"bytes"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestDepositTxV2Hash(t *testing.T) {
	// Create two identical transactions, one V1 and one V2
	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	
	v1 := &Transaction{inner: &DepositTx{
		SourceHash:          common.HexToHash("0xdeadbeef"),
		From:                addr,
		To:                  &addr,
		Mint:                big.NewInt(1000),
		Value:               big.NewInt(2000),
		Gas:                 50000,
		IsSystemTransaction: true,
		Data:                []byte("test data"),
	}}
	
	v2 := &Transaction{inner: &DepositTxV2{DepositTx{
		SourceHash:          common.HexToHash("0xdeadbeef"),
		From:                addr,
		To:                  &addr,
		Mint:                big.NewInt(1000),
		Value:               big.NewInt(2000),
		Gas:                 50000,
		IsSystemTransaction: true,
		Data:                []byte("test data"),
	}}}
	
	// V1 and V2 should have different hashes
	hash1 := v1.Hash()
	hash2 := v2.Hash()
	
	if hash1 == hash2 {
		t.Errorf("V1 and V2 deposit transactions should have different hashes, got: %s", hash1.Hex())
	}
	
	// Create V2 without Mint (but same IsSystemTransaction)
	v2NoMint := &Transaction{inner: &DepositTxV2{DepositTx{
		SourceHash:          common.HexToHash("0xdeadbeef"),
		From:                addr,
		To:                  &addr,
		Mint:                nil,
		Value:               big.NewInt(2000),
		Gas:                 50000,
		IsSystemTransaction: true,  // Same as original
		Data:                []byte("test data"),
	}}}
	
	// V2 with Mint should hash the same as V2 without Mint
	hash2NoMint := v2NoMint.Hash()
	
	if hash2 != hash2NoMint {
		t.Errorf("V2 deposit transactions should have same hash regardless of Mint\ngot: %s\nwant: %s", hash2.Hex(), hash2NoMint.Hex())
	}
	
	// Create V2 with different IsSystemTransaction - should have different hash
	v2DiffSystem := &Transaction{inner: &DepositTxV2{DepositTx{
		SourceHash:          common.HexToHash("0xdeadbeef"),
		From:                addr,
		To:                  &addr,
		Mint:                big.NewInt(1000),
		Value:               big.NewInt(2000),
		Gas:                 50000,
		IsSystemTransaction: false,  // Different from original
		Data:                []byte("test data"),
	}}}
	
	hash2DiffSystem := v2DiffSystem.Hash()
	
	if hash2 == hash2DiffSystem {
		t.Errorf("V2 deposit transactions with different IsSystemTransaction should have different hashes")
	}
}

func TestDepositTxV2Type(t *testing.T) {
	tx := &Transaction{inner: &DepositTxV2{}}
	
	if tx.Type() != DepositTxV2Type {
		t.Errorf("DepositTxV2 type mismatch: got %d, want %d", tx.Type(), DepositTxV2Type)
	}
	
	if !tx.IsDepositTx() {
		t.Error("DepositTxV2 should return true for IsDepositTx()")
	}
}

func TestDepositTxV2Copy(t *testing.T) {
	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	original := &DepositTxV2{DepositTx{
		SourceHash:          common.HexToHash("0xdeadbeef"),
		From:                addr,
		To:                  &addr,
		Mint:                big.NewInt(1000),
		Value:               big.NewInt(2000),
		Gas:                 50000,
		IsSystemTransaction: true,
		Data:                []byte("test data"),
	}}
	
	// Test copy
	copied := original.copy()
	copiedV2, ok := copied.(*DepositTxV2)
	if !ok {
		t.Fatal("copy() should return *DepositTxV2")
	}
	
	// Verify deep copy
	if copiedV2.Mint == original.Mint {
		t.Error("Mint should be deep copied")
	}
	if copiedV2.Value == original.Value {
		t.Error("Value should be deep copied")
	}
	
	// Verify values are equal
	if copiedV2.Mint.Cmp(original.Mint) != 0 {
		t.Error("Copied Mint value mismatch")
	}
	if copiedV2.Value.Cmp(original.Value) != 0 {
		t.Error("Copied Value mismatch")
	}
}

func TestDepositTxV2Marshalling(t *testing.T) {
	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	
	// Test transaction without nonce
	tx1 := &Transaction{inner: &DepositTxV2{DepositTx{
		SourceHash:          common.HexToHash("0xdeadbeef"),
		From:                addr,
		To:                  &addr,
		Mint:                big.NewInt(1000),
		Value:               big.NewInt(2000),
		Gas:                 50000,
		IsSystemTransaction: true,
		Data:                []byte("test data"),
	}}}
	
	// Marshal to JSON
	jsonData, err := tx1.MarshalJSON()
	if err != nil {
		t.Fatalf("Failed to marshal DepositTxV2: %v", err)
	}
	
	// Unmarshal back
	var tx2 Transaction
	if err := tx2.UnmarshalJSON(jsonData); err != nil {
		t.Fatalf("Failed to unmarshal DepositTxV2: %v", err)
	}
	
	// Verify type
	if tx2.Type() != DepositTxV2Type {
		t.Errorf("Unmarshalled type mismatch: got %d, want %d", tx2.Type(), DepositTxV2Type)
	}
	
	// Verify values
	v2, ok := tx2.inner.(*DepositTxV2)
	if !ok {
		t.Fatal("Unmarshalled transaction is not DepositTxV2")
	}
	
	if v2.SourceHash != common.HexToHash("0xdeadbeef") {
		t.Error("SourceHash mismatch after unmarshalling")
	}
	if v2.From != addr {
		t.Error("From address mismatch after unmarshalling")
	}
	if v2.Mint.Cmp(big.NewInt(1000)) != 0 {
		t.Error("Mint value mismatch after unmarshalling")
	}
}

func TestDepositTxV2WithNonce(t *testing.T) {
	// Create JSON with nonce
	jsonStr := `{
		"type": "0x7d",
		"sourceHash": "0x000000000000000000000000000000000000000000000000000000000000dead",
		"from": "0x1234567890123456789012345678901234567890",
		"to": "0x1234567890123456789012345678901234567890",
		"mint": "0x3e8",
		"value": "0x7d0",
		"gas": "0xc350",
		"isSystemTx": true,
		"input": "0x74657374",
		"nonce": "0x42",
		"hash": "0x0000000000000000000000000000000000000000000000000000000000000000"
	}`
	
	var tx Transaction
	if err := json.Unmarshal([]byte(jsonStr), &tx); err != nil {
		t.Fatalf("Failed to unmarshal DepositTxV2 with nonce: %v", err)
	}
	
	// Should be wrapped with nonce
	wrapper, ok := tx.inner.(*depositTxV2WithNonce)
	if !ok {
		t.Fatal("Transaction with nonce should be wrapped in depositTxV2WithNonce")
	}
	
	if wrapper.EffectiveNonce != 0x42 {
		t.Errorf("Nonce mismatch: got %d, want %d", wrapper.EffectiveNonce, 0x42)
	}
	
	// Verify it still identifies as deposit
	if !tx.IsDepositTx() {
		t.Error("Transaction with nonce should still return true for IsDepositTx()")
	}
}

func TestDepositTxV2RLP(t *testing.T) {
	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	
	tx := &DepositTxV2{DepositTx{
		SourceHash:          common.HexToHash("0xdeadbeef"),
		From:                addr,
		To:                  &addr,
		Mint:                big.NewInt(1000),
		Value:               big.NewInt(2000),
		Gas:                 50000,
		IsSystemTransaction: true,
		Data:                []byte("test data"),
	}}
	
	// Encode
	var buf bytes.Buffer
	if err := tx.encode(&buf); err != nil {
		t.Fatalf("Failed to encode DepositTxV2: %v", err)
	}
	
	// Decode
	var decoded DepositTxV2
	if err := decoded.decode(buf.Bytes()); err != nil {
		t.Fatalf("Failed to decode DepositTxV2: %v", err)
	}
	
	// Verify values match
	if decoded.SourceHash != tx.SourceHash {
		t.Error("SourceHash mismatch after RLP round trip")
	}
	if decoded.From != tx.From {
		t.Error("From mismatch after RLP round trip")
	}
	if decoded.Mint.Cmp(tx.Mint) != 0 {
		t.Error("Mint mismatch after RLP round trip")
	}
}

func TestDepositTxV2HelperMethods(t *testing.T) {
	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	sourceHash := common.HexToHash("0xdeadbeef")
	mint := big.NewInt(1000)
	
	tx := &Transaction{inner: &DepositTxV2{DepositTx{
		SourceHash:          sourceHash,
		From:                addr,
		To:                  &addr,
		Mint:                mint,
		Value:               big.NewInt(2000),
		Gas:                 50000,
		IsSystemTransaction: true,
		Data:                []byte("test data"),
	}}}
	
	// Test SourceHash()
	if tx.SourceHash() != sourceHash {
		t.Error("SourceHash() mismatch")
	}
	
	// Test Mint()
	if tx.Mint().Cmp(mint) != 0 {
		t.Error("Mint() mismatch")
	}
	
	// Test RollupCostData()
	costData := tx.RollupCostData()
	if costData != (RollupCostData{}) {
		t.Error("RollupCostData() should return zero value for deposit transactions")
	}
}

func TestDepositTxV2Signing(t *testing.T) {
	tx := &Transaction{inner: &DepositTxV2{}}
	signer := NewLondonSigner(big.NewInt(1))
	
	// Test that Sender works with V2
	addr, err := signer.Sender(tx)
	if err != nil {
		t.Fatalf("Failed to get sender: %v", err)
	}
	if addr != (common.Address{}) {
		t.Error("Sender should return zero address for unsigned deposit")
	}
	
	// Test that SignatureValues returns error
	_, _, _, err = signer.SignatureValues(tx, nil)
	if err == nil {
		t.Error("SignatureValues should return error for deposit transactions")
	}
	
	// Test that Hash panics
	defer func() {
		if r := recover(); r == nil {
			t.Error("Hash should panic for deposit transactions")
		}
	}()
	signer.Hash(tx)
}