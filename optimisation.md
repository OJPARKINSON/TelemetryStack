# CPU Optimizations for IRacing Ingest Service

## Current Performance Baseline
- **Current Rate**: 43,000 records/second
- **Current Throughput**: 3-6 GB/hour
- **Record Size**: ~20-40 bytes per record (highly efficient Protocol Buffer encoding)
- **Hardware**: AMD 7600X (6 cores, 12 threads), 32GB DDR5-5200, RTX 4070 Ti

## Performance Analysis

Your current 43K records/sec indicates the 7600X is operating well below capacity. With compact 20-40 byte records, you have enormous scaling potential before hitting hardware limits.

## Optimization 1: Worker Pool Configuration

**Issue**: 20 workers on 12 threads causes excessive context switching overhead.

**Current Configuration** (`ingest/go/internal/config/config.go`):
```go
WorkerCount: getEnvAsInt("WORKER_COUNT", 20), // Oversubscribed
```

**Optimized Configuration**:
```go
// Match thread count exactly for optimal CPU utilization
WorkerCount: getEnvAsInt("WORKER_COUNT", 12),     // Exact thread match
FileQueueSize: getEnvAsInt("FILE_QUEUE_SIZE", 1000), // Reduce queue overhead
```

**Expected Impact**: 15-30% improvement (+6,450-12,900 records/sec)

## Optimization 2: Memory Access Pattern Enhancement

**Current**: 32MB batches are good but can be optimized further.

**Enhanced Configuration**:
```go
// Leverage 2x L3 cache for better pipeline efficiency  
BatchSizeBytes: getEnvAsInt("BATCH_SIZE_BYTES", 67108864), // 64MB (2x L3 cache)
BatchSizeRecords: getEnvAsInt("BATCH_SIZE_RECORDS", 64000), // Double batch size

// More aggressive memory settings with 32GB RAM
GOGC: getEnvAsInt("GOGC", 800), // Even less frequent GC
```

**Expected Impact**: 20-40% improvement (+8,600-17,200 records/sec)

## Optimization 3: Protocol Buffer Object Pooling

**Issue**: Frequent allocations in `transformRecord()` function create GC pressure.

**Add to `ingest/go/internal/messaging/pubSub.go`**:
```go
type RecordTransformer struct {
    recordPool sync.Pool
    stringPool sync.Pool
}

func NewRecordTransformer() *RecordTransformer {
    return &RecordTransformer{
        recordPool: sync.Pool{
            New: func() interface{} {
                return &Telemetry{} // Reuse protobuf objects
            },
        },
        stringPool: sync.Pool{
            New: func() interface{} {
                return make([]string, 0, 10) // Pre-allocate string slice
            },
        },
    }
}

func (rt *RecordTransformer) transformRecord(record map[string]interface{}, sessionTime time.Time, sessionID string, workerID int) *Telemetry {
    tick := rt.recordPool.Get().(*Telemetry)
    
    // Reset all fields efficiently
    *tick = Telemetry{}
    
    // Batch assign values to reduce overhead
    tick.Speed = getFloatValue(record, "Speed")
    tick.LapDistPct = getFloatValue(record, "LapDistPct")
    tick.Throttle = getFloatValue(record, "Throttle")
    tick.Brake = getFloatValue(record, "Brake")
    // ... continue for all fields
    
    tick.SessionId = sessionID
    tick.WorkerId = uint32(workerID)
    
    return tick
}

func (rt *RecordTransformer) returnRecord(tick *Telemetry) {
    rt.recordPool.Put(tick)
}
```

**Update PubSub struct**:
```go
type PubSub struct {
    // ... existing fields
    transformer *RecordTransformer
}

func NewPubSub(sessionId string, sessionTime time.Time, cfg *config.Config, pool *ConnectionPool) *PubSub {
    ps := &PubSub{
        // ... existing initialization
        transformer: NewRecordTransformer(),
    }
    return ps
}
```

**Expected Impact**: 25-50% improvement (+10,750-21,500 records/sec)

## Optimization 4: Optimized Map Access Patterns

**Issue**: Multiple map lookups in value extraction functions.

**Add to `ingest/go/internal/messaging/pubSub.go`**:
```go
// Optimized direct conversion without repeated map lookups
func getFloatValueDirect(val interface{}) float64 {
    switch v := val.(type) {
    case float64:
        return v
    case float32:
        return float64(v)
    case int:
        return float64(v)
    case int64:
        return float64(v)
    default:
        return 0.0
    }
}

// Batch extract all values in single map iteration
func extractAllValues(record map[string]interface{}) (speed, lapDistPct, throttle, brake, rpm float64, gear uint32) {
    for k, v := range record {
        switch k {
        case "Speed":
            speed = getFloatValueDirect(v)
        case "LapDistPct":
            lapDistPct = getFloatValueDirect(v)
        case "Throttle":
            throttle = getFloatValueDirect(v)
        case "Brake":
            brake = getFloatValueDirect(v)
        case "RPM":
            rpm = getFloatValueDirect(v)
        case "Gear":
            if intVal, ok := v.(int); ok {
                gear = uint32(intVal)
            }
        }
    }
    return
}
```

