package types

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"testing"
)

func TestFunctionParamsAsMap(t *testing.T) {
	actualResult := functionParametersPower.AsMap()
	expectedResult := map[string]sdk.Int{
		"m": sdk.NewInt(12),
		"n": sdk.NewInt(2),
		"c": sdk.NewInt(100),
	}
	require.Equal(t, expectedResult, actualResult)
}

func TestFunctionParamsStringWithZeroParameters(t *testing.T) {
	testCases := []struct {
		params   FunctionParams
		expected string
	}{
		{FunctionParams{}, "{}"},
		{FunctionParams{NewFunctionParam("a", sdk.OneInt())}, "{a:1}"},
		{functionParametersPower, "{m:12,n:2,c:100}"},
		{functionParametersSigmoid, "{a:3,b:5,c:1}"},
	}
	for _, tc := range testCases {
		require.Equal(t, tc.expected, tc.params.String())
	}
}

func TestFunctionParamsAsMapReturnIsAsExpected(t *testing.T) {
	actualResult := functionParametersPower.AsMap()
	expectedResult := map[string]sdk.Int{"m": sdk.NewInt(12), "n": sdk.NewInt(2), "c": sdk.NewInt(100)}
	require.Equal(t, expectedResult, actualResult)
}

func TestNewBondDefaultValuesAndSorting(t *testing.T) {
	customReserveTokens := []string{"b", "a"}
	customOrderQuantityLimits, _ := sdk.ParseCoins("100bbb,100aaa")
	sortedReserveTokens := []string{"a", "b"}
	sortedOrderQuantityLimits, _ := sdk.ParseCoins("100aaa,100bbb")

	bond := NewBond(initToken, initName, initDescription,
		initCreator, PowerFunction, functionParametersPower,
		customReserveTokens, initReserveAddress, initTxFeePercentage,
		initExitFeePercentage, initFeeAddress, initMaxSupply,
		customOrderQuantityLimits, initSanityRate, initSanityMarginPercentage,
		initAllowSell, initSigners, initBatchBlocks)

	expectedCurrentSupply := sdk.NewInt64Coin(bond.Token, 0)

	require.Equal(t, expectedCurrentSupply, bond.CurrentSupply)
	require.Equal(t, sortedReserveTokens, bond.ReserveTokens)
	require.Equal(t, sortedOrderQuantityLimits, bond.OrderQuantityLimits)
}

func TestGetNewReserveCoinReturnPasses(t *testing.T) {
	bond := getValidBond()

	require.Equal(t, sdk.NewInt64Coin(bond.Token, 0), bond.CurrentSupply)
}

func TestGetNewReserveDecCoins(t *testing.T) {
	bond := getValidBond()
	bond.ReserveTokens = []string{"aaa", "bbb"}

	amount := sdk.MustNewDecFromStr("10")
	actualResult := bond.GetNewReserveDecCoins(amount)

	expectedResult := sdk.NewDecCoins(sdk.NewCoins(
		sdk.NewInt64Coin("aaa", 10),
		sdk.NewInt64Coin("bbb", 10),
	))

	require.Equal(t, expectedResult, actualResult)
}

func TestGetPriceAtSupply(t *testing.T) {
	bond := getValidBond()
	// TODO: add more test cases

	testCases := []struct {
		functionType      string
		functionParams    FunctionParams
		reserveTokens     []string
		supply            sdk.Int
		expected          string
		functionAvailable bool
	}{
		{PowerFunction, functionParametersPower, multitokenReserve, sdk.NewInt(0), "100", true},
		{PowerFunction, functionParametersPower, multitokenReserve, sdk.NewInt(1000), "12000100", true},
		{SigmoidFunction, functionParametersSigmoid, multitokenReserve, sdk.NewInt(1000), "5.999998484889207399", true},
		{SwapperFunction, nil, swapperReserves, sdk.NewInt(100), "100", false},
	}
	for _, tc := range testCases {
		bond.FunctionType = tc.functionType
		bond.FunctionParameters = tc.functionParams
		bond.ReserveTokens = tc.reserveTokens

		actualResult, err := bond.GetPricesAtSupply(tc.supply)
		if tc.functionAvailable {
			require.Nil(t, err)
			expectedDec := sdk.MustNewDecFromStr(tc.expected)
			expectedResult := NewDecMultitokenReserveFromDec(expectedDec)
			require.Equal(t, expectedResult, actualResult)
		} else {
			require.Error(t, err)
		}
	}
}

