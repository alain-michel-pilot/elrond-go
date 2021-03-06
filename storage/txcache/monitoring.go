package txcache

import (
	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/core"
)

var log = logger.GetOrCreate("txcache")

func (cache *TxCache) monitorEvictionWrtSenderLimit(sender []byte, evicted [][]byte) {
	log.Trace("TxCache.AddTx() evict transactions wrt. limit by sender", "name", cache.name, "sender", sender, "num", len(evicted))

	for i := 0; i < core.MinInt(len(evicted), numEvictedTxsToDisplay); i++ {
		log.Trace("TxCache.AddTx() evict transactions wrt. limit by sender", "name", cache.name, "sender", sender, "tx", evicted[i])
	}
}

func (cache *TxCache) monitorEvictionStart() *core.StopWatch {
	log.Debug("TxCache: eviction started", "name", cache.name, "numBytes", cache.NumBytes(), "txs", cache.CountTx(), "senders", cache.CountSenders())
	cache.displaySendersHistogram()
	sw := core.NewStopWatch()
	sw.Start("eviction")
	return sw
}

func (cache *TxCache) monitorEvictionEnd(stopWatch *core.StopWatch) {
	stopWatch.Stop("eviction")
	duration := stopWatch.GetMeasurement("eviction")
	log.Debug("TxCache: eviction ended", "name", cache.name, "duration", duration, "numBytes", cache.NumBytes(), "txs", cache.CountTx(), "senders", cache.CountSenders())
	cache.evictionJournal.display()
	cache.displaySendersHistogram()
}

func (cache *TxCache) monitorSelectionStart() *core.StopWatch {
	log.Debug("TxCache: selection started", "name", cache.name, "numBytes", cache.NumBytes(), "txs", cache.CountTx(), "senders", cache.CountSenders())
	cache.displaySendersHistogram()
	sw := core.NewStopWatch()
	sw.Start("selection")
	return sw
}

func (cache *TxCache) monitorSelectionEnd(selection []*WrappedTransaction, stopWatch *core.StopWatch) {
	stopWatch.Stop("selection")
	duration := stopWatch.GetMeasurement("selection")
	numSendersSelected := cache.numSendersSelected.Reset()
	numSendersWithInitialGap := cache.numSendersWithInitialGap.Reset()
	numSendersWithMiddleGap := cache.numSendersWithMiddleGap.Reset()
	numSendersInGracePeriod := cache.numSendersInGracePeriod.Reset()

	log.Debug("TxCache: selection ended", "name", cache.name, "duration", duration,
		"numTxSelected", len(selection),
		"numSendersSelected", numSendersSelected,
		"numSendersWithInitialGap", numSendersWithInitialGap,
		"numSendersWithMiddleGap", numSendersWithMiddleGap,
		"numSendersInGracePeriod", numSendersInGracePeriod,
	)

	cache.displaySendersHistogram()
}

type batchSelectionJournal struct {
	isFirstBatch  bool
	copied        int
	hasInitialGap bool
	hasMiddleGap  bool
	isGracePeriod bool
}

func (cache *TxCache) monitorBatchSelectionEnd(journal batchSelectionJournal) {
	if !journal.isFirstBatch {
		return
	}

	if journal.hasInitialGap {
		cache.numSendersWithInitialGap.Increment()
	} else if journal.hasMiddleGap {
		// Currently, we only count middle gaps on first batch (for simplicity)
		cache.numSendersWithMiddleGap.Increment()
	}

	if journal.isGracePeriod {
		cache.numSendersInGracePeriod.Increment()
	} else if journal.copied > 0 {
		cache.numSendersSelected.Increment()
	}
}

func (cache *TxCache) monitorSweepingStart() *core.StopWatch {
	sw := core.NewStopWatch()
	sw.Start("sweeping")
	return sw
}

func (cache *TxCache) monitorSweepingEnd(numTxs uint32, numSenders uint32, stopWatch *core.StopWatch) {
	stopWatch.Stop("sweeping")
	duration := stopWatch.GetMeasurement("sweeping")
	log.Debug("TxCache: swept senders:", "name", cache.name, "duration", duration, "txs", numTxs, "senders", numSenders)
	cache.displaySendersHistogram()
}

func (cache *TxCache) displaySendersHistogram() {
	txListBySenderMap := cache.txListBySender.backingMap
	log.Debug("TxCache.sendersHistogram:", "chunks", txListBySenderMap.ChunksCounts(), "scoreChunks", txListBySenderMap.ScoreChunksCounts())
}

// evictionJournal keeps a short journal about the eviction process
// This is useful for debugging and reasoning about the eviction
type evictionJournal struct {
	evictionPerformed bool
	passOneNumTxs     uint32
	passOneNumSenders uint32
	passOneNumSteps   uint32
}

func (journal *evictionJournal) display() {
	log.Debug("Eviction.pass1:", "txs", journal.passOneNumTxs, "senders", journal.passOneNumSenders, "steps", journal.passOneNumSteps)
}

func (cache *TxCache) diagnose() {
	sw := core.NewStopWatch()
	sw.Start("diagnose")

	sizeInBytes := cache.NumBytes()
	numTxsEstimate := int(cache.CountTx())
	numTxsInChunks := cache.txByHash.backingMap.Count()
	txsKeys := cache.txByHash.backingMap.Keys()
	numSendersEstimate := uint32(cache.CountSenders())
	numSendersInChunks := cache.txListBySender.backingMap.Count()
	numSendersInScoreChunks := cache.txListBySender.backingMap.CountSorted()
	sendersKeys := cache.txListBySender.backingMap.Keys()
	sendersKeysSorted := cache.txListBySender.backingMap.KeysSorted()
	sendersSnapshot := cache.txListBySender.getSnapshotAscending()

	sw.Stop("diagnose")
	duration := sw.GetMeasurement("diagnose")

	fine := (numSendersEstimate == numSendersInChunks && numSendersEstimate == numSendersInScoreChunks)
	fine = fine && (len(sendersKeys) == len(sendersKeysSorted) && len(sendersKeys) == len(sendersSnapshot))
	fine = fine && (int(numSendersEstimate) == len(sendersKeys))
	fine = fine && (numTxsEstimate == numTxsInChunks && numTxsEstimate == len(txsKeys))

	logFunc := log.Trace
	if !fine {
		logFunc = log.Debug
	}

	logFunc("Diagnose", "name", cache.name, "duration", duration, "fine", fine)
	logFunc("Size:", "current", sizeInBytes, "max", cache.config.NumBytesThreshold)
	logFunc("NumSenders:", "estimate", numSendersEstimate, "inChunks", numSendersInChunks, "inScoreChunks", numSendersInScoreChunks)
	logFunc("NumSenders (continued):", "keys", len(sendersKeys), "keysSorted", len(sendersKeysSorted), "snapshot", len(sendersSnapshot))
	logFunc("NumTxs:", "estimate", numTxsEstimate, "inChunks", numTxsInChunks, "keys", len(txsKeys))
}
