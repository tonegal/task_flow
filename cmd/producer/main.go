package main

import (
	"context"
	"fmt"
	"math/rand"
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

var (
	tasksReceived = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "tasks_received_total",
		Help: "The total number of tasks received from Producer service in one execution",
	})
)

func Generate_Tasks(cfg *config.Config, ctx context.Context, quer *internal.Queries, bs internal.ByteStream) {

	taskID, err := quer.GetMaxTaskID(ctx)
	if err != nil {
		panic(err)
	}

	var maxID int32
	if taskID == nil {
		fmt.Println("No tasks found. ID is defaulting to 0.")
		maxID = 0
	} else {
		maxID = taskID.(int32)
		fmt.Printf("Maximal ID value among stored Task records: %d\n", maxID)
		maxID++
	}

	// Calculate message producer loop's sleep time based on rate limits
	sleep_time_ms := 1000/min(cfg.Producer.Prod_Rate, cfg.Consumer.Rate_Limit) + 1 // +1 is needed because division rounds down

	for i := 0; i < cfg.Producer.Max_Backlog; i++ {
		time.Sleep(time.Duration(sleep_time_ms) * time.Millisecond)

		arena := capnp.SingleSegment(nil)
		_, seg, err := capnp.NewMessage(arena)
		if err != nil {
			panic(err)
		}

		taskObj, err := internal.NewTaskMessage(seg)
		if err != nil {
			panic(err)
		}

		//rand.Seed(time.Now().UnixNano())
		tid := maxID + int32(i)
		ttype := rand.Intn(10)   // Random integer between 0 and 9
		tvalue := rand.Intn(100) // Random integer between 0 and 99
		taskObj.SetTid(uint32(tid))
		taskObj.SetTtype(uint8(ttype))
		taskObj.SetTvalue(uint8(tvalue))

		now_utc := time.Now().UTC()
		_, err = quer.CreateTask(ctx, internal.CreateTaskParams{
			ID:             tid,
			Type:           int16(taskObj.Ttype()),
			Value:          int16(taskObj.Tvalue()),
			State:          "received",
			CreationTime:   pgtype.Timestamp{Time: now_utc, Valid: true},
			LastUpdateTime: pgtype.Timestamp{Time: now_utc, Valid: true},
		})
		if err != nil {
			panic(err)
		}

		tasksReceived.Inc()

		//fmt.Printf("Task inserted to DB: ID: %d, Type: %d, Value: %d\n", inTask.ID, inTask.Type, inTask.Value)

		err = bs.StreamTask(ctx, func(p internal.ByteStream_streamTask_Params) error {
			return p.SetTask(taskObj)
		})
		if err != nil {
			panic(err)
		}
		//fmt.Printf("Sent Task: ID: %d, Type: %d Value: %d\n", taskObj.Tid(), taskObj.Ttype(), taskObj.Tvalue())
	}
}

func Init_Producer() {

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
		panic(err)
	}
	defer pg_conn.Close(ctx)

	queries := internal.New(pg_conn)

	fmt.Println("Postgres connection created...")

	internal.StartMetricsServer(cfg.Prometheus.Prod_Metrics_Port)
	tasksReceived.Set(0)

	fmt.Println("Metrics populator is started...")

	address := fmt.Sprintf("%s:%s", cfg.Consumer.Host, cfg.Consumer.Port)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	fmt.Println("Producer connected to consumer...")

	rpc_conn := rpc.NewConn(rpc.NewStreamTransport(conn), nil)
	defer rpc_conn.Close()
	bs := internal.ByteStream(rpc_conn.Bootstrap(ctx))
	bs.SetFlowLimiter(flowcontrol.NewFixedLimiter(cfg.Producer.Flow_Size_Limit)) // 1 Message is ~8 Byte

	fmt.Println("ByteStream RPC connection created...")

	Generate_Tasks(cfg, ctx, queries, bs)

	future, release := bs.Done(ctx, nil)
	defer release()
	_, err = future.Struct()
	if err != nil {
		panic(err)
	}

	if err := bs.WaitStreaming(); err != nil {
		panic(err)
	}
	fmt.Println("Tasks sent successfully")
}

func main() {
	Init_Producer()
}