func TestGetCurrentPrices(t *testing.T) {
	bond := getValidBond()
	// TODO: add more test cases

	swapperReserveBalances := sdk.NewCoins(
		sdk.NewInt64Coin(reserveToken, 10000),
		sdk.NewInt64Coin(reserveToken2, 10000),
	)

	testCases := []struct {
		functionType    string
		functionParams  FunctionParams
		reserveTokens   []string
		currentSupply   sdk.Int
		reserveBalances sdk.Coins
		expected        string
	}{
		{PowerFunction, functionParametersPower, multitokenReserve, sdk.NewInt(100), nil, "120100"},
		{SigmoidFunction, functionParametersSigmoid, multitokenReserve, sdk.NewInt(100), nil, "5.999833808828064549"},
		{SwapperFunction, nil, swapperReserves, sdk.NewInt(100), swapperReserveBalances, "100"},
	}
	for _, tc := range testCases {
		bond.FunctionType = tc.functionType
		bond.FunctionParameters = tc.functionParams
		bond.ReserveTokens = tc.reserveTokens
		bond.CurrentSupply = sdk.NewCoin(bond.Token, tc.currentSupply)

		actualResult, _ := bond.GetCurrentPricesPT(tc.reserveBalances)
		expectedDec := sdk.MustNewDecFromStr(tc.expected)
		expectedResult := NewDecMultitokenReserveFromDec(expectedDec)
		require.Equal(t, expectedResult, actualResult)
	}
}

func TestCurveIntegral(t *testing.T) {
	bond := getValidBond()

	testCases := []struct {
		functionType   string
		functionParams FunctionParams
		supply         sdk.Int
		expected       string
	}{
		{PowerFunction, functionParametersPower, sdk.NewInt(100), "4010000"},
		{PowerFunction, functionParametersPower, maxInt64, "3138550867693340380897047610841017818694071568064447512472.0"},
		{PowerFunction, functionParametersPowerHuge, sdk.NewInt(5), "390525200604461289807786418456824866174854670846050992460534124091120.049504950495049505"},
		//{PowerFunction, functionParametersPowerHuge, sdk.NewInt(6), ""}, // causes integer overflow

		{SigmoidFunction, functionParametersSigmoid, sdk.NewInt(100), "569.718730497"},
		{SigmoidFunction, functionParametersSigmoid, maxInt64, "55340232221128654811.702941461"},
		{SigmoidFunction, functionParametersSigmoidHuge, sdk.NewInt(1), "13043817821891587770.728894534000000000"},
		{SigmoidFunction, functionParametersSigmoidHuge, maxInt64, "170141183460469231685570443531610226691.0"},
	}
	for _, tc := range testCases {
		bond.FunctionType = tc.functionType
		bond.FunctionParameters = tc.functionParams

		actualResult := bond.CurveIntegral(tc.supply)
		expectedResult := sdk.MustNewDecFromStr(tc.expected)
		require.Equal(t, expectedResult, actualResult)
	}
}

func TestGetReserveDeltaForLiquidityDelta(t *testing.T) {
	bond := getValidBond()
	bond.FunctionType = SwapperFunction
	bond.ReserveTokens = swapperReserves
	// TODO: add more test cases

	reserveBalances := sdk.NewCoins(
		sdk.NewInt64Coin(reserveToken, 10000),
		sdk.NewInt64Coin(reserveToken2, 10000),
	)

	testCases := []struct {
		currentSupply  sdk.Int
		liquidityDelta sdk.Int
	}{
		{sdk.NewInt(2), sdk.NewInt(10)},
	}
	for _, tc := range testCases {
		bond.CurrentSupply = sdk.NewCoin(bond.Token, tc.currentSupply)

		actualResult := bond.GetReserveDeltaForLiquidityDelta(tc.liquidityDelta, reserveBalances)
		expectedResult := NewDecMultitokenReserveFromInt(50000)
		require.Equal(t, expectedResult, actualResult)
	}
}

