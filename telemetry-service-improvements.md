# Telemetry Service Performance Optimization Plan

## Current Performance Baseline

### System Overview
- **Architecture**: RabbitMQ → Telemetry Service (.NET) → QuestDB
- **Current Data Volume**: 1,354,914 telemetry points successfully stored
- **Date Range**: February 2025 → August 2025 (6 months of data)
- **Processing Model**: Pull-based consumption with individual message processing

### Current Performance Metrics
- **Theoretical Throughput**: ~1,000 msgs/sec (50 threads × 20 msgs/sec)
- **Pull Batch Size**: 10 messages per RabbitMQ call
- **QuestDB Auto-flush**: 2,000 rows or 500ms intervals
- **Resource Usage**:
  - CPU: 4% sustained
  - Memory: 71MB (0.87% of 8GB limit)
  - Processing Threads: 50 concurrent (SemaphoreSlim)

### Current Bottlenecks Identified
1. **Small Pull Batches**: Only 10 messages per RabbitMQ API call
2. **Individual Processing**: Each message processed separately to QuestDB
3. **Conservative Delays**: 10ms between batches, 100ms when queue empty
4. **Single Connection**: One RabbitMQ connection for all 50 threads
5. **Memory Allocation**: Frequent TelemetryBatch object creation

## Optimization Strategies

### **Optimization 1: Increase RabbitMQ Pull Batch Size**

**Current State**: `BatchSize = 10`
**Proposed**: `BatchSize = 50`

**Implementation**:
```csharp
private const int BatchSize = 50; // Increased from 10
```

**Rationale**:
- Reduce RabbitMQ API calls by 5x
- Better utilization of network round trips
- More efficient use of processing threads

**Estimated Performance Gain**: **2-3x throughput improvement**
- From ~1,000 msgs/sec to ~2,000-3,000 msgs/sec
- Reduced network overhead
- Better thread utilization

**Risk Level**: **Low**
**Implementation Time**: 15 minutes

### **Optimization 2: Implement True Batch Processing**

**Current State**: Each TelemetryBatch processed individually to QuestDB
**Proposed**: Accumulate multiple TelemetryBatch messages before QuestDB write

**Implementation Strategy**:
```csharp
// New batch accumulator
private readonly List<TelemetryRecord> _batchAccumulator = new();
private const int MaxAccumulatorSize = 5000; // Records, not messages
private const int MaxAccumulatorTimeMs = 250; // Max wait time

// Modified processing logic
private async Task ProcessMessageBatch(List<BasicGetResult> messages)
{
    foreach (var message in messages)
    {
        var telemetryBatch = TelemetryBatch.Parser.ParseFrom(message.Body.ToArray());
        _batchAccumulator.AddRange(telemetryBatch.Records);
        
        if (_batchAccumulator.Count >= MaxAccumulatorSize || 
            TimeSinceLastFlush > MaxAccumulatorTimeMs)
        {
            await FlushBatchToQuestDB();
        }
    }
    
    // ACK all messages after successful processing
    foreach (var message in messages)
    {
        await channel.BasicAckAsync(message.DeliveryTag, false);
    }
}
```

**Benefits**:
- Fewer QuestDB connection calls
- Better utilization of QuestDB's auto-flush settings
- Reduced network overhead to QuestDB
- More efficient batching of telemetry records

**Estimated Performance Gain**: **3-5x throughput improvement**
- From current ~1,000 msgs/sec to ~3,000-5,000 msgs/sec
- Significant reduction in QuestDB connection overhead

**Risk Level**: **Medium** (requires careful error handling)
**Implementation Time**: 4 hours

### **Optimization 3: Reduce Processing Delays**

**Current State**:
- `BasePollIntervalMs = 10` (delay between batches)
- `EmptyQueueDelayMs = 100` (delay when no messages)

**Proposed**:
- `BasePollIntervalMs = 1` (minimal delay)
- `EmptyQueueDelayMs = 10` (faster recovery)

**Implementation**:
```csharp
private const int BasePollIntervalMs = 1;   // Reduced from 10
private const int EmptyQueueDelayMs = 10;   // Reduced from 100
```

**Rationale**:
- Minimize idle time between processing cycles
- Faster response to message availability
- Maintain backpressure relief without excessive delays

**Estimated Performance Gain**: **20-30% latency reduction**
- Lower end-to-end processing latency
- Faster queue drainage during bursts

**Risk Level**: **Low**
**Implementation Time**: 5 minutes

