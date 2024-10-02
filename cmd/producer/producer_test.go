package main

import (
	"context"
	"testing"

	"example.com/task_flow/config"
	"example.com/task_flow/internal"
	"example.com/task_flow/mocks"

	"github.com/golang/mock/gomock"
)

func TestGenerateTasks_StreamTaskAlwaysCalled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := config.LoadConfig()
	ctx := context.Background()
	bs := mocks.NewMockByteStream_Server(ctrl)

	mockQueries := internal.MockQueries{
		GetMaxTaskIDFunc: func(ctx context.Context) (interface{}, error) {
			return int32(10), nil
		},
		CreateTaskFunc: func(ctx context.Context, params internal.CreateTaskParams) (interface{}, error) {
			return nil, nil
		},
	}

	Generate_Tasks(cfg, ctx, mockQueries, bs)

}
