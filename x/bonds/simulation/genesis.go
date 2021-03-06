package simulation

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	"github.com/ixoworld/bonds/x/bonds/internal/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"math/rand"
)

// Simulation parameters constants
const (
	InitialBonds            = "initial_bonds"
	MaxBonds                = "max_bonds"
	MaxNumberOfInitialBonds = 100
	MaxNumberOfBonds        = 100000
)

// GenInitialNumberOfBonds randomized initial number of bonds
func GenInitialNumberOfBonds(r *rand.Rand) (initialBonds uint64) {
	return uint64(r.Int63n(MaxNumberOfInitialBonds) + 1)
}

// GenMaxNumberOfBonds randomized max number of bonds
func GenMaxNumberOfBonds(r *rand.Rand) (maxBonds uint64) {
	return uint64(r.Int63n(MaxNumberOfBonds-MaxNumberOfInitialBonds) + MaxNumberOfInitialBonds + 1)
}

// RandomizedGenState generates a random GenesisState
func RandomizedGenState(simState *module.SimulationState) {
	r := simState.Rand

	// Generate a random number of initial bonds and maximum bonds
	var initialBonds, maxBonds uint64
	simState.AppParams.GetOrGenerate(
		simState.Cdc, InitialBonds, &initialBonds, simState.Rand,
		func(r *rand.Rand) { initialBonds = GenInitialNumberOfBonds(r) },
	)
	simState.AppParams.GetOrGenerate(
		simState.Cdc, MaxBonds, &maxBonds, simState.Rand,
		func(r *rand.Rand) { maxBonds = GenMaxNumberOfBonds(r) },
	)

	if initialBonds > maxBonds {
		panic("initialBonds > maxBonds")
	}
	maxBondCount = int(maxBonds)

	var bonds []types.Bond
	var batches []types.Batch
	for i := 0; i < int(initialBonds); i++ {
		simAccount, _ := simulation.RandomAcc(r, simState.Accounts)
		address := simAccount.Address

		token := getNextBondName()
		name := getRandomNonEmptyString(r)
		desc := getRandomNonEmptyString(r)

		creator := address
		signers := []sdk.AccAddress{creator}

		var functionType string
		var reserveTokens []string
		randFunctionType := simulation.RandIntBetween(r, 0, 3)
		if randFunctionType == 0 {
			functionType = types.PowerFunction
			reserveTokens = defaultReserveTokens
		} else if randFunctionType == 1 {
			functionType = types.SigmoidFunction
			reserveTokens = defaultReserveTokens
		} else if randFunctionType == 2 {
			functionType = types.SwapperFunction
			reserveToken1, ok1 := getRandomBondName(r)
			reserveToken2, ok2 := getRandomBondNameExcept(r, reserveToken1)
			if !ok1 || !ok2 {
				initialBonds -= 1 // Ignore this iteration
				continue
			}
			reserveTokens = []string{reserveToken1, reserveToken2}
		} else {
			panic("unexpected randFunctionType")
		}
		functionParameters := getRandomFunctionParameters(r, functionType)

		// Max fee is 100, so exit fee uses 100-txFee as max
		txFeePercentage := simulation.RandomDecAmount(r, sdk.NewDec(100))
		exitFeePercentage := simulation.RandomDecAmount(r, sdk.NewDec(100).Sub(txFeePercentage))

		// Addresses
		reserveAddress := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
		feeAddress := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())

		// Max supply, allow sells, batch blocks
		maxSupply := sdk.NewCoin(token, sdk.NewInt(int64(
			simulation.RandIntBetween(r, 1000000, 1000000000))))
		allowSells := getRandomAllowSellsValue(r)
		batchBlocks := sdk.NewUint(uint64(
			simulation.RandIntBetween(r, 1, 10)))

		bond := types.NewBond(token, name, desc, creator, functionType,
			functionParameters, reserveTokens, reserveAddress, txFeePercentage,
			exitFeePercentage, feeAddress, maxSupply, blankOrderQuantityLimits,
			blankSanityRate, blankSanityMarginPercentage, allowSells, signers, batchBlocks)
		batch := types.NewBatch(bond.Token, bond.BatchBlocks)

		bonds = append(bonds, bond)
		batches = append(batches, batch)
		incrementBondCount()
		if bond.FunctionType == types.SwapperFunction {
			newSwapperBond(bond.Token)
		}
	}

	bondsGenesis := types.NewGenesisState(bonds, batches)

	fmt.Printf("Selected randomly generated bonds genesis state:\n%s\n", codec.MustMarshalJSONIndent(simState.Cdc, bondsGenesis))
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(bondsGenesis)
}
