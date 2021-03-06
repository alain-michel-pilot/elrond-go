package process

import (
	"bytes"
	"encoding/hex"
	"math"
	"math/big"
	"testing"

	arwenConfig "github.com/ElrondNetwork/arwen-wasm-vm/config"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/state"
	factoryState "github.com/ElrondNetwork/elrond-go/data/state/factory"
	"github.com/ElrondNetwork/elrond-go/data/trie"
	"github.com/ElrondNetwork/elrond-go/data/trie/factory"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/genesis"
	"github.com/ElrondNetwork/elrond-go/genesis/mock"
	"github.com/ElrondNetwork/elrond-go/genesis/parsing"
	"github.com/ElrondNetwork/elrond-go/process/economics"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/vm/systemSmartContracts/defaults"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var nodePrice = big.NewInt(5000)

//TODO improve code coverage of this package
func createMockArgument(
	t *testing.T,
	genesisFilename string,
	initialNodes genesis.InitialNodesHandler,
	entireSupply *big.Int,
) ArgsGenesisBlockCreator {

	memDBMock := mock.NewMemDbMock()
	storageManager, _ := trie.NewTrieStorageManagerWithoutPruning(memDBMock)

	trieStorageManagers := make(map[string]data.StorageManager)
	trieStorageManagers[factory.UserAccountTrie] = storageManager
	trieStorageManagers[factory.PeerAccountTrie] = storageManager

	arg := ArgsGenesisBlockCreator{
		GenesisTime:              0,
		StartEpochNum:            0,
		PubkeyConv:               mock.NewPubkeyConverterMock(32),
		Blkc:                     &mock.BlockChainStub{},
		Marshalizer:              &mock.MarshalizerMock{},
		Hasher:                   &mock.HasherMock{},
		Uint64ByteSliceConverter: &mock.Uint64ByteSliceConverterMock{},
		DataPool:                 mock.NewPoolsHolderMock(),
		TxLogsProcessor:          &mock.TxLogProcessorMock{},
		VirtualMachineConfig:     config.VirtualMachineConfig{},
		HardForkConfig:           config.HardforkConfig{},
		SystemSCConfig: config.SystemSmartContractsConfig{
			ESDTSystemSCConfig: config.ESDTSystemSCConfig{
				BaseIssuingCost: "5000000000000000000000",
				OwnerAddress:    "erd1932eft30w753xyvme8d49qejgkjc09n5e49w4mwdjtm0neld797su0dlxp",
			},
		},
		TrieStorageManagers: trieStorageManagers,
		BlockSignKeyGen:     &mock.KeyGenMock{},
	}

	arg.ShardCoordinator = &mock.ShardCoordinatorMock{
		NumOfShards: 2,
		SelfShardId: 0,
	}

	var err error
	arg.Accounts, err = createAccountAdapter(
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		factoryState.NewAccountCreator(),
		trieStorageManagers[factory.UserAccountTrie],
	)
	require.Nil(t, err)

	arg.ValidatorAccounts = &mock.AccountsStub{
		RootHashCalled: func() ([]byte, error) {
			return make([]byte, 0), nil
		},
		CommitCalled: func() ([]byte, error) {
			return make([]byte, 0), nil
		},
		SaveAccountCalled: func(account state.AccountHandler) error {
			return nil
		},
		LoadAccountCalled: func(address []byte) (state.AccountHandler, error) {
			return state.NewEmptyPeerAccount(), nil
		},
	}

	arg.GasMap = arwenConfig.MakeGasMapForTests()
	defaults.FillGasMapInternal(arg.GasMap, 1)

	ted := &economics.TestEconomicsData{
		EconomicsData: &economics.EconomicsData{},
	}
	ted.SetGenesisNodePrice(nodePrice)
	ted.SetMinStep(big.NewInt(1))
	ted.SetTotalSupply(entireSupply)
	ted.SetUnJailPrice(big.NewInt(1))
	ted.SetMaxGasLimitPerBlock(math.MaxUint64)
	arg.Economics = ted.EconomicsData

	arg.Store = &mock.ChainStorerMock{
		GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
			return mock.NewStorerMock()
		},
	}

	arg.AccountsParser, err = parsing.NewAccountsParser(
		genesisFilename,
		arg.Economics.TotalSupply(),
		arg.PubkeyConv,
		&mock.KeyGeneratorStub{},
	)
	require.Nil(t, err)

	arg.SmartContractParser, err = parsing.NewSmartContractsParser(
		"testdata/smartcontracts.json",
		arg.PubkeyConv,
		&mock.KeyGeneratorStub{},
	)
	require.Nil(t, err)

	arg.InitialNodesSetup = initialNodes

	return arg
}

