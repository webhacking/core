// nolint:deadcode unused DONTCOVER
package simulation

import (
	"bufio"
	"encoding/csv"
	"io"
	"os"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	core "github.com/terra-project/core/types"

	"github.com/terra-project/core/x/market"
	"github.com/terra-project/core/x/market/internal/keeper"
	"github.com/terra-project/core/x/market/internal/types"
)

var (
	uSDRAmt    = sdk.NewInt(1005 * core.MicroUnit)
	stakingAmt = sdk.TokensFromConsensusPower(10)

	randomPrice = sdk.NewDec(1700)
)

func setup(t *testing.T) (keeper.TestInput, sdk.Handler) {
	input := keeper.CreateTestInput(t)
	params := input.MarketKeeper.GetParams(input.Ctx)
	input.MarketKeeper.SetParams(input.Ctx, params)
	h := market.NewHandler(input.MarketKeeper)

	return input, h
}

func readFile(t *testing.T, name string) (reader *csv.Reader) {
	csvFile, err := os.Open(name)
	require.NoError(t, err)

	reader = csv.NewReader(bufio.NewReader(csvFile))
	return
}

func TestSwapSimulation(t *testing.T) {
	input, handler := setup(t)
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroSDRDenom, sdk.NewDecWithPrec(1, 0))
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroMNTDenom, sdk.NewDecWithPrec(3200, 0))
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroKRWDenom, sdk.NewDecWithPrec(1600, 0))
	input.OracleKeeper.SetLunaExchangeRate(input.Ctx, core.MicroUSDDenom, sdk.NewDecWithPrec(12, 1))

	testcase := readFile(t, "testcase.csv")

	for {
		line, err := testcase.Read()
		if err == io.EOF {
			break
		}

		require.NoError(t, err)
		offerDenom := line[1]
		offerAmount, ok := sdk.NewIntFromString(line[2])
		require.True(t, ok)
		askDenom := line[3]
		expectedSpentAmount, err := sdk.NewDecFromStr(line[4])
		require.NoError(t, err)
		// expectedDelta := line[5]

		swapMsg := types.NewMsgSwap(keeper.Addrs[0], sdk.NewCoin(offerDenom, offerAmount), askDenom)
		res := handler(input.Ctx, swapMsg)
		require.True(t, res.IsOK())

		spentAmount := sdk.ZeroDec()
		for _, e := range res.Events {
			if e.Type == types.EventSwap {
				for _, attr := range e.Attributes {
					if string(attr.Key) == types.AttributeKeySwapCoin {
						swapCoin, err := sdk.ParseCoin(string(attr.Value))
						require.NoError(t, err)

						spentAmount = spentAmount.Add(sdk.NewDecFromInt(swapCoin.Amount))
					} else if string(attr.Key) == types.AttributeKeySwapFee {
						swapFee, err := sdk.ParseDecCoin(string(attr.Value))
						require.NoError(t, err)

						spentAmount = spentAmount.Add(swapFee.Amount)
					}
				}
			}
		}

		require.Equal(t, expectedSpentAmount, spentAmount)
	}

	// delta := sdk.NewDec(1000000)
	// regressionAmt := delta.QuoInt64(input.MarketKeeper.PoolRecoveryPeriod(input.Ctx))
	// input.MarketKeeper.SetTerraPoolDelta(input.Ctx, delta)

	// input.MarketKeeper.ReplenishPools(input.Ctx)

	// terraPoolDelta := input.MarketKeeper.GetTerraPoolDelta(input.Ctx)
	// require.Equal(t, delta.Sub(regressionAmt), terraPoolDelta)
}
