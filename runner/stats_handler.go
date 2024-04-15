package runner

import (
	"context"
	"sync"
	"strconv"
	// "log"
	// "math/rand"
	"time"
	// "fmt"
	"google.golang.org/grpc/metadata"
	// "reflect"
	// "encoding/json"
	// "github.com/google/uuid"
	"github.com/docker/docker/api/types"
	// "github.com/docker/docker/client"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
)

// StatsHandler is for gRPC stats
type statsHandler struct {
	results chan *callResult
	id     int
	hasLog bool
	log    Logger

	lock   sync.RWMutex
	ignore bool
}

type MutableObject struct {
    InMetadata metadata.MD // Example mutable field
}
// HandleConn handle the connection
func (c *statsHandler) HandleConn(ctx context.Context, cs stats.ConnStats) {

	// no-op
}

// TagConn exists to satisfy gRPC stats.Handler.
func (c *statsHandler) TagConn(ctx context.Context, cti *stats.ConnTagInfo) context.Context {
	// no-op

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
		var headerValue metadata.MD 
		 // You can access the `InHeader` field from the `s` object to get the received headers.
		 if rs.Client {
		ign := false
		c.lock.RLock()
		ign = c.ignore
		c.lock.RUnlock()
		if !ign {
			headerValue = rs.Header

			if header, ok := ctx.Value("InHeader").(*MutableObject); ok {
                header.InMetadata = headerValue
            }
		}
		}

	case *stats.End:

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

			// Retrieve the header value from the context
			
			// fmt.Printf("------------->>>>>>>>>%T",ctx.Value("InHeader"))
			// fmt.Println("------------->>>>>>>>>",ctx.Value("InHeader"))
			var ts time.Time
			if header, ok := ctx.Value("InHeader").(*MutableObject); ok {
                // fmt.Println(header.InMetadata["ts"][0])
				databroker_timestamp, err := strconv.ParseInt(header.InMetadata["ts"][0], 10, 64)
				if err == nil {
					ts = time.Unix(databroker_timestamp,0)
				}
            }
			c.results <- &callResult{rs.Error, st, duration, rs.EndTime, ts, 10, 10}

			if c.hasLog {
				c.log.Debugw("Received RPC Stats",
					"statsID", c.id, "code", st, "error", rs.Error,
					"duration", duration, "stats", rs)
			}
		}
	}

}

func (c *statsHandler) Ignore(val bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.ignore = val
}

// TagRPC implements per-RPC context management.
func (c *statsHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	
	ctx = context.WithValue(ctx, "InHeader", &MutableObject{})
	

	return ctx
}
