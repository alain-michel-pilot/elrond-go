package block

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"io/ioutil"
	"math/big"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/core/logger"
	"github.com/ElrondNetwork/elrond-go/core/statistics"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/integrationTests"
	"github.com/pkg/profile"
	"github.com/stretchr/testify/assert"
)

var agarioFile = "agarioV2.hex"
var stepDelay = time.Second

func TestShouldProcessWithScTxsJoinAndRewardOneRound(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	log := logger.DefaultLogger()
	log.SetLevel(logger.LogDebug)

	scCode, err := ioutil.ReadFile(agarioFile)
	assert.Nil(t, err)

	maxShards := uint32(1)
	numOfNodes := 4
	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()
	advertiserAddr := integrationTests.GetConnectableAddress(advertiser)

	nodes := make([]*integrationTests.TestProcessorNode, numOfNodes)
	for i := 0; i < numOfNodes; i++ {
		nodes[i] = integrationTests.NewTestProcessorNode(maxShards, 0, 0, advertiserAddr)
	}

	idxProposer := 0
	nrPlayers := 1
	players := make([]*integrationTests.TestWalletAccount, nrPlayers)
	for i := 0; i < nrPlayers; i++ {
		players[i] = integrationTests.CreateTestWalletAccount(nodes[idxProposer].ShardCoordinator, 0)
	}

	defer func() {
		_ = advertiser.Close()
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	for _, n := range nodes {
		_ = n.Messenger.Bootstrap()
	}

	fmt.Println("Delaying for nodes p2p bootstrap...")
	time.Sleep(stepDelay)

	round := uint64(0)
	round = incrementAndPrintRound(round)

	hardCodedSk, _ := hex.DecodeString("5561d28b0d89fa425bbbf9e49a018b5d1e4a462c03d2efce60faf9ddece2af06")
	hardCodedScResultingAddress, _ := hex.DecodeString("000000000000000000005fed9c659422cd8429ce92f8973bba2a9fb51e0eb3a1")
	nodes[idxProposer].LoadTxSignSkBytes(hardCodedSk)

	initialVal := big.NewInt(10000000)
	topUpValue := big.NewInt(500)
	integrationTests.MintAllNodes(nodes, initialVal)
	integrationTests.MintAllPlayers(nodes, players, initialVal)

	deployScTx(nodes, idxProposer, string(scCode))
	proposeBlock(nodes, idxProposer, round)
	time.Sleep(time.Second)
	syncBlock(t, nodes, idxProposer, round)
	round = incrementAndPrintRound(round)

	runMultipleRoundsOfTheGame(
		t,
		1,
		nrPlayers,
		idxProposer,
		nodes,
		players,
		topUpValue,
		hardCodedScResultingAddress,
		round,
	)

	checkRootHashes(t, nodes, []int{0})

	time.Sleep(1 * time.Second)
}

func TestProcessesJoinGameTheSamePlayerMultipleTimesRewardAndEndgameInMultipleRounds(t *testing.T) {
	t.Skip("this is a stress test for VM and AGAR.IO")

	stepDelay = time.Nanosecond

	p := profile.Start(profile.MemProfile, profile.ProfilePath("."), profile.NoShutdownHook)
	defer p.Stop()

	log := logger.DefaultLogger()
	log.SetLevel(logger.LogDebug)

	scCode, err := ioutil.ReadFile(agarioFile)
	assert.Nil(t, err)

	maxShards := uint32(1)
	numOfNodes := 4
	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()
	advertiserAddr := integrationTests.GetConnectableAddress(advertiser)

	nodes := make([]*integrationTests.TestProcessorNode, numOfNodes)
	for i := 0; i < numOfNodes; i++ {
		nodes[i] = integrationTests.NewTestProcessorNode(maxShards, 0, 0, advertiserAddr)
	}

	idxProposer := 0
	nrPlayers := 100
	players := make([]*integrationTests.TestWalletAccount, nrPlayers)
	players[0] = integrationTests.CreateTestWalletAccount(nodes[idxProposer].ShardCoordinator, 0)
	for i := 1; i < nrPlayers; i++ {
		players[i] = players[0]
	}
	nrPlayers = 1
	defer func() {
		_ = advertiser.Close()
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	for _, n := range nodes {
		_ = n.Messenger.Bootstrap()
	}

	fmt.Println("Delaying for nodes p2p bootstrap...")
	time.Sleep(time.Second)

	round := uint64(0)
	round = incrementAndPrintRound(round)

	hardCodedSk, _ := hex.DecodeString("5561d28b0d89fa425bbbf9e49a018b5d1e4a462c03d2efce60faf9ddece2af06")
	hardCodedScResultingAddress, _ := hex.DecodeString("000000000000000000005fed9c659422cd8429ce92f8973bba2a9fb51e0eb3a1")
	nodes[idxProposer].LoadTxSignSkBytes(hardCodedSk)

	initialVal := big.NewInt(10000000)
	topUpValue := big.NewInt(500)
	integrationTests.MintAllNodes(nodes, initialVal)
	integrationTests.MintAllPlayers(nodes, players, initialVal)

	deployScTx(nodes, idxProposer, string(scCode))
	proposeBlock(nodes, idxProposer, round)
	time.Sleep(time.Second)
	syncBlock(t, nodes, idxProposer, round)
	round = incrementAndPrintRound(round)

	runMultipleRoundsOfTheGame(
		t,
		10,
		nrPlayers,
		idxProposer,
		nodes,
		players,
		topUpValue,
		hardCodedScResultingAddress,
		round,
	)

	checkRootHashes(t, nodes, []int{0})

	time.Sleep(time.Second)
}

func TestProcessesJoinGame100PlayersMultipleTimesRewardAndEndgameInMultipleRounds(t *testing.T) {
	t.Skip("this is a stress test for VM and AGAR.IO")

	stepDelay = time.Nanosecond

	p := profile.Start(profile.MemProfile, profile.ProfilePath("."), profile.NoShutdownHook)
	defer p.Stop()

	log := logger.DefaultLogger()
	log.SetLevel(logger.LogDebug)

	scCode, err := ioutil.ReadFile(agarioFile)
	assert.Nil(t, err)

	maxShards := uint32(1)
	numOfNodes := 4
	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()
	advertiserAddr := integrationTests.GetConnectableAddress(advertiser)

	nodes := make([]*integrationTests.TestProcessorNode, numOfNodes)
	for i := 0; i < numOfNodes; i++ {
		nodes[i] = integrationTests.NewTestProcessorNode(maxShards, 0, 0, advertiserAddr)
	}

	idxProposer := 0
	nrPlayers := 100
	players := make([]*integrationTests.TestWalletAccount, nrPlayers)
	for i := 1; i < nrPlayers; i++ {
		players[i] = integrationTests.CreateTestWalletAccount(nodes[idxProposer].ShardCoordinator, 0)
	}

	defer func() {
		_ = advertiser.Close()
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	for _, n := range nodes {
		_ = n.Messenger.Bootstrap()
	}

	fmt.Println("Delaying for nodes p2p bootstrap...")
	time.Sleep(time.Second)

	round := uint64(0)
	round = incrementAndPrintRound(round)

	hardCodedSk, _ := hex.DecodeString("5561d28b0d89fa425bbbf9e49a018b5d1e4a462c03d2efce60faf9ddece2af06")
	hardCodedScResultingAddress, _ := hex.DecodeString("000000000000000000005fed9c659422cd8429ce92f8973bba2a9fb51e0eb3a1")
	nodes[idxProposer].LoadTxSignSkBytes(hardCodedSk)

	initialVal := big.NewInt(10000000)
	topUpValue := big.NewInt(500)
	integrationTests.MintAllNodes(nodes, initialVal)
	integrationTests.MintAllPlayers(nodes, players, initialVal)

	deployScTx(nodes, idxProposer, string(scCode))
	proposeBlock(nodes, idxProposer, round)
	time.Sleep(time.Second)
	syncBlock(t, nodes, idxProposer, round)
	round = incrementAndPrintRound(round)

	runMultipleRoundsOfTheGame(
		t,
		100,
		nrPlayers,
		idxProposer,
		nodes,
		players,
		topUpValue,
		hardCodedScResultingAddress,
		round,
	)

	checkRootHashes(t, nodes, []int{0})

	time.Sleep(time.Second)
}

func getPercentageOfValue(value *big.Int, percentage float64) *big.Int {
	x := new(big.Float).SetInt(value)
	y := big.NewFloat(percentage)

	z := new(big.Float).Mul(x, y)

	op := big.NewInt(0)
	result, _ := z.Int(op)

	return result
}

func runMultipleRoundsOfTheGame(
	t *testing.T,
	nrRounds, nrPlayers, idxProposer int,
	nodes []*integrationTests.TestProcessorNode,
	players []*integrationTests.TestWalletAccount,
	topUpValue *big.Int,
	hardCodedScResultingAddress []byte,
	round uint64,
) {
	rMonitor := &statistics.ResourceMonitor{}
	nrRewardedPlayers := 10
	if nrRewardedPlayers > nrPlayers {
		nrRewardedPlayers = nrPlayers
	}

	totalWithdrawValue := big.NewInt(0).SetUint64(topUpValue.Uint64() * uint64(len(players)))
	withDrawValues := make([]*big.Int, nrRewardedPlayers)
	winnerRate := 1.0 - 0.05*float64(nrRewardedPlayers-1)
	withDrawValues[0] = big.NewInt(0).Set(getPercentageOfValue(totalWithdrawValue, winnerRate))
	for i := 1; i < nrRewardedPlayers; i++ {
		withDrawValues[i] = big.NewInt(0).Set(getPercentageOfValue(totalWithdrawValue, 0.05))
	}

	for rr := 0; rr < nrRounds; rr++ {
		for _, player := range players {
			playerJoinsGame(
				nodes[idxProposer],
				player.Address.Bytes(),
				player.SingleSigner,
				player.SkTxSign,
				topUpValue,
				rr,
				hardCodedScResultingAddress,
			)
			newBalance := big.NewInt(0)
			newBalance = newBalance.Sub(player.Balance, topUpValue)
			player.Balance = player.Balance.Set(newBalance)
		}
		time.Sleep(time.Second)

		startTime := time.Now()
		proposeBlock(nodes, idxProposer, round)
		elapsedTime := time.Since(startTime)
		fmt.Printf("Block Created in %s\n", elapsedTime)

		time.Sleep(time.Second)
		syncBlock(t, nodes, idxProposer, round)
		round = incrementAndPrintRound(round)

		checkJoinGame(t, nodes, players, topUpValue, idxProposer, hardCodedScResultingAddress)

		for i := 0; i < nrRewardedPlayers; i++ {
			nodeCallsRewardAndSend(nodes, idxProposer, players[i].Address.Bytes(), withDrawValues[i], rr, hardCodedScResultingAddress)
			newBalance := big.NewInt(0)
			newBalance = newBalance.Add(players[i].Balance, withDrawValues[i])
			players[i].Balance = players[i].Balance.Set(newBalance)
		}
		//TODO activate endgame when it is corrected
		//nodeEndGame(nodes, idxProposer, rr, hardCodedScResultingAddress)
		time.Sleep(time.Second)

		startTime = time.Now()
		proposeBlock(nodes, idxProposer, round)
		elapsedTime = time.Since(startTime)
		fmt.Printf("Block Created in %s\n", elapsedTime)

		time.Sleep(time.Second)
		syncBlock(t, nodes, idxProposer, round)
		round = incrementAndPrintRound(round)

		checkRewardsDistribution(t, nodes, players, topUpValue, totalWithdrawValue,
			hardCodedScResultingAddress, idxProposer)

		fmt.Println(rMonitor.GenerateStatistics())
	}
}

func checkJoinGame(
	t *testing.T,
	nodes []*integrationTests.TestProcessorNode,
	players []*integrationTests.TestWalletAccount,
	topUpValue *big.Int,
	idxProposer int,
	hardCodedScResultingAddress []byte,
) {
	for _, player := range players {
		checkJoinGameIsDoneCorrectlyPlayerSide(
			t,
			nodes,
			idxProposer,
			player,
		)
	}

	nrPlayers := len(players)
	allTopUpValue := big.NewInt(0).SetUint64(topUpValue.Uint64() * uint64(nrPlayers))
	checkJoinGameIsDoneCorrectlySCSide(
		t,
		nodes,
		idxProposer,
		allTopUpValue,
		hardCodedScResultingAddress,
	)
}

func checkRewardsDistribution(
	t *testing.T,
	nodes []*integrationTests.TestProcessorNode,
	players []*integrationTests.TestWalletAccount,
	topUpValue *big.Int,
	withdrawValue *big.Int,
	hardCodedScResultingAddress []byte,
	idxProposer int,
) {
	for _, player := range players {
		checkRewardIsDoneCorrectlyPlayerSide(
			t,
			nodes,
			idxProposer,
			player,
		)
	}

	nrPlayers := len(players)
	allTopUpValue := big.NewInt(0).SetUint64(topUpValue.Uint64() * uint64(nrPlayers))
	checkRewardIsDoneCorrectlySCSide(
		t,
		nodes,
		idxProposer,
		allTopUpValue,
		withdrawValue,
		hardCodedScResultingAddress,
	)
}

func incrementAndPrintRound(round uint64) uint64 {
	round++
	fmt.Printf("#################################### ROUND %d BEGINS ####################################\n\n", round)

	return round
}

func deployScTx(nodes []*integrationTests.TestProcessorNode, senderIdx int, scCode string) {
	fmt.Println("Deploying SC...")
	txDeploy := createTxDeploy(nodes[senderIdx], scCode)
	nodes[senderIdx].SendTransaction(txDeploy)
	fmt.Println("Delaying for disseminating the deploy tx...")
	time.Sleep(stepDelay)

	fmt.Println(integrationTests.MakeDisplayTable(nodes))
}

func proposeBlock(nodes []*integrationTests.TestProcessorNode, idxProposer int, round uint64) {
	fmt.Println("Proposing block...")
	for idx, n := range nodes {
		if idx != idxProposer {
			continue
		}

		body, header, _ := n.ProposeBlock(round)
		n.BroadcastBlock(body, header)
		n.CommitBlock(body, header)
	}

	fmt.Println("Delaying for disseminating headers and miniblocks...")
	time.Sleep(stepDelay)
	fmt.Println(integrationTests.MakeDisplayTable(nodes))
}

func syncBlock(t *testing.T, nodes []*integrationTests.TestProcessorNode, idxProposer int, round uint64) {
	fmt.Println("All other shard nodes sync the proposed block...")
	for idx, n := range nodes {
		if idx == idxProposer {
			continue
		}

		err := n.SyncNode(uint64(round))
		if err != nil {
			assert.Fail(t, err.Error())
			return
		}
	}

	time.Sleep(stepDelay)
	fmt.Println(integrationTests.MakeDisplayTable(nodes))
}

func playerJoinsGame(
	txDispatcherNode *integrationTests.TestProcessorNode,
	sndAddress []byte,
	signer crypto.SingleSigner,
	privKey crypto.PrivateKey,
	joinGameVal *big.Int,
	round int,
	scAddress []byte,
) {

	fmt.Println("Calling SC.joinGame...")
	txScCall := createTxJoinGame(sndAddress, signer, privKey, joinGameVal, round, scAddress)
	txDispatcherNode.SendTransaction(txScCall)
	fmt.Println("Delaying for disseminating SC call tx...")
	time.Sleep(stepDelay)
}

func nodeEndGame(
	nodes []*integrationTests.TestProcessorNode,
	idxNode int,
	round int,
	scAddress []byte,
) {

	fmt.Println("Calling SC.endGame...")
	txScCall := createTxEndGame(nodes[idxNode], round, scAddress)
	nodes[idxNode].SendTransaction(txScCall)
	time.Sleep(stepDelay)

	fmt.Println(integrationTests.MakeDisplayTable(nodes))
}

func nodeCallsRewardAndSend(
	nodes []*integrationTests.TestProcessorNode,
	idxNodeOwner int,
	winnerAddress []byte,
	prize *big.Int,
	round int,
	scAddress []byte,
) {

	fmt.Println("Calling SC.rewardAndSendToWallet...")
	txScCall := createTxRewardAndSendToWallet(nodes[idxNodeOwner], winnerAddress, prize, round, scAddress)
	nodes[idxNodeOwner].SendTransaction(txScCall)
	fmt.Println("Delaying for disseminating SC call tx...")
	time.Sleep(stepDelay)
}

func createTxDeploy(tn *integrationTests.TestProcessorNode, scCode string) *transaction.Transaction {
	tx := &transaction.Transaction{
		Nonce:    0,
		Value:    big.NewInt(0),
		RcvAddr:  make([]byte, 32),
		SndAddr:  tn.PkTxSignBytes,
		Data:     scCode,
		GasPrice: 0,
		GasLimit: 1000000000000,
	}
	txBuff, _ := integrationTests.TestMarshalizer.Marshal(tx)
	tx.Signature, _ = tn.SingleSigner.Sign(tn.SkTxSign, txBuff)

	return tx
}

func createTxEndGame(tn *integrationTests.TestProcessorNode, round int, scAddress []byte) *transaction.Transaction {
	tx := &transaction.Transaction{
		Nonce:    0,
		RcvAddr:  scAddress,
		SndAddr:  tn.PkTxSignBytes,
		Data:     fmt.Sprintf("endGame@%d", round),
		GasPrice: 0,
		GasLimit: 1000000000000,
	}
	txBuff, _ := integrationTests.TestMarshalizer.Marshal(tx)
	tx.Signature, _ = tn.SingleSigner.Sign(tn.SkTxSign, txBuff)

	fmt.Printf("End %s\n", hex.EncodeToString(tn.PkTxSignBytes))

	return tx
}

func createTxJoinGame(sndAddress []byte, signer crypto.SingleSigner, privKey crypto.PrivateKey, joinGameVal *big.Int, round int, scAddress []byte) *transaction.Transaction {
	tx := &transaction.Transaction{
		Nonce:    0,
		Value:    joinGameVal,
		RcvAddr:  scAddress,
		SndAddr:  sndAddress,
		Data:     fmt.Sprintf("joinGame@%d", round),
		GasPrice: 0,
		GasLimit: 1000000000000,
	}
	txBuff, _ := integrationTests.TestMarshalizer.Marshal(tx)
	tx.Signature, _ = signer.Sign(privKey, txBuff)

	fmt.Printf("Join %s\n", hex.EncodeToString(sndAddress))

	return tx
}

func createTxRewardAndSendToWallet(tnOwner *integrationTests.TestProcessorNode, winnerAddress []byte, prizeVal *big.Int, round int, scAddress []byte) *transaction.Transaction {
	tx := &transaction.Transaction{
		Nonce:    0,
		Value:    big.NewInt(0),
		RcvAddr:  scAddress,
		SndAddr:  tnOwner.PkTxSignBytes,
		Data:     fmt.Sprintf("rewardAndSendToWallet@%d@%s@%X", round, hex.EncodeToString(winnerAddress), prizeVal),
		GasPrice: 0,
		GasLimit: 1000000000000,
	}
	txBuff, _ := integrationTests.TestMarshalizer.Marshal(tx)
	tx.Signature, _ = tnOwner.SingleSigner.Sign(tnOwner.SkTxSign, txBuff)

	fmt.Printf("Reward %s\n", hex.EncodeToString(winnerAddress))

	return tx
}

func checkJoinGameIsDoneCorrectlyPlayerSide(
	t *testing.T,
	nodes []*integrationTests.TestProcessorNode,
	idxNodeCallerExists int,
	player *integrationTests.TestWalletAccount,
) {
	nodeWithCaller := nodes[idxNodeCallerExists]

	fmt.Println("Checking sender has initial-topUp val...")
	fmt.Printf("Checking %s\n", hex.EncodeToString(nodeWithCaller.PkTxSignBytes))
	accnt, _ := nodeWithCaller.AccntState.GetExistingAccount(integrationTests.CreateAddresFromAddrBytes(player.Address.Bytes()))
	assert.NotNil(t, accnt)
	ok := assert.Equal(t, player.Balance.Uint64(), accnt.(*state.Account).Balance.Uint64())
	if !ok {
		fmt.Printf("Expected player balance %d Actual player balance %d\n", player.Balance.Uint64(), accnt.(*state.Account).Balance.Uint64())
	}
}

func checkJoinGameIsDoneCorrectlySCSide(
	t *testing.T,
	nodes []*integrationTests.TestProcessorNode,
	idxNodeScExists int,
	topUpVal *big.Int,
	scAddressBytes []byte,
) {
	nodeWithSc := nodes[idxNodeScExists]

	fmt.Println("Checking SC account received topUp val...")
	accnt, _ := nodeWithSc.AccntState.GetExistingAccount(integrationTests.CreateAddresFromAddrBytes(scAddressBytes))
	assert.NotNil(t, accnt)
	ok := assert.Equal(t, topUpVal.Uint64(), accnt.(*state.Account).Balance.Uint64())
	if !ok {
		fmt.Printf("Expected topUp val %d Actual topU val %d\n", topUpVal.Uint64(), accnt.(*state.Account).Balance.Uint64())
	}
}

func checkRewardIsDoneCorrectlyPlayerSide(
	t *testing.T,
	nodes []*integrationTests.TestProcessorNode,
	idxNodeCallerExists int,
	player *integrationTests.TestWalletAccount,
) {
	nodeWithCaller := nodes[idxNodeCallerExists]

	fmt.Println("Checking sender has initial-topUp+withdraw val...")
	fmt.Printf("Checking %s\n", hex.EncodeToString(nodeWithCaller.PkTxSignBytes))
	accnt, _ := nodeWithCaller.AccntState.GetExistingAccount(integrationTests.CreateAddresFromAddrBytes(player.Address.Bytes()))
	assert.NotNil(t, accnt)
	ok := assert.Equal(t, player.Balance.Uint64(), accnt.(*state.Account).Balance.Uint64())
	if !ok {
		fmt.Printf("Expected sender val %d Actual sender val %d\n", player.Balance.Uint64(), accnt.(*state.Account).Balance.Uint64())
	}
}

func checkRewardIsDoneCorrectlySCSide(
	t *testing.T,
	nodes []*integrationTests.TestProcessorNode,
	idxNodeScExists int,
	topUpVal *big.Int,
	withdraw *big.Int,
	scAddressBytes []byte,
) {

	nodeWithSc := nodes[idxNodeScExists]

	fmt.Println("Checking SC account has topUp-withdraw val...")
	accnt, _ := nodeWithSc.AccntState.GetExistingAccount(integrationTests.CreateAddresFromAddrBytes(scAddressBytes))
	assert.NotNil(t, accnt)
	expectedSC := big.NewInt(0).Set(topUpVal)
	expectedSC.Sub(expectedSC, withdraw)
	ok := assert.Equal(t, expectedSC.Uint64(), accnt.(*state.Account).Balance.Uint64())
	if !ok {
		fmt.Printf("Expected smart contract val %d Actual smart contract val %d\n", expectedSC.Uint64(), accnt.(*state.Account).Balance.Uint64())
	}
}

func checkRootHashes(t *testing.T, nodes []*integrationTests.TestProcessorNode, idxProposers []int) {
	for _, idx := range idxProposers {
		checkRootHashInShard(t, nodes, idx)
	}
}

func checkRootHashInShard(t *testing.T, nodes []*integrationTests.TestProcessorNode, idxProposer int) {
	proposerNode := nodes[idxProposer]
	proposerRootHash, _ := proposerNode.AccntState.RootHash()

	for i := 0; i < len(nodes); i++ {
		node := nodes[i]

		if node.ShardCoordinator.SelfId() != proposerNode.ShardCoordinator.SelfId() {
			continue
		}

		fmt.Printf("Testing roothash for node index %d, shard ID %d...\n", i, node.ShardCoordinator.SelfId())
		nodeRootHash, _ := node.AccntState.RootHash()
		assert.Equal(t, proposerRootHash, nodeRootHash)
	}
}