func TestGetPricesToMint(t *testing.T) {
	bond := getValidBond()
	// TODO: add more test cases

	reserveBalances1000 := sdk.NewCoins(
		sdk.NewInt64Coin(reserveToken, 10000),
		sdk.NewInt64Coin(reserveToken2, 10000),
	)
	reserveBalances10 := sdk.NewCoins(
		sdk.NewInt64Coin(reserveToken, 10),
		sdk.NewInt64Coin(reserveToken2, 10),
	)

	testCases := []struct {
		functionType    string
		functionParams  FunctionParams
		reserveTokens   []string
		reserveBalances sdk.Coins
		currentSupply   sdk.Int
		amount          sdk.Int
		expectedPrice   string
		fails           bool
	}{
		{PowerFunction, functionParametersPower, multitokenReserve, reserveBalances1000, sdk.ZeroInt(), sdk.NewInt(100), "4000000", false},
		{PowerFunction, functionParametersPower, multitokenReserve, nil, sdk.ZeroInt(), sdk.NewInt(100), "4010000", false},
		{SigmoidFunction, functionParametersSigmoid, multitokenReserve, nil, sdk.ZeroInt(), sdk.NewInt(100), "569.718730497", false},
		{SigmoidFunction, functionParametersSigmoid, multitokenReserve, reserveBalances10, sdk.ZeroInt(), sdk.NewInt(100), "559.718730497", false},
		{SwapperFunction, FunctionParams{}, swapperReserves, reserveBalances1000, sdk.NewInt(2), sdk.NewInt(10), "50000", false},
		{SwapperFunction, FunctionParams{}, swapperReserves, nil, sdk.NewInt(2), sdk.NewInt(10), "0", false}, // impossible scenario
		{SwapperFunction, FunctionParams{}, swapperReserves, nil, sdk.ZeroInt(), sdk.NewInt(10), "0", true},
	}
	for _, tc := range testCases {
		bond.FunctionType = tc.functionType
		bond.FunctionParameters = tc.functionParams
		bond.ReserveTokens = tc.reserveTokens
		bond.CurrentSupply = sdk.NewCoin(bond.Token, tc.currentSupply)

		actualResult, err := bond.GetPricesToMint(tc.amount, tc.reserveBalances)
		if tc.fails {
			require.Error(t, err)
		} else {
			require.Nil(t, err)
			expectedDec := sdk.MustNewDecFromStr(tc.expectedPrice)
			expectedResult := NewDecMultitokenReserveFromDec(expectedDec)
			require.Equal(t, expectedResult, actualResult)
		}
	}
}

func TestGetReturnsForBurn(t *testing.T) {
	bond := getValidBond()
	// TODO: add more test cases

	reserveBalances232 := sdk.NewCoins(
		sdk.NewInt64Coin(reserveToken, 232),
		sdk.NewInt64Coin(reserveToken2, 232),
	)

	swapperReserveBalances := sdk.NewCoins(
		sdk.NewInt64Coin(reserveToken, 10000),
		sdk.NewInt64Coin(reserveToken2, 10000),
	)

	testCases := []struct {
		functionType    string
		functionParams  FunctionParams
		reserveTokens   []string
		reserveBalances sdk.Coins
		currentSupply   sdk.Int
		amount          sdk.Int
		expectedReturn  string
	}{
		{PowerFunction, functionParametersPower, multitokenReserve, reserveBalances232, sdk.NewInt(2), sdk.OneInt(), "128"},
		{SigmoidFunction, functionParametersSigmoid, multitokenReserve, reserveBalances232, sdk.NewInt(2), sdk.OneInt(), "231.927741664"},
		{SwapperFunction, FunctionParams{}, swapperReserves, swapperReserveBalances, sdk.NewInt(2), sdk.OneInt(), "5000"},
	}
	for _, tc := range testCases {
		bond.FunctionType = tc.functionType
		bond.FunctionParameters = tc.functionParams
		bond.ReserveTokens = tc.reserveTokens
		bond.CurrentSupply = sdk.NewCoin(bond.Token, tc.currentSupply)

		actualResult := bond.GetReturnsForBurn(tc.amount, tc.reserveBalances)
		expectedDec := sdk.MustNewDecFromStr(tc.expectedReturn)
		expectedResult := NewDecMultitokenReserveFromDec(expectedDec)
		require.Equal(t, expectedResult, actualResult)
	}
}

