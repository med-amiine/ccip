// Code generated by mockery v2.43.2. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// USDCReader is an autogenerated mock type for the USDCReader type
type USDCReader struct {
	mock.Mock
}

// GetUSDCMessagePriorToLogIndexInTx provides a mock function with given fields: ctx, logIndex, usdcTokenIndexOffset, txHash
func (_m *USDCReader) GetUSDCMessagePriorToLogIndexInTx(ctx context.Context, logIndex int64, usdcTokenIndexOffset int, txHash string) ([]byte, error) {
	ret := _m.Called(ctx, logIndex, usdcTokenIndexOffset, txHash)

	if len(ret) == 0 {
		panic("no return value specified for GetUSDCMessagePriorToLogIndexInTx")
	}

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, int64, int, string) ([]byte, error)); ok {
		return rf(ctx, logIndex, usdcTokenIndexOffset, txHash)
	}
	if rf, ok := ret.Get(0).(func(context.Context, int64, int, string) []byte); ok {
		r0 = rf(ctx, logIndex, usdcTokenIndexOffset, txHash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, int64, int, string) error); ok {
		r1 = rf(ctx, logIndex, usdcTokenIndexOffset, txHash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewUSDCReader creates a new instance of USDCReader. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewUSDCReader(t interface {
	mock.TestingT
	Cleanup(func())
}) *USDCReader {
	mock := &USDCReader{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
