package keeper

import (
	"math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"

	core "github.com/terra-project/core/types"
	"github.com/terra-project/core/x/treasury/internal/types"
)

// NewQuerier is the module level router for state queries
func NewQuerier(keeper Keeper) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) (res []byte, err sdk.Error) {
		switch path[0] {
		case types.QueryTaxRate:
			return queryTaxRate(ctx, keeper)
		case types.QueryTaxCap:
			return queryTaxCap(ctx, req, keeper)
		case types.QueryRewardWeight:
			return queryRewardWeight(ctx, keeper)
		case types.QuerySeigniorageProceeds:
			return querySeigniorageProceeds(ctx, keeper)
		case types.QueryTaxProceeds:
			return queryTaxProceeds(ctx, keeper)
		case types.QueryParameters:
			return queryParameters(ctx, keeper)
		case types.QueryIndicators:
			return queryIndicators(ctx, keeper)
		default:
			return nil, sdk.ErrUnknownRequest("unknown treasury query endpoint")
		}
	}
}

func queryIndicators(ctx sdk.Context, keeper Keeper) ([]byte, sdk.Error) {
	// Compute Total Staked Luna (TSL)
	TSL := keeper.stakingKeeper.TotalBondedTokens(ctx)

	// Compute Tax Rewards (TR)
	taxRewards := sdk.NewDecCoins(keeper.PeekEpochTaxProceeds(ctx))
	TR := keeper.alignCoins(ctx, taxRewards, core.MicroSDRDenom)

	epoch := core.GetEpoch(ctx)
	var res types.IndicatorQueryResonse
	if epoch == 0 {
		res = types.IndicatorQueryResonse{
			TRLYear:  TR.QuoInt(TSL),
			TRLMonth: TR.QuoInt(TSL),
		}
	} else {
		params := keeper.GetParams(ctx)
		previousEpochCtx := ctx.WithBlockHeight(ctx.BlockHeight() - core.BlocksPerEpoch)
		trlYear := keeper.rollingAverageIndicator(previousEpochCtx, params.WindowLong-1, TRL)
		trlMonth := keeper.rollingAverageIndicator(previousEpochCtx, params.WindowShort-1, TRL)

		computedEpochForYear := int64(math.Min(float64(params.WindowLong-1), float64(epoch)))
		computedEpochForMonty := int64(math.Min(float64(params.WindowShort-1), float64(epoch)))

		trlYear = trlYear.MulInt64(computedEpochForYear).Add(TR.QuoInt(TSL)).QuoInt64(computedEpochForYear + 1)
		trlMonth = trlMonth.MulInt64(computedEpochForMonty).Add(TR.QuoInt(TSL)).QuoInt64(computedEpochForMonty + 1)

		res = types.IndicatorQueryResonse{
			TRLYear:  trlYear,
			TRLMonth: trlMonth,
		}
	}

	bz, err := codec.MarshalJSONIndent(keeper.cdc, res)
	if err != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", err.Error()))
	}
	return bz, nil
}

func queryTaxRate(ctx sdk.Context, keeper Keeper) ([]byte, sdk.Error) {
	taxRate := keeper.GetTaxRate(ctx)
	bz, err := codec.MarshalJSONIndent(keeper.cdc, taxRate)
	if err != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", err.Error()))
	}

	return bz, nil
}

func queryTaxCap(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	var params types.QueryTaxCapParams
	err := keeper.cdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdk.ErrUnknownRequest(sdk.AppendMsgToErr("incorrectly formatted request data", err.Error()))
	}

	taxCap := keeper.GetTaxCap(ctx, params.Denom)
	bz, err := codec.MarshalJSONIndent(keeper.cdc, taxCap)
	if err != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", err.Error()))
	}

	return bz, nil
}

func queryRewardWeight(ctx sdk.Context, keeper Keeper) ([]byte, sdk.Error) {
	taxRate := keeper.GetRewardWeight(ctx)
	bz, err := codec.MarshalJSONIndent(keeper.cdc, taxRate)
	if err != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", err.Error()))
	}

	return bz, nil
}

func querySeigniorageProceeds(ctx sdk.Context, keeper Keeper) ([]byte, sdk.Error) {
	seigniorage := keeper.PeekEpochSeigniorage(ctx)
	bz, err := codec.MarshalJSONIndent(keeper.cdc, seigniorage)
	if err != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", err.Error()))
	}
	return bz, nil
}

func queryTaxProceeds(ctx sdk.Context, keeper Keeper) ([]byte, sdk.Error) {
	proceeds := keeper.PeekEpochTaxProceeds(ctx)
	bz, err := codec.MarshalJSONIndent(keeper.cdc, proceeds)
	if err != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", err.Error()))
	}
	return bz, nil
}

func queryParameters(ctx sdk.Context, keeper Keeper) ([]byte, sdk.Error) {
	bz, err := codec.MarshalJSONIndent(keeper.cdc, keeper.GetParams(ctx))
	if err != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", err.Error()))
	}
	return bz, nil
}