func TestGetReturnsForSwap(t *testing.T) {
	bond := getValidBond()
	bond.FunctionType = SwapperFunction
	bond.FunctionParameters = nil
	bond.ReserveTokens = swapperReserves

	reserveBalances := sdk.NewCoins(
		sdk.NewInt64Coin(reserveToken, 10000),
		sdk.NewInt64Coin(reserveToken2, 10000),
	)

	zeroPoint1Percent := sdk.MustNewDecFromStr("0.001")
	largeInput := maxInt64
	largeFee := sdk.NewDecFromInt(largeInput).Mul(zeroPoint1Percent).Ceil().TruncateInt()
	smallInput := sdk.NewInt(3) // but not too small
	smallFee := sdk.NewDecFromInt(smallInput).Mul(zeroPoint1Percent).Ceil().TruncateInt()

	testCases := []struct {
		bondTxFee           string
		from                string
		to                  string
		amount              sdk.Int
		expectedReturn      sdk.Int
		expectedFee         sdk.Int
		amountInvalid       bool // too large or too small
		invalidReserveToken bool
	}{
		{"0.1", reserveToken, reserveToken2, smallInput, sdk.OneInt(), smallFee, false, false},
		{"0.1", reserveToken, reserveToken2, sdk.NewInt(2), sdk.OneInt(), sdk.OneInt(), true, false},
		{"0.1", reserveToken, reserveToken2, sdk.NewInt(1), sdk.OneInt(), sdk.OneInt(), true, false},
		{"0.1", reserveToken, reserveToken2, sdk.NewInt(0), sdk.OneInt(), sdk.OneInt(), true, false},
		{"0.1", reserveToken, reserveToken2, largeInput, sdk.NewInt(9999), largeFee, false, false},
		{"0.1", reserveToken, "dummytoken", sdk.NewInt(3), sdk.OneInt(), sdk.OneInt(), false, true},  // identical to first case but dummytoken
		{"0.1", "dummytoken", reserveToken2, sdk.NewInt(3), sdk.OneInt(), sdk.OneInt(), false, true}, // identical to first case but dummytoken
	}
	for _, tc := range testCases {
		bond.TxFeePercentage = sdk.MustNewDecFromStr(tc.bondTxFee)
		fromAmount := sdk.NewCoin(tc.from, tc.amount)
		actualResult, actualFee, err := bond.GetReturnsForSwap(fromAmount, tc.to, reserveBalances)
		if tc.amountInvalid {
			require.Error(t, err)
			require.Equal(t, err.Code(), CodeSwapAmountInvalid)
		} else if tc.invalidReserveToken {
			require.Error(t, err)
			require.Equal(t, err.Code(), CodeReserveTokenInvalid)
		} else {
			require.Nil(t, err)
			expectedResult := sdk.NewCoins(sdk.NewCoin(tc.to, tc.expectedReturn))
			expectedFee := sdk.NewCoin(tc.from, tc.expectedFee)
			require.Equal(t, expectedResult, actualResult)
			require.Equal(t, expectedFee, actualFee)
		}
	}
}

func TestGetReturnsForSwapNonSwapperFunctionFails(t *testing.T) {
	bond := getValidBond()
	testCases := []string{PowerFunction, SigmoidFunction}

	for _, tc := range testCases {
		bond.FunctionType = tc

		dummyCoin := sdk.NewCoin(reserveToken, sdk.OneInt()) // to avoid panic

		_, _, err := bond.GetReturnsForSwap(dummyCoin, "", sdk.Coins{})
		require.Error(t, err)
		require.False(t, err.Result().IsOK())
		require.Equal(t, err.Code(), CodeFunctionNotAvailableForFunctionType)
	}
}

func TestBondGetTxFee(t *testing.T) {
	bond := Bond{}
	zeroPointOne := sdk.MustNewDecFromStr("0.1")

	// Fee is always rounded to ceiling, so for any input N > 0, fee(N) > 0

	testCases := []struct {
		input           string
		txFeePercentage sdk.Dec
		expected        int64
	}{

		{"2000000000000", zeroPointOne, 2000000000},
		{"2000", zeroPointOne, 2},
		{"200", zeroPointOne, 1},      // 200 * 0.1 = 0.2 = 1 (rounded)
		{"20", zeroPointOne, 1},       // 20 * 0.1 = 00.2 = 1 (rounded)
		{"0.000002", zeroPointOne, 1}, // 0.000002 * 0.1 = small number = 1 (rounded)
		{"0", zeroPointOne, 0},
		{"2000", sdk.ZeroDec(), 0},
		{"0.000002", sdk.ZeroDec(), 0},
	}
	for _, tc := range testCases {
		inputToken := sdk.NewDecCoinFromDec(reserveToken, sdk.MustNewDecFromStr(tc.input))
		expected := sdk.NewInt64Coin(reserveToken, tc.expected)

		bond.TxFeePercentage = tc.txFeePercentage
		require.Equal(t, expected, bond.GetTxFee(inputToken))
	}
}