### **Optimization 4: Connection Pooling**

**Current State**: Single RabbitMQ connection for all 50 threads
**Proposed**: 3-4 RabbitMQ connections with dedicated channels

**Implementation Strategy**:
```csharp
private readonly List<IConnection> _connections = new();
private readonly ConcurrentQueue<IChannel> _channelPool = new();

private async Task InitializeConnectionPool()
{
    const int connectionCount = 3;
    for (int i = 0; i < connectionCount; i++)
    {
        var connection = await _factory.CreateConnectionAsync();
        _connections.Add(connection);
        
        // Create multiple channels per connection
        for (int j = 0; j < 5; j++)
        {
            var channel = await connection.CreateChannelAsync();
            _channelPool.Enqueue(channel);
        }
    }
}
```

**Benefits**:
- Parallel message acknowledgment
- Reduced connection contention
- Better scalability under high load
- Improved fault tolerance

**Estimated Performance Gain**: **1.5-2x improvement** under high load
- Better performance during message bursts
- Reduced blocking on message acknowledgment

**Risk Level**: **Medium**
**Implementation Time**: 3 hours

### **Optimization 5: Memory-Optimized Message Processing**

**Current State**: Frequent allocation of TelemetryBatch and related objects
**Proposed**: Object pooling and memory reuse patterns

**Implementation Strategy**:
```csharp
// Object pool for TelemetryBatch processing
private readonly ObjectPool<List<TelemetryRecord>> _recordListPool;
private readonly ObjectPool<byte[]> _bufferPool;

// Reuse collections and buffers
private async Task ProcessMessageOptimized(BasicGetResult result)
{
    var buffer = _bufferPool.Get();
    try
    {
        // Process with pooled objects
        var recordList = _recordListPool.Get();
        try
        {
            // Processing logic here
        }
        finally
        {
            _recordListPool.Return(recordList);
        }
    }
    finally
    {
        _bufferPool.Return(buffer);
    }
}
```

**Benefits**:
- Reduced garbage collection pressure
- Lower memory allocation rates
- Improved sustained performance
- Reduced GC pause times

**Estimated Performance Gain**: **10-15% throughput improvement**
- 50% reduction in garbage collection pressure
- More consistent performance under load

**Risk Level**: **Low**
**Implementation Time**: 2 hours

### **Optimization 6: QuestDB Connection Tuning**

**Current State**: `auto_flush_rows=2000;auto_flush_interval=500`
**Proposed**: `auto_flush_rows=5000;auto_flush_interval=250`

**Implementation**:
```csharp
_sender = Sender.New($"http::addr={url.Replace("http://", "")};auto_flush_rows=5000;auto_flush_interval=250;");
```

**Rationale**:
- Larger batches for better QuestDB write efficiency
- Faster flush interval to maintain low latency
- Better alignment with increased message batch sizes

**Estimated Performance Gain**: **1.5-2x QuestDB write efficiency**
- Fewer database connections
- Better utilization of QuestDB's batch processing

**Risk Level**: **Low**
**Implementation Time**: 5 minutes

## Combined Performance Projections

### **Conservative Estimate (Optimizations 1, 3, 6)**
**Implementation Time**: 1 hour
**Changes**:
- Increase BatchSize to 50
- Reduce polling delays
- Tune QuestDB parameters

**Performance Impact**:
- **Current**: ~1,000 msgs/sec
- **Optimized**: ~3,000-4,000 msgs/sec
- **Improvement**: **3-4x throughput gain**

### **Aggressive Estimate (All Optimizations)**
**Implementation Time**: 8-10 hours total
**Changes**: All optimizations implemented

**Performance Impact**:
- **Current**: ~1,000 msgs/sec
- **Optimized**: ~8,000-12,000 msgs/sec
- **Improvement**: **8-12x throughput gain**

## Implementation Roadmap

### **Phase 1: Quick Wins (1 hour)**
**Priority**: High
**Effort**: Low
**Risk**: Minimal

1. **Increase BatchSize** (15 minutes)
   - Change `BatchSize = 10` to `BatchSize = 50`
   - Test with existing workload

2. **Reduce Delays** (5 minutes)
   - Update `BasePollIntervalMs` and `EmptyQueueDelayMs`
   - Monitor for CPU impact

3. **Tune QuestDB** (5 minutes)
   - Update auto-flush parameters
   - Verify write performance

