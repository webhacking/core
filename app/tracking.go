package app

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	core "github.com/terra-project/core/types"
	"github.com/terra-project/core/x/auth"
	"github.com/terra-project/core/x/staking"
)

// organize delegations into account:amount map and undelegations into account:amount map
func organizeStaking(ctx sdk.Context, validators *staking.Validators, delegations *staking.Delegations, undelegations *staking.UnbondingDelegations) (accs map[string]sdk.Int) {
	accs = make(map[string]sdk.Int)
	tokenShareRates := make(map[string]sdk.Dec)
	for _, validator := range *validators {
		tokenShareRates[validator.GetOperator().String()] = validator.GetBondedTokens().ToDec().Quo(validator.GetDelegatorShares())
	}

	for _, delegation := range *delegations {
		delAddr := delegation.GetDelegatorAddr().String()
		valAddr := delegation.GetValidatorAddr().String()
		tokenShareRate := tokenShareRates[valAddr]
		delegationAmt := delegation.GetShares().Mul(tokenShareRate).TruncateInt()

		amt, ok := accs[delAddr]
		if ok {
			amt = amt.Add(delegationAmt)
		} else {
			amt = delegationAmt
		}

		accs[delAddr] = amt
	}

	for _, undelegation := range *undelegations {
		delAddr := undelegation.DelegatorAddress.String()
		undelegationAmt := sdk.ZeroInt()
		for _, entry := range undelegation.Entries {
			undelegationAmt = undelegationAmt.Add(entry.Balance)
		}

		amt, ok := accs[delAddr]
		if ok {
			amt = amt.Add(undelegationAmt)
		} else {
			amt = undelegationAmt
		}

		accs[delAddr] = amt
	}

	return accs
}

func (app TerraApp) getAllUnbondDelegations(ctx sdk.Context, validators staking.Validators) (undelegations staking.UnbondingDelegations) {
	for _, val := range validators {
		undelegations = append(undelegations, app.stakingKeeper.GetUnbondingDelegationsFromValidator(ctx, val.GetOperator())...)
	}
	return
}

func (app *TerraApp) exportVestingSupply(ctx sdk.Context, accs []auth.Account) {
	app.Logger().Info("Start Tracking Vesting Luna Supply")
	vestingCoins := sdk.NewCoins()
	for _, acc := range accs {
		vacc, ok := acc.(auth.VestingAccount)
		if ok {
			vestingCoins = vestingCoins.Add(vacc.GetVestingCoins(ctx.BlockHeader().Time))
		}
	}

	bz, err := codec.MarshalJSONIndent(app.cdc, vestingCoins)
	if err != nil {
		app.Logger().Error(err.Error())
	}

	err = ioutil.WriteFile(fmt.Sprintf("/tmp/vesting-%s.json", time.Now().Format(time.RFC3339)), bz, 0644)
	if err != nil {
		app.Logger().Error(err.Error())
	}

	app.Logger().Info("End Tracking Vesting Luna Supply")
}

// ExportAccount is ranking export account format
type ExportAccount struct {
	Address sdk.AccAddress `json:"address"`
	Amount  sdk.Int        `json:"amount"`
}

// NewExportAccount returns new ExportAccount instance
func NewExportAccount(address sdk.AccAddress, amount sdk.Int) ExportAccount {
	return ExportAccount{
		Address: address,
		Amount:  amount,
	}
}

func (app *TerraApp) exportRanking(ctx sdk.Context, accs []auth.Account,
	stakingMap map[string]sdk.Int, denoms []string) {
	app.Logger().Info("Start Tracking Top 1000 Rankers")

	maxEntries := 1000
	if len(accs) < maxEntries {
		maxEntries = len(accs)
	}

	for _, denom := range denoms {

		var topRankerList []ExportAccount

		tmpAccs := make([]auth.Account, len(accs))
		copy(tmpAccs, accs)

		for i := 0; i < maxEntries; i++ {

			var topRankerAmt sdk.Int
			var topRankerAddr sdk.AccAddress
			var topRankerIdx int

			for idx, acc := range tmpAccs {
				addr := acc.GetAddress()
				amt := acc.GetCoins().AmountOf(denom)

				// apply delegation & undelegation amt
				if denom == core.MicroLunaDenom {
					stakingAmt, ok := stakingMap[addr.String()]
					if ok {
						amt = amt.Add(stakingAmt)
					}
				}

				if idx == 0 || amt.GT(topRankerAmt) {
					topRankerIdx = idx
					topRankerAmt = amt
					topRankerAddr = addr
				}
			}

			topRankerList = append(topRankerList, NewExportAccount(topRankerAddr, topRankerAmt))
			tmpAccs[topRankerIdx] = tmpAccs[len(tmpAccs)-1]
			tmpAccs = tmpAccs[:len(tmpAccs)-1]
		}

		bz, err := codec.MarshalJSONIndent(app.cdc, topRankerList)
		if err != nil {
			app.Logger().Error(err.Error())
		}

		err = ioutil.WriteFile(fmt.Sprintf("/tmp/tracking-%s-%s.json", denom, time.Now().Format(time.RFC3339)), bz, 0644)
		if err != nil {
			app.Logger().Error(err.Error())
		}
	}

	app.Logger().Info("End Tracking Top 1000 Rankers")
}
