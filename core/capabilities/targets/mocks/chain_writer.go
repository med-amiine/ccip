// Code generated by mockery v2.43.2. DO NOT EDIT.

package mocks

import (
	context "context"
	big "math/big"

	mock "github.com/stretchr/testify/mock"

	types "github.com/smartcontractkit/chainlink-common/pkg/types"
)

// ChainWriter is an autogenerated mock type for the ChainWriter type
type ChainWriter struct {
	mock.Mock
}

// GetFeeComponents provides a mock function with given fields: ctx
func (_m *ChainWriter) GetFeeComponents(ctx context.Context) (*types.ChainFeeComponents, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for GetFeeComponents")
	}

	var r0 *types.ChainFeeComponents
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (*types.ChainFeeComponents, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) *types.ChainFeeComponents); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ChainFeeComponents)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetTransactionStatus provides a mock function with given fields: ctx, transactionID
func (_m *ChainWriter) GetTransactionStatus(ctx context.Context, transactionID string) (types.TransactionStatus, error) {
	ret := _m.Called(ctx, transactionID)

	if len(ret) == 0 {
		panic("no return value specified for GetTransactionStatus")
	}

	var r0 types.TransactionStatus
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (types.TransactionStatus, error)); ok {
		return rf(ctx, transactionID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) types.TransactionStatus); ok {
		r0 = rf(ctx, transactionID)
	} else {
		r0 = ret.Get(0).(types.TransactionStatus)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, transactionID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SubmitTransaction provides a mock function with given fields: ctx, contractName, method, args, transactionID, toAddress, meta, value
func (_m *ChainWriter) SubmitTransaction(ctx context.Context, contractName string, method string, args interface{}, transactionID string, toAddress string, meta *types.TxMeta, value *big.Int) error {
	ret := _m.Called(ctx, contractName, method, args, transactionID, toAddress, meta, value)

	if len(ret) == 0 {
		panic("no return value specified for SubmitTransaction")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, interface{}, string, string, *types.TxMeta, *big.Int) error); ok {
		r0 = rf(ctx, contractName, method, args, transactionID, toAddress, meta, value)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewChainWriter creates a new instance of ChainWriter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewChainWriter(t interface {
	mock.TestingT
	Cleanup(func())
}) *ChainWriter {
	mock := &ChainWriter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
