package mock

import (
	"github.com/ElrondNetwork/elrond-go/data"
)

type UnsignedTxHandlerMock struct {
	CleanProcessedUtxsCalled func()
	AddProcessedUTxCalled    func(tx data.TransactionHandler)
	CreateAllUTxsCalled      func() []data.TransactionHandler
	VerifyCreatedUTxsCalled  func() error
	AddTxFeeFromBlockCalled  func(tx data.TransactionHandler)
}

func (ut *UnsignedTxHandlerMock) AddTxFeeFromBlock(tx data.TransactionHandler) {
	if ut.AddTxFeeFromBlockCalled == nil {
		return
	}

	ut.AddTxFeeFromBlockCalled(tx)
}

func (ut *UnsignedTxHandlerMock) CleanProcessedUTxs() {
	if ut.CleanProcessedUtxsCalled == nil {
		return
	}

	ut.CleanProcessedUtxsCalled()
}

func (ut *UnsignedTxHandlerMock) AddProcessedUTx(tx data.TransactionHandler) {
	if ut.AddProcessedUTxCalled == nil {
		return
	}

	ut.AddProcessedUTxCalled(tx)
}

func (ut *UnsignedTxHandlerMock) CreateAllUTxs() []data.TransactionHandler {
	if ut.CreateAllUTxsCalled == nil {
		return nil
	}
	return ut.CreateAllUTxsCalled()
}

func (ut *UnsignedTxHandlerMock) VerifyCreatedUTxs() error {
	if ut.VerifyCreatedUTxsCalled == nil {
		return nil
	}
	return ut.VerifyCreatedUTxsCalled()
}