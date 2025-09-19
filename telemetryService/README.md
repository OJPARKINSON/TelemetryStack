# TelemetryService

High-performance C# telemetry processing service for IRacing data. Consumes Protocol Buffer messages from RabbitMQ and persists to QuestDB with automatic schema optimization and memory-aware processing.

## Quick Start

### Prerequisites
- .NET 8.0 SDK
- Docker and Docker Compose
- RabbitMQ (configured via docker-compose.yml)
- QuestDB (configured via docker-compose.yml)

### Running with Docker Compose (Recommended)
```bash
# Start entire system including telemetry service
docker compose up -d

# View telemetry service logs
docker compose logs -f telemetry-service

# Restart just the telemetry service
docker compose restart telemetry-service
```

### Running Locally for Development
```bash
cd telemetryService/telemetryService

# Build the solution
dotnet build

# Run the API version (includes web endpoints)
dotnet run --project src/TelemetryService.API

# OR run the Worker version (console only, lower overhead)
dotnet run --project src/TelemetryService.Worker
```

## Service Variants

### TelemetryService.API
- **Web API** with Swagger documentation
- **Health check endpoints** at `/api/health`
- **Prometheus metrics** at `/metrics`
- **Schema optimization** via `POST /api/health/optimize-schema`
- Background RabbitMQ processing via hosted service
- **Best for**: Development, monitoring, and integrated deployments

### TelemetryService.Worker
- **Console application** with no HTTP overhead
- Pure RabbitMQ → QuestDB processing
- Optimized for memory and CPU efficiency
- **Best for**: Production deployments and resource-constrained environments

## Configuration

### Environment Variables
```bash
# Required
QUESTDB_URL=tcp://questdb:9009

# Optional (defaults to embedded connection string)
RABBITMQ_URL=amqp://admin:changeme@rabbitmq:5672/

# ASP.NET Core (API only)
ASPNETCORE_ENVIRONMENT=Production
ASPNETCORE_URLS=http://+:80
```

### Docker Environment
Place a `.env` file in the working directory:
```bash
QUESTDB_URL=tcp://questdb:9009
```

## Architecture Overview

```
Go Ingest Service (7600X) 
    ↓ 32MB Protocol Buffer batches
RabbitMQ (Pi5)
    ↓ Pull-based consumption (50 concurrent workers)
TelemetryService (Pi5)
    ↓ Memory-aware processing with auto-pause
QuestDB (Pi5) 
    ↓ PostgreSQL wire protocol queries
Next.js Dashboard (Pi5)
```

### Key Components
- **Subscriber**: RabbitMQ consumer with memory monitoring and backpressure
- **QuestDbService**: High-throughput TCP writer with connection resilience  
- **QuestDbSchemaManager**: Automatic schema optimization and migration
- **TelemetryBackgroundService**: ASP.NET Core hosted service integration

## Performance Characteristics

### Memory Management
- **Working Set Limit**: 5GB (auto-pause processing)
- **Recovery Threshold**: 4GB (resume processing)
- **Monitoring Interval**: 5 seconds
- **Concurrent Workers**: 50 (with semaphore throttling)

### Throughput
- **Sustained**: 3-6 GB/hour telemetry processing
- **Peak**: 8-10 GB/hour burst capacity
- **Write Batching**: 10K rows or 1-second auto-flush
- **Memory Usage**: 2-3GB typical, 5GB limit

### Resource Usage (Pi5 Optimized)
- **CPU**: 1-2 cores during active processing
- **Memory**: 2-3GB working set, 6GB container limit
- **Network**: 200-400 Mbps sustained throughput
- **Storage**: Optimized for USB 3.0 SSD patterns

## Data Flow

