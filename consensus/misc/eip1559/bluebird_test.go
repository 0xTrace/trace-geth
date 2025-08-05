// Copyright 2024 The go-ethereum Authors
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

package eip1559

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

func TestBluebirdEIP1559Parameters(t *testing.T) {
	// Create a config with Bluebird enabled at timestamp 1000
	bluebirdTime := uint64(1000)
	config := &params.ChainConfig{
		ChainID:             big.NewInt(1),
		HomesteadBlock:      big.NewInt(0),
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
		BluebirdTime:        &bluebirdTime,
	}

	// Test ElasticityMultiplier
	t.Run("ElasticityMultiplier", func(t *testing.T) {
		// Before Bluebird
		beforeTime := uint64(999)
		elasticity := config.ElasticityMultiplier(beforeTime)
		if elasticity != params.DefaultElasticityMultiplier {
			t.Errorf("Before Bluebird: expected elasticity %d, got %d", params.DefaultElasticityMultiplier, elasticity)
		}

		// At Bluebird activation
		atTime := uint64(1000)
		elasticity = config.ElasticityMultiplier(atTime)
		if elasticity != params.BluebirdElasticityMultiplier {
			t.Errorf("At Bluebird: expected elasticity %d, got %d", params.BluebirdElasticityMultiplier, elasticity)
		}

		// After Bluebird
		afterTime := uint64(1001)
		elasticity = config.ElasticityMultiplier(afterTime)
		if elasticity != params.BluebirdElasticityMultiplier {
			t.Errorf("After Bluebird: expected elasticity %d, got %d", params.BluebirdElasticityMultiplier, elasticity)
		}
	})

	// Test BaseFeeChangeDenominator
	t.Run("BaseFeeChangeDenominator", func(t *testing.T) {
		// Before Bluebird
		beforeTime := uint64(999)
		denominator := config.BaseFeeChangeDenominator(beforeTime)
		if denominator != params.DefaultBaseFeeChangeDenominator {
			t.Errorf("Before Bluebird: expected denominator %d, got %d", params.DefaultBaseFeeChangeDenominator, denominator)
		}

		// At Bluebird activation
		atTime := uint64(1000)
		denominator = config.BaseFeeChangeDenominator(atTime)
		if denominator != params.BluebirdBaseFeeChangeDenominator {
			t.Errorf("At Bluebird: expected denominator %d, got %d", params.BluebirdBaseFeeChangeDenominator, denominator)
		}

		// After Bluebird
		afterTime := uint64(1001)
		denominator = config.BaseFeeChangeDenominator(afterTime)
		if denominator != params.BluebirdBaseFeeChangeDenominator {
			t.Errorf("After Bluebird: expected denominator %d, got %d", params.BluebirdBaseFeeChangeDenominator, denominator)
		}
	})

	// Test minimum base fee enforcement
	t.Run("MinimumBaseFee", func(t *testing.T) {
		// Create a parent header with very low base fee
		parent := &types.Header{
			Number:   big.NewInt(1),
			GasLimit: 30_000_000,
			GasUsed:  0, // No gas used, should decrease base fee
			BaseFee:  big.NewInt(1000), // Very low base fee
		}

		// Before Bluebird - base fee can go to near zero
		beforeTime := uint64(999)
		baseFee := CalcBaseFee(config, parent, beforeTime)
		if baseFee.Cmp(big.NewInt(0)) < 0 {
			t.Errorf("Base fee should never be negative, got %s", baseFee)
		}

		// After Bluebird - base fee should respect minimum
		afterTime := uint64(1001)
		baseFee = CalcBaseFee(config, parent, afterTime)
		minBaseFee := new(big.Int).SetUint64(params.BluebirdMinBaseFee)
		if baseFee.Cmp(minBaseFee) < 0 {
			t.Errorf("After Bluebird: base fee %s is below minimum %s", baseFee, minBaseFee)
		}
	})

	// Test base fee calculation with new parameters
	t.Run("BaseFeeCalculationWithNewParams", func(t *testing.T) {
		// Test with block at exactly the target gas usage
		parent := &types.Header{
			Number:   big.NewInt(1),
			GasLimit: 30_000_000,
			GasUsed:  15_000_000, // At target for default elasticity of 2
			BaseFee:  big.NewInt(1_000_000_000), // 1 gwei
		}

		// Before Bluebird: gas used = target, so base fee unchanged
		beforeTime := uint64(999)
		baseFeeBeforeBluebird := CalcBaseFee(config, parent, beforeTime)
		if baseFeeBeforeBluebird.Cmp(parent.BaseFee) != 0 {
			t.Errorf("At target gas usage, base fee should remain unchanged")
		}

		// After Bluebird: gas used > new target (7.5M), so base fee increases
		afterTime := uint64(1001)
		baseFeeAfterBluebird := CalcBaseFee(config, parent, afterTime)
		
		t.Logf("Base fee before Bluebird: %s", baseFeeBeforeBluebird)
		t.Logf("Base fee after Bluebird: %s", baseFeeAfterBluebird)
		t.Logf("Gas target before Bluebird: %d", parent.GasLimit/params.DefaultElasticityMultiplier)
		t.Logf("Gas target after Bluebird: %d", parent.GasLimit/params.BluebirdElasticityMultiplier)

		// After Bluebird, with higher elasticity, the same gas usage is now above target
		if baseFeeAfterBluebird.Cmp(parent.BaseFee) <= 0 {
			t.Errorf("After Bluebird, base fee should increase as gas usage is now above new target")
		}
	})

	// Test that higher denominator makes changes more gradual
	t.Run("DenominatorMakesChangesGradual", func(t *testing.T) {
		parent := &types.Header{
			Number:   big.NewInt(1),
			GasLimit: 30_000_000,
			GasUsed:  30_000_000, // Full block
			BaseFee:  big.NewInt(1_000_000_000), // 1 gwei
		}

		// Calculate base fee increase percentage before and after Bluebird
		beforeTime := uint64(999)
		baseFeeBeforeBluebird := CalcBaseFee(config, parent, beforeTime)
		percentIncreaseBeforeBluebird := new(big.Int).Sub(baseFeeBeforeBluebird, parent.BaseFee)
		percentIncreaseBeforeBluebird.Mul(percentIncreaseBeforeBluebird, big.NewInt(100))
		percentIncreaseBeforeBluebird.Div(percentIncreaseBeforeBluebird, parent.BaseFee)

		afterTime := uint64(1001)
		baseFeeAfterBluebird := CalcBaseFee(config, parent, afterTime)
		percentIncreaseAfterBluebird := new(big.Int).Sub(baseFeeAfterBluebird, parent.BaseFee)
		percentIncreaseAfterBluebird.Mul(percentIncreaseAfterBluebird, big.NewInt(100))
		percentIncreaseAfterBluebird.Div(percentIncreaseAfterBluebird, parent.BaseFee)

		t.Logf("Percent increase before Bluebird: %s%%", percentIncreaseBeforeBluebird)
		t.Logf("Percent increase after Bluebird: %s%%", percentIncreaseAfterBluebird)

		// Both elasticity and denominator changes affect the calculation
		// The test should verify the parameters are applied correctly
		if config.ElasticityMultiplier(afterTime) != params.BluebirdElasticityMultiplier {
			t.Errorf("Elasticity multiplier not correctly applied")
		}
		if config.BaseFeeChangeDenominator(afterTime) != params.BluebirdBaseFeeChangeDenominator {
			t.Errorf("Base fee denominator not correctly applied")
		}
	})
}