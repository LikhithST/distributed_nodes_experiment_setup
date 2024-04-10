package runner

import (
	"context"
	"sync"
	// "math/rand"
	"time"
	"fmt"
	// "reflect"
	// "encoding/json"
	"github.com/google/uuid"
	"github.com/docker/docker/api/types"
	// "github.com/docker/docker/client"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
)

// StatsHandler is for gRPC stats
type statsHandler struct {
	results chan *callResult

	id     uuid.UUID
	hasLog bool
	log    Logger

	lock   sync.RWMutex
	ignore bool
}

// HandleConn handle the connection
func (c *statsHandler) HandleConn(ctx context.Context, cs stats.ConnStats) {

// ----------------------prints cs------------------------------

	// fmt.Println("---------------->>HandleConn>>>")
	// rpcStatsType := reflect.TypeOf(cs)
	// if rpcStatsType.Kind() == reflect.Ptr {
	// 	rpcStatsType = rpcStatsType.Elem()
	// }
	// for i := 0; i < rpcStatsType.NumField(); i++ {
	// 	field := rpcStatsType.Field(i)
	// 	fmt.Println(field.Name)
	// }
// ----------------------prints cs------------------------------

	// no-op
}

// TagConn exists to satisfy gRPC stats.Handler.
func (c *statsHandler) TagConn(ctx context.Context, cti *stats.ConnTagInfo) context.Context {
	// no-op
// ----------------------prints cti------------------------------

	// fmt.Println("---------------->>TagConn>>>")

	// rpcStatsType := reflect.TypeOf(cti)
	// if rpcStatsType.Kind() == reflect.Ptr {
	// 	rpcStatsType = rpcStatsType.Elem()
	// }
	// for i := 0; i < rpcStatsType.NumField(); i++ {
	// 	field := rpcStatsType.Field(i)
	// 	fmt.Println(field.Name)
	// }
// ----------------------prints cti------------------------------

	return ctx
}

// Helper function to calculate CPU usage percentage
func calculateCPUPercentage(stat *types.StatsJSON) float64 {
	cpuDelta := float64(stat.CPUStats.CPUUsage.TotalUsage) - float64(stat.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stat.CPUStats.SystemUsage) - float64(stat.PreCPUStats.SystemUsage)
	cpuPercent := (cpuDelta / systemDelta) * float64(len(stat.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	return cpuPercent
}

// Helper function to calculate memory usage percentage
func calculateMemoryPercentage(stat *types.StatsJSON) float64 {
	memUsage := float64(stat.MemoryStats.Usage)
	memLimit := float64(stat.MemoryStats.Limit)
	memPercent := (memUsage / memLimit) * 100.0
	return memPercent
}


// HandleRPC implements per-RPC tracing and stats instrumentation.
func (c *statsHandler) HandleRPC(ctx context.Context, rs stats.RPCStats) {
	switch rs := rs.(type) {

	case *stats.InHeader:
		 // You can access the `InHeader` field from the `s` object to get the received headers.
		 if rs.Client {
			fmt.Printf("Received headers for method------------------<<<<<<<<<<<<<<<inheader>>>>>>>>>>>>>>>>><%+v ",rs.Header)
		// 	ign := false
		// c.lock.RLock()
		// ign = c.ignore
		// c.lock.RUnlock()
		// if !ign {
			
			c.results <- &callResult{id:c.id, databroker_timestamp: time.Now()}
		// }
		}

	case *stats.End:
// ----------------------prints rs------------------------------
		// fmt.Printf("%+v\n", rs)
		// rpcStatsType := reflect.TypeOf(rs)
		// if rpcStatsType.Kind() == reflect.Ptr {
		// 	rpcStatsType = rpcStatsType.Elem()
		// }
		// for i := 0; i < rpcStatsType.NumField(); i++ {
		// 	field := rpcStatsType.Field(i)
		// 	fmt.Println(field.Name)
		// }
		// fmt.Printf("---------------->>end>>>>>>>%+v", rs.EndTime)
		// fmt.Printf("---------------->>>>>>>>>%+v", rs.BeginTime)
		// fmt.Printf("---------------->>>>>>>>>%#v", rs.Trailer)
		// fmt.Println("---------------->>>>>>>>>%d",c.id)

		fmt.Println("Processing")
		

// ----------------------prints rs------------------------------

		// cli, err := client.NewClientWithOpts(client.FromEnv)
		// if err != nil {
		// 	panic(err)
		// }

		// Specify the Docker ID for which you want to access the stats
		// containerID := "efa6bbff8645"

		// Get the Docker stats for the specified container
		// stats, err := cli.ContainerStats(context.Background(), containerID, false)
		// if err != nil {
		// 	panic(err)
		// }

		// defer stats.Body.Close()

		// var stat types.StatsJSON
		// if err := json.NewDecoder(stats.Body).Decode(&stat); err != nil {
		// 	panic(err)
		// }

		// cpu_utilisation := calculateCPUPercentage(&stat)
		// mem_utilisation := calculateMemoryPercentage(&stat)
		// fmt.Printf("CPU Usage: %.2f%%\n", cpu_utilisation)
		// fmt.Printf("Memory Usage: %.2f%%\n", mem_utilisation)
		// err = cli.Close()
		// if err != nil {
		// 	panic(err)
		// }

		

		ign := false
		c.lock.RLock()
		ign = c.ignore
		c.lock.RUnlock()

		if !ign {

			duration := rs.EndTime.Sub(rs.BeginTime)

			var st string
			s, ok := status.FromError(rs.Error)
			if ok {
				st = s.Code().String()
			}

			// c.lock.RLock()

				for elem := range c.results {
					fmt.Println(elem.id)
					if elem.id == c.id {
						fmt.Println("yes")
						fmt.Println(elem.id)
						elem.err = rs.Error
						elem.status = st
						elem.duration = duration
						elem.timestamp = rs.EndTime
						elem.cpu_utilisation = 110
						elem.mem_utilisation = 10
					}
					c.results <- elem

				}

			// 	c.lock.RUnlock()


			// c.results <- &callResult{id:c.id, databroker_timestamp: time.Now()}
			if c.hasLog {
				c.log.Debugw("Received RPC Stats",
					"statsID", c.id, "code", st, "error", rs.Error,
					"duration", duration, "stats", rs)
			}
		}
		// else{
		// 	fmt.Println("Ignored")
		// }
	}

}

func (c *statsHandler) Ignore(val bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.ignore = val
}

// TagRPC implements per-RPC context management.
func (c *statsHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	return ctx
}