### Message Processing Pipeline
1. **Go Ingest Service** processes .ibt files into 32MB Protocol Buffer batches
2. **RabbitMQ** queues batches on exchange `telemetry_topic` with routing key `telemetry.ticks`
3. **TelemetryService** pulls messages in batches of 10 with 50 concurrent workers
4. **QuestDbService** writes telemetry via TCP ingress (port 9009) with auto-flush
5. **Dashboard** queries QuestDB via PostgreSQL wire protocol (port 8812)

### Protocol Buffer Schema
```protobuf
message Telemetry {
    string session_id = 1;
    double speed = 2;
    double lap_dist_pct = 3;
    string track_name = 10;
    // ... 46 total fields including tire data, accelerations, etc.
    google.protobuf.Timestamp tick_time = 46;
}

message TelemetryBatch {
    repeated Telemetry records = 1;
    string batch_id = 2;
    uint32 worker_id = 4;
}
```

## Database Schema

### TelemetryTicks Table (Optimized)
```sql
CREATE TABLE TelemetryTicks (
    -- Indexed SYMBOL columns for categorical data
    session_id SYMBOL CAPACITY 50000 INDEX,
    track_name SYMBOL CAPACITY 100 INDEX,
    track_id SYMBOL CAPACITY 100 INDEX,
    lap_id SYMBOL CAPACITY 500,
    session_num SYMBOL CAPACITY 20,
    session_type SYMBOL CAPACITY 10,
    session_name SYMBOL CAPACITY 50,
    
    -- Core telemetry data
    car_id VARCHAR,
    gear INT,
    player_car_position LONG,
    speed DOUBLE,
    lap_dist_pct DOUBLE,
    session_time DOUBLE,
    lat DOUBLE,
    lon DOUBLE,
    
    -- Vehicle dynamics (FLOAT for efficiency)
    throttle FLOAT,
    brake FLOAT,
    steering_wheel_angle FLOAT,
    rpm FLOAT,
    velocity_x FLOAT,
    velocity_y FLOAT,
    velocity_z FLOAT,
    
    -- Tire pressures and temperatures
    lFpressure FLOAT,
    rFpressure FLOAT,
    lRpressure FLOAT,
    rRpressure FLOAT,
    lFtempM FLOAT,
    rFtempM FLOAT,
    lRtempM FLOAT,
    rRtempM FLOAT,
    
    timestamp TIMESTAMP
) TIMESTAMP(timestamp) PARTITION BY HOUR WITH maxUncommittedRows=1000000;

-- Performance indexes
ALTER TABLE TelemetryTicks ADD INDEX session_lap_idx (session_id, lap_id);
ALTER TABLE TelemetryTicks ADD INDEX track_session_idx (track_name, session_id);
ALTER TABLE TelemetryTicks ADD INDEX session_time_idx (session_id, session_time);
```

### Schema Management
- **Automatic Creation**: Creates optimized table if none exists
- **Migration Support**: Upgrades legacy schemas with backup/rollback
- **Orphan Cleanup**: Removes old session-based tables automatically
- **Index Optimization**: Adds composite indexes for common query patterns

## API Endpoints

### Health and Monitoring
```http
GET /api/health
# Returns service health status

POST /api/health/optimize-schema  
# Triggers manual schema optimization

GET /metrics
# Prometheus metrics endpoint
```

### Example Response
```json
{
    "message": "Test endpoint working!",
    "timestamp": "2024-01-15T10:30:00Z"
}
```

## Error Handling

### Memory Protection
- **OutOfMemoryException**: Automatic processing pause with recovery
- **Memory Monitoring**: 5-second intervals with detailed logging
- **Garbage Collection**: Forced cleanup during memory pressure
- **Graceful Degradation**: Temporary pause instead of crash

### Connection Resilience
- **RabbitMQ**: 10 retry attempts with exponential backoff
- **QuestDB**: Automatic sender reset on connection errors
- **Message Safety**: Acknowledgment only after successful write
- **Health Checks**: Dependency verification during startup

