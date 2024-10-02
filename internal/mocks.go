package internal

import (
	"context"
)

type MockQueries struct {
	GetMaxTaskIDFunc func(ctx context.Context) (interface{}, error)
	CreateTaskFunc   func(ctx context.Context, params CreateTaskParams) (interface{}, error)
}

func (m *MockQueries) GetMaxTaskID(ctx context.Context) (interface{}, error) {
	if m.GetMaxTaskIDFunc != nil {
		return m.GetMaxTaskIDFunc(ctx)
	}
	return nil, nil
}

func (m *MockQueries) CreateTask(ctx context.Context, params CreateTaskParams) (interface{}, error) {
	if m.CreateTaskFunc != nil {
		return m.CreateTaskFunc(ctx, params)
	}
	return nil, nil
}
