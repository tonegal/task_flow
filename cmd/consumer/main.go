package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"example.com/task_flow/config"
	"example.com/task_flow/internal"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/flowcontrol"
	"capnproto.org/go/capnp/v3/rpc"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type ConsumerImpl struct {
	queries *internal.Queries
}

func (ci ConsumerImpl) StreamTask(ctx context.Context, call internal.ByteStream_streamTask) error {
	task_obj, err := call.Args().Task()
	if err != nil {
		panic(err)
	}

	tid := task_obj.Tid()
	ttype := task_obj.Ttype()
	tvalue := task_obj.Tvalue()

	now_utc := time.Now().UTC()
	_, err = ci.queries.UpdateTaskState(ctx, internal.UpdateTaskStateParams{
		ID:             int32(tid),
		State:          "processing",
		LastUpdateTime: pgtype.Timestamp{Time: now_utc, Valid: true},
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Task started to be processed: ID: %d, Type: %d, Value: %d\n", tid, ttype, tvalue)
	time.Sleep(time.Duration(tvalue) * time.Millisecond)

	now_utc = time.Now().UTC()
	_, err = ci.queries.UpdateTaskState(ctx, internal.UpdateTaskStateParams{
		ID:             int32(tid),
		State:          "done",
		LastUpdateTime: pgtype.Timestamp{Time: now_utc, Valid: true},
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Task %d processed\n", tid)
	return nil
}

func (ConsumerImpl) Done(ctx context.Context, call internal.ByteStream_done) error {
	fmt.Println("RPC call for Done method landed")
	return nil
}

func Init_Consumer() error {
	cfg := config.LoadConfig()

	ctx := context.Background()

	pg_conn, err := pgx.Connect(ctx, "host=localhost user=postgres password=postgres dbname=task_flow sslmode=disable")
	if err != nil {
		panic(err)
	}
	defer pg_conn.Close(ctx)

	queries := internal.New(pg_conn)

	fmt.Println("Postgres connection created...")

	server := ConsumerImpl{queries: queries}
	client := internal.ByteStream_ServerToClient(server)
	client.SetFlowLimiter(flowcontrol.NewFixedLimiter(cfg.Consumer.Flow_Size_Limit)) // 1 Message is ~8 Byte

	listener, err := net.Listen("tcp", "localhost:12345")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	for {
		fmt.Println("Consumer listener ready...")
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Failed to accept connection: %v\n", err)
			continue
		}
		defer conn.Close()

		// Set up Cap'n Proto RPC
		rpcConn := rpc.NewConn(rpc.NewStreamTransport(conn), &rpc.Options{
			BootstrapClient: capnp.Client(client),
		})

		defer rpcConn.Close()

		select {
		case <-rpcConn.Done():
			return nil
		case <-ctx.Done():
			return rpcConn.Close()
		}
	}
}

func main() {
	Init_Consumer()
}