### Data Integrity
- **Validation**: Required field checks and data sanitization
- **Transaction Safety**: Atomic batch writes to QuestDB
- **Error Classification**: Connection vs. data format errors
- **Recovery Logic**: Automatic retry for transient failures

## Monitoring

### Key Metrics
- **Processing Rate**: Messages per second with 10-second reporting
- **Memory Usage**: Working set and GC memory with thresholds
- **Queue Health**: Processing semaphore availability
- **Write Performance**: QuestDB batch sizes and latency
- **Connection Status**: RabbitMQ and QuestDB connectivity

### Logging Output
```
🚀 Telemetry Service Starting...
📊 Initial Memory Usage: 0.15GB
🔗 Initializing QuestDB connections:
   TCP Ingress: tcp::addr=questdb:9009
   HTTP Schema: http://questdb:9000
🔧 Checking QuestDB schema optimization...
✅ TelemetryTicks table is already optimized
🔌 Starting telemetry service subscriber...
📥 Ready to pull messages from queue...
📊 Pull Stats: 1250 msgs processed, 125.0 msgs/sec, 45/50 threads available
🔄 Starting batch write to QuestDB: 500 records
✅ Successfully wrote 500/500 telemetry points in 45.2ms
```

## Development

### Project Structure
```
telemetryService/
├── src/
│   ├── TelemetryService.Domain/          # Protocol Buffers & Models
│   ├── TelemetryService.Application/     # Business Logic & DTOs  
│   ├── TelemetryService.Infrastructure/  # Database & Messaging
│   ├── TelemetryService.API/            # HTTP Web API
│   └── TelemetryService.Worker/         # Console Application
├── tests/                               # Unit & Integration Tests
├── Dockerfile                          # Multi-stage container build
├── Dockerfile.Worker                   # Optimized worker container
└── telemetryService.sln                # Solution file
```

### Building and Testing
```bash
# Restore dependencies
dotnet restore

# Build all projects
dotnet build

# Run tests
dotnet test

# Publish for deployment
dotnet publish src/TelemetryService.API -c Release -o publish/api
dotnet publish src/TelemetryService.Worker -c Release -o publish/worker
```

### Docker Build
```bash
# API version
docker build -t telemetry-service:api .

# Worker version  
docker build -f Dockerfile.Worker -t telemetry-service:worker .
```

## Troubleshooting

### Common Issues

#### High Memory Usage
```bash
# Check container memory limit
docker stats telemetry-service

# View memory monitoring logs
docker logs telemetry-service | grep "Memory Status"

# Trigger garbage collection via schema optimization
curl -X POST http://localhost/api/health/optimize-schema
```

#### RabbitMQ Connection Issues
```bash
# Verify RabbitMQ is accessible
docker exec telemetry-service ping rabbitmq

# Check RabbitMQ management interface
open http://localhost:15672  # admin/changeme

# View connection logs
docker logs telemetry-service | grep "RabbitMQ"
```

#### QuestDB Write Errors
```bash
# Check QuestDB HTTP interface
curl http://localhost:9000/

# Verify TCP ingress port
netstat -an | grep 9009

# Check table structure
curl "http://localhost:9000/exec?query=SHOW%20COLUMNS%20FROM%20TelemetryTicks"
```

### Performance Tuning

#### Memory Optimization
- Adjust `MaxConcurrentProcessing` in Subscriber.cs for available RAM
- Tune `auto_flush_rows` in QuestDbService for batch sizes
- Modify memory thresholds in `CheckMemoryPressure()`

#### Throughput Optimization  
- Increase `BatchSize` for higher message pull rates
- Adjust `BasePollIntervalMs` for processing latency vs. CPU usage
- Tune `prefetch` settings in RabbitMQ channel configuration

#### Database Performance
- Monitor QuestDB partition sizes for query performance
- Add custom indexes for specific query patterns
- Adjust `maxUncommittedRows` for write performance vs. memory

For detailed architecture information, see [ARCHITECTURE.md](ARCHITECTURE.md).