func TestGenesisBlockCreator_CreateGenesisBlocksJustDelegationShouldWork(t *testing.T) {
	scAddressBytes, _ := hex.DecodeString("00000000000000000500761b8c4a25d3979359223208b412285f635e71300102")
	stakedAddr, _ := hex.DecodeString("b00102030405060708090001020304050607080900010203040506070809000b")
	initialNodesSetup := &mock.InitialNodesHandlerStub{
		InitialNodesInfoCalled: func() (map[uint32][]sharding.GenesisNodeInfoHandler, map[uint32][]sharding.GenesisNodeInfoHandler) {
			return map[uint32][]sharding.GenesisNodeInfoHandler{
				0: {
					&mock.GenesisNodeInfoHandlerMock{
						AddressBytesValue: scAddressBytes,
						PubKeyBytesValue:  bytes.Repeat([]byte{1}, 96),
					},
					&mock.GenesisNodeInfoHandlerMock{
						AddressBytesValue: stakedAddr,
						PubKeyBytesValue:  bytes.Repeat([]byte{2}, 96),
					},
				},
				1: {
					&mock.GenesisNodeInfoHandlerMock{
						AddressBytesValue: scAddressBytes,
						PubKeyBytesValue:  bytes.Repeat([]byte{3}, 96),
					},
				},
			}, make(map[uint32][]sharding.GenesisNodeInfoHandler)
		},
		MinNumberOfNodesCalled: func() uint32 {
			return 1
		},
	}
	arg := createMockArgument(
		t,
		"testdata/genesisTest1.json",
		initialNodesSetup,
		big.NewInt(22000),
	)

	gbc, err := NewGenesisBlockCreator(arg)
	require.Nil(t, err)

	blocks, err := gbc.CreateGenesisBlocks()

	assert.Nil(t, err)
	assert.Equal(t, 3, len(blocks))
}

func TestGenesisBlockCreator_CreateGenesisBlocksStakingAndDelegationShouldWork(t *testing.T) {
	scAddressBytes, _ := hex.DecodeString("00000000000000000500761b8c4a25d3979359223208b412285f635e71300102")
	stakedAddr, _ := hex.DecodeString("b00102030405060708090001020304050607080900010203040506070809000b")
	stakedAddr2, _ := hex.DecodeString("d00102030405060708090001020304050607080900010203040506070809000d")
	initialNodesSetup := &mock.InitialNodesHandlerStub{
		InitialNodesInfoCalled: func() (map[uint32][]sharding.GenesisNodeInfoHandler, map[uint32][]sharding.GenesisNodeInfoHandler) {
			return map[uint32][]sharding.GenesisNodeInfoHandler{
				0: {
					&mock.GenesisNodeInfoHandlerMock{
						AddressBytesValue: scAddressBytes,
						PubKeyBytesValue:  bytes.Repeat([]byte{1}, 96),
					},
					&mock.GenesisNodeInfoHandlerMock{
						AddressBytesValue: stakedAddr,
						PubKeyBytesValue:  bytes.Repeat([]byte{2}, 96),
					},
					&mock.GenesisNodeInfoHandlerMock{
						AddressBytesValue: scAddressBytes,
						PubKeyBytesValue:  bytes.Repeat([]byte{3}, 96),
					},
					&mock.GenesisNodeInfoHandlerMock{
						AddressBytesValue: stakedAddr2,
						PubKeyBytesValue:  bytes.Repeat([]byte{8}, 96),
					},
				},
				1: {
					&mock.GenesisNodeInfoHandlerMock{
						AddressBytesValue: scAddressBytes,
						PubKeyBytesValue:  bytes.Repeat([]byte{4}, 96),
					},
					&mock.GenesisNodeInfoHandlerMock{
						AddressBytesValue: scAddressBytes,
						PubKeyBytesValue:  bytes.Repeat([]byte{5}, 96),
					},
					&mock.GenesisNodeInfoHandlerMock{
						AddressBytesValue: stakedAddr2,
						PubKeyBytesValue:  bytes.Repeat([]byte{6}, 96),
					},
					&mock.GenesisNodeInfoHandlerMock{
						AddressBytesValue: stakedAddr2,
						PubKeyBytesValue:  bytes.Repeat([]byte{7}, 96),
					},
				},
			}, make(map[uint32][]sharding.GenesisNodeInfoHandler)
		},
		MinNumberOfNodesCalled: func() uint32 {
			return 1
		},
	}
	arg := createMockArgument(
		t,
		"testdata/genesisTest2.json",
		initialNodesSetup,
		big.NewInt(47000),
	)
	gbc, err := NewGenesisBlockCreator(arg)
	require.Nil(t, err)

	blocks, err := gbc.CreateGenesisBlocks()

	assert.Nil(t, err)
	assert.Equal(t, 3, len(blocks))
}