func TestBondGetExitFee(t *testing.T) {
	bond := Bond{}
	zeroPointOne := sdk.MustNewDecFromStr("0.1")

	// Fee is always rounded to ceiling, so for any input N > 0, fee(N) > 0

	testCases := []struct {
		input             string
		exitFeePercentage sdk.Dec
		expected          int64
	}{
		{"2000000000000", zeroPointOne, 2000000000},
		{"2000", zeroPointOne, 2},
		{"200", zeroPointOne, 1},      // 200 * 0.1 = 0.2 = 1 (rounded)
		{"20", zeroPointOne, 1},       // 20 * 0.1 = 00.2 = 1 (rounded)
		{"0.000002", zeroPointOne, 1}, // 0.000002 * 0.1 = small number = 1 (rounded)
		{"0", zeroPointOne, 0},
		{"2000", sdk.ZeroDec(), 0},
		{"0.000002", sdk.ZeroDec(), 0},
	}
	for _, tc := range testCases {
		inputToken := sdk.NewDecCoinFromDec(reserveToken, sdk.MustNewDecFromStr(tc.input))
		expected := sdk.NewInt64Coin(reserveToken, tc.expected)

		bond.ExitFeePercentage = tc.exitFeePercentage
		require.Equal(t, expected, bond.GetExitFee(inputToken))
	}
}

func TestBondGetTxFees(t *testing.T) {
	bond := Bond{}
	bond.TxFeePercentage = sdk.MustNewDecFromStr("0.1")

	// Fee is always rounded to ceiling, so for any input N > 0, fee(N) > 0

	inputTokens, err := sdk.ParseDecCoins("" +
		"200000000.0aaa," +
		"2000.0bbb," +
		"200.0ccc," +
		"20.0ddd," +
		"0.000002eee")
	require.Nil(t, err)

	expected, err := sdk.ParseCoins("" +
		"200000aaa," +
		"2bbb," +
		"1ccc," +
		"1ddd," +
		"1eee")
	require.Nil(t, err)

	require.Equal(t, expected, bond.GetTxFees(inputTokens))
}

func TestBondGetExitFees(t *testing.T) {
	bond := Bond{}
	bond.ExitFeePercentage = sdk.MustNewDecFromStr("0.1")

	// Fee is always rounded to ceiling, so for any input N > 0, fee(N) > 0

	inputTokens, err := sdk.ParseDecCoins("" +
		"200000000.0aaa," +
		"2000.0bbb," +
		"200.0ccc," +
		"20.0ddd," +
		"0.000002eee")
	require.Nil(t, err)

	expected, err := sdk.ParseCoins("" +
		"200000aaa," +
		"2bbb," +
		"1ccc," +
		"1ddd," +
		"1eee")
	require.Nil(t, err)

	require.Equal(t, expected, bond.GetExitFees(inputTokens))
}

func TestSignersEqualTo(t *testing.T) {
	bond := getValidBond()

	addr1 := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
	addr2 := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
	addr3 := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
	bond.Signers = []sdk.AccAddress{addr1, addr2}

	testCases := []struct {
		toCompareTo   []sdk.AccAddress
		expectedEqual bool
	}{
		{[]sdk.AccAddress{addr1}, false},               // One missing
		{[]sdk.AccAddress{addr1, addr2, addr3}, false}, // One extra
		{[]sdk.AccAddress{addr1, addr3}, false},        // One different
		{[]sdk.AccAddress{addr2, addr1}, false},        // Incorrect order
		{[]sdk.AccAddress{addr1, addr2}, true},         // Equal
	}
	for _, tc := range testCases {
		require.Equal(t, tc.expectedEqual, bond.SignersEqualTo(tc.toCompareTo))
	}
}

