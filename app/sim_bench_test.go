package app_test

import (
	"os"
	"testing"

	"cosmossdk.io/log"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/cosmos/cosmos-sdk/testutils/sims"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	simcli "github.com/cosmos/cosmos-sdk/x/simulation/client/cli"

	"protochainecho/app"
)

// Profile with:
// `go test -benchmem -run=^$ -bench ^BenchmarkFullAppSimulation ./app -Commit=true -cpuprofile cpu.out`
func BenchmarkFullAppSimulation(b *testing.B) {
	b.ReportAllocs()

	config := simcli.NewConfigFromFlags()
	config.ChainID = SimAppChainID

	db, dir, logger, skip, err := simtestutil.SetupSimulation(config, "goleveldb-app-sim", "Simulation", simcli.FlagVerboseValue, simcli.FlagEnabledValue)
	if err != nil {
		b.Fatalf("simulation setup failed: %s", err.Error())
	}

	if skip {
		b.Skip("skipping benchmark application simulation")
	}

	defer func() {
		require.NoError(b, db.Close())
		require.NoError(b, os.RemoveAll(dir))
	}()

	appOptions := viper.New()
	appOptions.SetDefault(flags.FlagHome, app.DefaultNodeHome)
	appOptions.SetDefault(server.FlagInvCheckPeriod, simcli.FlagPeriodValue)

	bApp := app.New(logger, db, nil, true, appOptions, interBlockCacheOpt(), baseapp.SetChainID(sims.SimAppChainID))
	require.NoError(b, err)
	require.Equal(b, app.Name, bApp.Name())

	// run randomized simulation
	simParams, simErr := simulation.SimulateFromSeedX(
		b,
		log.NewNopLogger(),
		os.Stdout,
		bApp.BaseApp,
		simtestutil.AppStateFn(bApp.AppCodec(), bApp.AuthKeeper.AddressCodec(), bApp.StakingKeeper.ValidatorAddressCodec(), bApp.SimulationManager(), bApp.DefaultGenesis()),
		simtypes.RandomAccounts, // Replace with own random account function if using keys other than secp256k1
		simtestutil.SimulationOperations(bApp, bApp.AppCodec(), config, bApp.TxConfig()),
		app.BlockedAddresses(),
		config,
		bApp.AppCodec(),
		bApp.TxConfig().SigningContext().AddressCodec(),
		&simulation.DummyLogWriter{},
	)

	// export state and simParams before the simulation error is checked
	if err = simtestutil.CheckExportSimulation(bApp, config, simParams); err != nil {
		b.Fatal(err)
	}

	if simErr != nil {
		b.Fatal(simErr)
	}

	if config.Commit {
		db, ok := db.(dbm.DB)
		if ok {
			simtestutil.PrintStats(db)
		}
	}
}