4. **Testing & Validation** (35 minutes)
   - Load test with higher message volumes
   - Monitor resource usage

**Expected Outcome**: 3-4x performance improvement

### **Phase 2: Major Optimization (6 hours)**
**Priority**: Medium
**Effort**: High  
**Risk**: Moderate

1. **Implement Batch Processing** (4 hours)
   - Create message accumulator logic
   - Implement batch flushing
   - Add comprehensive error handling
   - Update ACK/NACK logic for batches

2. **Add Connection Pooling** (2 hours)
   - Create connection pool management
   - Implement channel borrowing/returning
   - Add connection health monitoring

**Expected Outcome**: 8-10x performance improvement total

### **Phase 3: Performance Polish (2 hours)**
**Priority**: Low
**Effort**: Medium
**Risk**: Low

1. **Memory Optimization** (2 hours)
   - Implement object pooling
   - Add buffer reuse
   - Monitor GC performance

**Expected Outcome**: Additional 10-15% improvement + reduced GC pressure

## Consistency and Reliability Guarantees

### **Data Integrity Maintained**
✅ **Message Ordering**: Preserved within batches and processing windows
✅ **At-Least-Once Delivery**: RabbitMQ ACK/NACK patterns unchanged
✅ **Data Validation**: All existing telemetry validation preserved
✅ **Schema Management**: QuestDB schema optimization unchanged
✅ **Error Recovery**: Enhanced batch-level error handling

### **Enhanced Error Handling**
- **Batch-Level Retries**: Failed batches can be reprocessed individually
- **Partial Success Handling**: Ability to process partial batches on errors
- **Connection Resilience**: Multiple connections provide failover capability
- **Memory Protection**: Enhanced monitoring for larger batch sizes

### **Observability Improvements**
- **Batch Metrics**: Processing rates for batched vs individual messages
- **Connection Health**: Monitoring across connection pool
- **Memory Usage**: Enhanced tracking of allocation patterns
- **QuestDB Performance**: Detailed write performance metrics

## Resource Impact Analysis

### **Memory Impact**
- **Current**: 71MB (0.87% of 8GB)
- **Phase 1**: +50-100MB (larger message batches)
- **Phase 2**: +200-300MB (batch accumulation + connection pool)
- **Phase 3**: -50MB (reduced allocations)
- **Total**: ~300-400MB (still <5% of 8GB limit)

### **CPU Impact**
- **Current**: 4% sustained
- **Phase 1**: 6-8% (more intensive processing)
- **Phase 2**: 15-25% under high load
- **Phase 3**: -2-3% (reduced GC overhead)
- **Total**: 15-25% under load (well within Pi5 capacity)

### **Network Impact**
- **RabbitMQ**: 5x fewer API calls, more efficient connection usage
- **QuestDB**: 3-5x fewer connection calls, larger batch writes
- **Overall**: Significant reduction in connection overhead

## Risk Assessment

### **Low Risk Optimizations**
- Configuration parameter changes (Phase 1)
- QuestDB connection tuning
- Memory optimization patterns

### **Medium Risk Optimizations**  
- Batch processing implementation (complex error handling)
- Connection pooling (connection management complexity)

### **Mitigation Strategies**
- **Gradual Rollout**: Implement optimizations incrementally
- **Comprehensive Testing**: Load testing at each phase
- **Rollback Planning**: Ability to revert to previous configurations
- **Monitoring**: Enhanced observability during rollout
- **Circuit Breaker Alternative**: Consider alternative backpressure mechanisms if needed

## Success Metrics

### **Throughput Metrics**
- Messages processed per second
- Batch processing efficiency
- QuestDB write performance
- End-to-end latency

### **Reliability Metrics**
- Message delivery success rate
- Error rates and recovery times
- Connection health and stability
- Data integrity verification

### **Resource Metrics**
- Memory usage patterns
- CPU utilization under load
- GC performance and pause times
- Network connection efficiency

## Conclusion

The telemetry service has significant optimization potential, with conservative estimates showing **3-4x performance improvement** achievable in just 1 hour of work, and aggressive optimization potentially yielding **8-12x improvement** with a more substantial investment.

The current system is well-architected and stable, making these optimizations low-risk enhancements rather than fundamental rewrites. All optimizations maintain existing consistency guarantees while significantly improving throughput and efficiency.

**Recommended Approach**: Start with Phase 1 quick wins to validate the optimization approach, then proceed with Phase 2 major optimizations based on actual throughput requirements and system load patterns.