func TestReserveDenomsEqualTo(t *testing.T) {
	bond := getValidBond()

	denom1 := reserveToken
	denom2 := reserveToken2
	denom3 := reserveToken3
	bond.ReserveTokens = []string{denom1, denom2}

	testCases := []struct {
		toCompareTo   []string
		expectedEqual bool
	}{
		{[]string{denom1}, false},                 // One missing
		{[]string{denom1, denom2, denom3}, false}, // One extra
		{[]string{denom1, denom3}, false},         // One different
		{[]string{denom2, denom1}, true},          // Incorrect order (allowed)
		{[]string{denom1, denom2}, true},          // Equal
	}
	for _, tc := range testCases {
		coins := sdk.Coins{}
		for _, res := range tc.toCompareTo {
			coins = coins.Add(sdk.Coins{sdk.NewCoin(res, sdk.OneInt())})
		}
		require.Equal(t, tc.expectedEqual, bond.ReserveDenomsEqualTo(coins))
	}
}

func TestAnyOrderQuantityLimitsExceeded(t *testing.T) {
	bond := getValidBond()
	bond.OrderQuantityLimits, _ = sdk.ParseCoins("100aaa,200bbb")

	testCases := []struct {
		amounts         string
		exceedsAnyLimit bool
	}{
		{"99aaa", false},         // aaa <= 100
		{"100aaa", false},        // aaa <= 100
		{"101aaa", true},         // aaa >  100
		{"101bbb", false},        // bbb <= 200
		{"100aaa,200bbb", false}, // aaa <= 100, bbb <= 200
		{"101aaa,200bbb", true},  // aaa >  100, bbb <= 200
		{"100aaa,201bbb", true},  // aaa <= 100, bbb >  200
		{"101aaa,201bbb", true},  // aaa >  100, bbb >  200
	}
	for _, tc := range testCases {
		amounts, _ := sdk.ParseCoins(tc.amounts)
		require.Equal(t, tc.exceedsAnyLimit, bond.AnyOrderQuantityLimitsExceeded(amounts))
	}
}

func TestReservesViolateSanityRateReturnsFalseWhenSanityRateIsZero(t *testing.T) {
	bond := getValidBond()

	r1 := reserveToken
	r2 := reserveToken2
	bond.ReserveTokens = []string{r1, r2}

	testCases := []struct {
		reserves               string
		sanityRate             string
		sanityMarginPercentage string
		violates               bool
	}{
		{fmt.Sprintf(" 500%s,1000%s", r1, r2), "0", "0", false},     // no sanity checks
		{fmt.Sprintf("1000%s,1000%s", r1, r2), "0", "0", false},     // no sanity checks
		{fmt.Sprintf(" 500%s,1000%s", r1, r2), "0.5", "0", false},   //  500/1000 == 0.5
		{fmt.Sprintf("1000%s,1000%s", r1, r2), "0.5", "0", true},    // 1000/1000 != 0.5
		{fmt.Sprintf(" 100%s,1000%s", r1, r2), "0.5", "0", true},    //  100/1000 != 0.5
		{fmt.Sprintf(" 100%s,1000%s", r1, r2), "0.5", "79", true},   // 0.5+-79% => 0.105 to 0.895, and 100/1000 is in not this range
		{fmt.Sprintf(" 100%s,1000%s", r1, r2), "0.5", "80", false},  // 0.5+-80% => 0.100 to 0.900, and 100/1000 is in this range
		{fmt.Sprintf(" 100%s,1000%s", r1, r2), "0.5", "81", false},  // 0.5+-81% => 0.095 to 0.905, and 100/1000 is in this range
		{fmt.Sprintf(" 100%s,1000%s", r1, r2), "0.5", "101", false}, // identical to above but negative lower limit gets rounded to 0
	}
	for _, tc := range testCases {
		reserves, _ := sdk.ParseCoins(tc.reserves)
		srDec := sdk.MustNewDecFromStr(tc.sanityRate)
		smpDec := sdk.MustNewDecFromStr(tc.sanityMarginPercentage)

		bond.SanityRate = srDec
		bond.SanityMarginPercentage = smpDec

		actualResult := bond.ReservesViolateSanityRate(reserves)
		require.Equal(t, tc.violates, actualResult)
	}
}
