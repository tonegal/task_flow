package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"example.com/task_flow/config"
	"example.com/task_flow/internal"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/flowcontrol"
	"capnproto.org/go/capnp/v3/rpc"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metric variables
var (
	tasksProcessing = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "tasks_processing_total",
		Help: "The total number of tasks being processed by Consumer service",
	})

	tasksDone = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "tasks_done_total",
		Help: "The total number of tasks completed by Consumer service in one execution",
	})

	tasksSumValuesByType = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tasks_sum_values_by_type",
			Help: "Sum of values of tasks completed by Consumer service in one execution, grouped by task type",
		},
		[]string{"type"}, // Label for task type
	)

	tasksDoneByType = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tasks_done_by_type",
			Help: "The total number of tasks completed by Consumer service in one execution, grouped by task type",
		},
		[]string{"type"}, // Label for task type
	)
)

// Recevier handler of RPC stream
type ConsumerImpl struct {
	queries *internal.Queries
}

func (ci ConsumerImpl) StreamTask(ctx context.Context, call internal.ByteStream_streamTask) error {
	task_obj, err := call.Args().Task()
	if err != nil {
		log.Panic(err)
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
		log.Panic(err)
	}

	tasksProcessing.Inc()

	log.Printf("Task processing started: ID: %d, Type: %d, Value: %d\n", tid, ttype, tvalue)

	// Simulate task with a simple sleep call
	time.Sleep(time.Duration(tvalue) * time.Millisecond)

	now_utc = time.Now().UTC()
	_, err = ci.queries.UpdateTaskState(ctx, internal.UpdateTaskStateParams{
		ID:             int32(tid),
		State:          "done",
		LastUpdateTime: pgtype.Timestamp{Time: now_utc, Valid: true},
	})
	if err != nil {
		log.Panic(err)
	}

	tasksProcessing.Dec()
	ttype_string := fmt.Sprintf("%d", ttype)
	tasksSumValuesByType.WithLabelValues(ttype_string).Add(float64(tvalue))
	tasksDone.Inc()
	tasksDoneByType.WithLabelValues(ttype_string).Inc()

	log.Printf("Task %d processed\n", tid)
	return nil
}

func (ConsumerImpl) Done(ctx context.Context, call internal.ByteStream_done) error {
	log.Println("RPC Done call landed")
	return nil
}

func CleanUpDb(ctx context.Context, quer *internal.Queries) {

	now_utc := time.Now().UTC()
	updated_recs, err := quer.ReplaceAllTaskState(ctx, internal.ReplaceAllTaskStateParams{
		State:          "processing",
		State_2:        "interrupt",
		LastUpdateTime: pgtype.Timestamp{Time: now_utc, Valid: true},
	})
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Database states cleaned: %d records are updated from 'processing' to 'interrupt'\n", len(updated_recs))
}

func Receive_Tasks(cfg *config.Config, ctx context.Context, client internal.ByteStream) error {

	// Set up listener
	address := fmt.Sprintf("%s:%s", cfg.Consumer.Host, cfg.Consumer.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Panic(err)
	}
	defer listener.Close()

	for {
		// Get incoming connection
		log.Println("Consumer listener ready...")
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v\n", err)
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

func Init_Consumer() {
	cfg := config.LoadConfig()

	db_conn_str := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Dbname,
		cfg.Database.Sslmode,
	)

	ctx := context.Background()

	pg_conn, err := pgx.Connect(ctx, db_conn_str)
	if err != nil {
		log.Panic(err)
	}
	defer pg_conn.Close(ctx)

	queries := internal.New(pg_conn)

	log.Println("Postgres connection created...")

	internal.StartMetricsServer(cfg.Prometheus.Cons_Metrics_Port)
	tasksProcessing.Set(0)
	tasksDone.Set(0)
	tasksSumValuesByType.Reset()
	tasksDoneByType.Reset()

	log.Println("Metrics populator is started...")

	// If there are remaining 'processing' states, update their state to 'interrupt'
	CleanUpDb(ctx, queries)

	server := ConsumerImpl{queries: queries}
	client := internal.ByteStream_ServerToClient(server)
	client.SetFlowLimiter(flowcontrol.NewFixedLimiter(cfg.Consumer.Flow_Size_Limit)) // 1 Message is ~8 Byte

	Receive_Tasks(cfg, ctx, client)

}

func main() {
	Init_Consumer()
}