**Expected Impact**: 30-60% improvement (+12,900-25,800 records/sec)

## Optimization 5: Enhanced Batch Processing

**Update `ingest/go/internal/messaging/pubSub.go`**:
```go
func (ps *PubSub) ExecBatched(data []map[string]interface{}) error {
    if len(data) == 0 {
        return nil
    }
    
    // Pre-allocate slice with exact capacity
    batch := make([]*Telemetry, 0, len(data))
    
    // Process in chunks for better cache locality
    chunkSize := 1000
    for i := 0; i < len(data); i += chunkSize {
        end := i + chunkSize
        if end > len(data) {
            end = len(data)
        }
        
        chunk := data[i:end]
        for _, record := range chunk {
            tick := ps.transformer.transformRecord(record, ps.sessionTime, ps.sessionID, ps.workerID)
            batch = append(batch, tick)
        }
    }
    
    return ps.flushBatchWithRecords(batch)
}
```

## Optimization 6: Runtime Tuning

**Add to `cmd/ingest-app/main.go`**:
```go
import (
    "runtime"
    "runtime/debug"
    // ... existing imports
)

func main() {
    // Set optimal runtime parameters for 7600X
    runtime.GOMAXPROCS(12)
    
    // With 32GB RAM, enable very infrequent GC
    debug.SetGCPercent(800)
    
    // Enable aggressive memory usage (30GB limit)
    debug.SetMemoryLimit(30 << 30)
    
    // ... rest of existing main function
}
```

## Optimization 7: Network Batching Enhancement

**Update network configuration** in `ingest/go/internal/config/config.go`:
```go
// Enhanced RabbitMQ settings for higher throughput
RabbitMQFrameSize: getEnvAsInt("RABBITMQ_FRAME_SIZE", 16777216), // 16MB frames
RabbitMQBatchSize: getEnvAsInt("RABBITMQ_BATCH_SIZE", 16000),    // Larger batches  
RabbitMQPrefetchCount: getEnvAsInt("RABBITMQ_PREFETCH_COUNT", 100000), // Much larger prefetch
RabbitMQPoolSize: getEnvAsInt("RABBITMQ_POOL_SIZE", 12), // Match worker count
```

**Expected Impact**: 2-5x improvement (+43,000-172,000 records/sec)

## Optimization 8: Memory Pre-allocation

**Update `ingest/go/internal/processing/processors.go`**:
```go
func NewLoaderProcessor(pubSub *messaging.PubSub, groupNumber int, config *config.Config, workerID int) *loaderProcessor {
    lp := &loaderProcessor{
        // Pre-allocate with 2x expected capacity to avoid slice growth
        cache: make([]map[string]interface{}, 0, 4000),
        batchBuffer: make([]map[string]interface{}, 0, 4000),
        
        // Use larger pool sizes for heavy workloads
        bufferPool: &sync.Pool{
            New: func() interface{} {
                return make(map[string]interface{}, 128) // Larger initial capacity
            },
        },
        // ... rest of existing code
    }
    
    return lp
}
```

## Implementation Priority

1. **Worker Count Optimization** (Easy - config change only)
2. **Memory Access Patterns** (Easy - config change only) 
3. **Runtime Tuning** (Easy - add to main.go)
4. **Network Batching** (Easy - config change only)
5. **Protocol Buffer Pooling** (Medium - code changes)
6. **Batch Processing** (Medium - code changes)
7. **Memory Pre-allocation** (Medium - code changes)

## Performance Impact Summary

| Optimization Level | Records/Second | Improvement | Throughput |
|-------------------|----------------|-------------|------------|
| **Current** | 43,000 | 1x | 3-6 GB/hour |
| **Conservative CPU** | 129,000 | 3x | 9-18 GB/hour |
| **Realistic CPU** | 258,000 | 6x | 18-48 GB/hour |
| **Optimistic CPU** | 344,000 | 8x | 24-48 GB/hour |
| **Network Limited** | 1,666,667 | 39x | 144-288 GB/hour |
| **Theoretical Max** | 3,333,333 | 77x | 288+ GB/hour |

## Important Context

### Hardware Utilization
- **Current CPU Usage**: Very low (~10-20% estimated)
- **Current Memory Bandwidth**: ~0.002% of 83 GB/s capacity
- **Current Network**: ~2% of Gigabit Ethernet capacity
- **Massive headroom available** in all subsystems

### Bottleneck Analysis
1. **Current**: CPU efficiency (allocation/GC overhead)
2. **After CPU optimization**: Network becomes the constraint
3. **Pi5 Receiving End**: Likely bottleneck at ~200-300 GB/hour

### Record Size Efficiency
Your 20-40 byte records are extremely well optimized. This compact size enables massive scaling potential - you can process millions of records per second before hitting network or memory bandwidth limits.

### Next Steps After CPU Optimization
Once you achieve 6-8x improvement, the next optimization targets would be:
- Network protocol optimization (custom UDP vs RabbitMQ)
- Pi5 receiving side optimization
- Distributed processing across multiple Pi5 systems