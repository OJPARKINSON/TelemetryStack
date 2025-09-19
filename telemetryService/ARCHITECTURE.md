# TelemetryService Architecture Documentation

## Overview

The TelemetryService is a high-performance C# telemetry processing system that consumes IRacing telemetry data from RabbitMQ and persists it to QuestDB for real-time dashboard visualization. The service is designed for distributed deployment, optimized for Pi5 (16GB RAM) with memory-aware processing and automatic schema management.

## Architecture Diagram

```mermaid
graph TB
    subgraph "External Systems"
        GO[Go Ingest Service<br/>7600X System]
        RABBIT[RabbitMQ<br/>Pi5 System]
        QUEST[QuestDB<br/>Pi5 System]
        DASH[Next.js Dashboard<br/>Pi5 System]
    end

    subgraph "TelemetryService (Pi5)"
        subgraph "Entry Points"
            API[TelemetryService.API<br/>HTTP Web API]
            WORKER[TelemetryService.Worker<br/>Console Application]
        end

        subgraph "Core Components"
            BG[TelemetryBackgroundService<br/>Hosted Service]
            SUB[Subscriber<br/>RabbitMQ Consumer]
            QUEST_SVC[QuestDbService<br/>Database Writer]
            SCHEMA[QuestDbSchemaManager<br/>Schema Optimizer]
        end

        subgraph "Domain Layer"
            PROTO[telemetryTick.proto<br/>Protocol Buffers]
            MODEL[TelemetryTick.cs<br/>Domain Models]
        end

        subgraph "Infrastructure"
            MSG[Messaging Layer<br/>RabbitMQ Client]
            PERSIST[Persistence Layer<br/>QuestDB TCP Client]
            CONFIG[Configuration<br/>Environment Variables]
        end
    end

    %% Data Flow
    GO -->|Protocol Buffer Batches 32MB batches| RABBIT
    RABBIT -->|Pull-based consumption 50 concurrent workers| SUB
    SUB -->|Deserialize batches| MODEL
    SUB -->|Write telemetry data| QUEST_SVC
    QUEST_SVC -->|TCP Ingress port 9009 Auto-flush batches| QUEST
    DASH -->|PostgreSQL wire port 8812 Query telemetry| QUEST

    %% Service Management
    API --> BG
    WORKER --> SUB
    BG --> SUB
    SUB --> QUEST_SVC
    QUEST_SVC --> SCHEMA
    SCHEMA -->|HTTP port 9000 Schema operations| QUEST

    %% Memory Management
    SUB -.->|Memory monitoring 5s intervals| SUB
    SUB -.->|Pause on 5GB+ Resume on 4GB| SUB

    style GO fill:#e1f5fe
    style RABBIT fill:#fff3e0
    style QUEST fill:#f3e5f5
    style DASH fill:#e8f5e8
    style API fill:#ffebee
    style WORKER fill:#ffebee
    style SUB fill:#fff8e1
    style QUEST_SVC fill:#f1f8e9
```

## System Components

### 1. Entry Points

#### TelemetryService.API
- **Purpose**: HTTP Web API for external interactions and monitoring
- **Key Features**:
  - Swagger/OpenAPI documentation
  - Health check endpoints
  - Schema optimization triggers
  - Prometheus metrics exposure at `/metrics`
  - CORS support for dashboard integration
- **Deployment**: Runs as web service with background processing
- **File**: `src/TelemetryService.API/Program.cs`

#### TelemetryService.Worker  
- **Purpose**: Dedicated console application for pure message processing
- **Key Features**:
  - Standalone RabbitMQ consumer
  - Memory monitoring and crash recovery
  - Optimized for Docker deployment
  - Lower overhead than web API
- **Deployment**: Runs as background service/daemon
- **File**: `src/TelemetryService.Worker/Program.cs`

### 2. Core Processing Components

#### TelemetryBackgroundService
- **Purpose**: Hosted service bridge for API deployment
- **Features**:
  - Integrates RabbitMQ consumption with ASP.NET Core
  - Graceful shutdown handling
  - Exception logging and monitoring
- **File**: `src/TelemetryService.API/BackgroundServices/TelemetryBackgroundService.cs`

#### Subscriber (RabbitMQ Consumer)
- **Purpose**: High-performance message consumer with memory management
- **Key Features**:
  - **Pull-based consumption** (not push) for better control
  - **50 concurrent workers** with semaphore throttling
  - **Memory monitoring** every 5 seconds
  - **Automatic pause/resume** on memory pressure (5GB threshold)
  - **Batch processing** of 10 messages per pull cycle
  - **Connection resilience** with retry logic (10 attempts)
  - **OutOfMemoryException recovery** with temporary processing pause
- **Performance Optimizations**:
  - 1000 message prefetch buffer
  - 10ms base polling interval when active
  - 100ms delay when queue empty
  - Message acknowledgment after successful QuestDB write
- **File**: `src/TelemetryService.Infrastructure/Messaging/Subscriber.cs`

#### QuestDbService (Database Writer)
- **Purpose**: QuestDB TCP client with connection management
- **Key Features**:
  - **TCP Ingress Protocol** (port 9009) for high-throughput writes
  - **Auto-flush** every 10,000 rows or 1 second
  - **Data sanitization** to prevent line protocol errors
  - **Connection error detection** and automatic sender reset
  - **Memory-aware processing** with OutOfMemoryException handling
  - **Concurrent write safety** with semaphore locks
- **Data Validation**:
  - NaN/Infinity value filtering
  - String sanitization (removes line protocol special characters)
  - Required field validation (session_id, track_name)
- **File**: `src/TelemetryService.Infrastructure/Persistance/QuestService.cs`

#### QuestDbSchemaManager
- **Purpose**: Automatic database schema optimization and management
- **Key Features**:
  - **Automatic table creation** with optimized structure
  - **Schema migration** from legacy formats to optimized layout
  - **Index management** for performance-critical columns
  - **Orphaned table cleanup** (removes old session-based tables)
  - **Composite index creation** for common query patterns
  - **Backup and rollback** during schema migrations
- **Optimizations Applied**:
  - SYMBOL columns with capacity limits for categorical data
  - Partitioning by HOUR with 1M uncommitted row buffer
  - Indexes on session_id, track_name, track_id
  - Composite indexes for session+lap, track+session queries
- **File**: `src/TelemetryService.Infrastructure/Persistance/QuestDbSchemaManager.cs`

### 3. Domain Layer

#### Protocol Buffers Schema
- **Purpose**: Efficient binary serialization format for telemetry data
- **Structure**:
  - `Telemetry`: Individual telemetry record (46 fields)
  - `TelemetryBatch`: Container for multiple records with metadata
- **Fields Include**: Speed, position, engine data, tire pressures/temps, accelerations, etc.
- **File**: `src/TelemetryService.Domain/telemetryTick.proto`

#### Domain Models
- **Purpose**: C# classes generated from Protocol Buffers
- **Features**: Strongly-typed telemetry data structures
- **Generated From**: `telemetryTick.proto` during build process

### 4. Infrastructure Layer

#### Messaging Infrastructure
- **RabbitMQ Configuration**:
  - **Exchange**: `telemetry_topic` (Topic type)
  - **Queue**: `telemetry_queue` (Durable)
  - **Routing Key**: `telemetry.ticks`
  - **Connection**: `amqp://admin:changeme@rabbitmq:5672/`

#### Persistence Infrastructure  
- **QuestDB Protocols**:
  - **TCP Ingress** (port 9009): High-throughput data writes
  - **HTTP API** (port 9000): Schema management operations
  - **PostgreSQL Wire** (port 8812): Query interface (used by dashboard)

#### Configuration Management
- **Environment Variables**: Database URLs, connection strings
- **DotEnv Support**: Loads `.env` files for local development
- **Container-First**: Designed for Docker deployment

## Data Flow Architecture

### 1. Ingest Pipeline
```
Go Ingest Service (7600X)
  ↓ Protocol Buffer batches (32MB)
RabbitMQ (Pi5)
  ↓ Pull-based consumption (50 workers)
TelemetryService Subscriber
  ↓ Deserialize and validate
QuestDbService
  ↓ TCP Ingress batches (10K rows)
QuestDB (Pi5)
```

### 2. Query Pipeline
```
Next.js Dashboard (Pi5)
  ↓ PostgreSQL wire protocol
QuestDB (Pi5)
  ↓ SQL queries with indexes
Optimized TelemetryTicks table
```

### 3. Schema Management
```
TelemetryService Startup
  ↓ Check existing schema
QuestDbSchemaManager
  ↓ Apply optimizations if needed
  ├─ Create/migrate table structure
  ├─ Add performance indexes  
  ├─ Clean up orphaned tables
  └─ Verify table statistics
QuestDB (Optimized)
```

## Performance Characteristics

### Memory Management
- **Working Set Limit**: 5GB (83% of 6GB container limit)
- **Memory Monitoring**: 5-second intervals with automatic logging
- **Pressure Response**: Pause processing above 5GB, resume below 4GB
- **GC Optimization**: Forced collection on OutOfMemoryException
- **Semaphore Control**: 50 concurrent workers max

### Throughput Metrics
- **Sustained Processing**: 3-6 GB/hour telemetry data
- **Peak Capacity**: 8-10 GB/hour burst processing
- **Message Rate**: 200K-800K messages/second (RabbitMQ capacity)
- **Write Performance**: 10K rows auto-flush, 1-second intervals
- **Query Performance**: Sub-50ms with proper indexing

### Resource Optimization (Pi5)
- **Memory Usage**: 2-3GB during normal operation
- **CPU Utilization**: 1-2 cores during processing
- **Network**: 200-400 Mbps sustained throughput
- **Storage**: Optimized for USB 3.0 SSD patterns

## Deployment Configurations

### Docker Compose Integration
```yaml
# From main docker-compose.yml
telemetry-service:
  build: ./telemetryService/telemetryService
  container_name: telemetry-service
  environment:
    - QUESTDB_URL=tcp://questdb:9009
  depends_on:
    - questdb
    - rabbitmq
  networks:
    - telemetry_network
  deploy:
    resources:
      limits:
        memory: 6G
        cpus: '2.0'
```

### Environment Variables
- `QUESTDB_URL`: QuestDB TCP connection string
- `RABBITMQ_URL`: RabbitMQ AMQP connection (optional, defaults embedded)
- Standard ASP.NET Core configuration variables

## Monitoring and Observability

### Health Checks
- **HTTP Endpoint**: `/api/health` - Service status
- **Schema Optimization**: `POST /api/health/optimize-schema`
- **Prometheus Metrics**: `/metrics` endpoint

### Logging Strategy
- **Structured Logging**: JSON format with correlation IDs
- **Performance Metrics**: Processing rates, memory usage, queue depths
- **Error Classification**: Connection errors vs. data errors
- **Memory Alerts**: Warnings at 4.5GB, critical at 5GB

### Key Metrics Tracked
- Messages processed per second
- Memory usage (working set + GC)
- QuestDB write latency and batch sizes
- RabbitMQ connection health
- Schema optimization status

## Error Handling and Resilience

### Connection Resilience
- **RabbitMQ**: 10 retry attempts with 5-second delays
- **QuestDB**: Automatic sender reset on connection errors
- **Health Checks**: Wait for dependencies during startup

### Memory Protection
- **OutOfMemoryException**: Graceful handling with processing pause
- **Garbage Collection**: Forced cleanup during memory pressure
- **Backpressure**: Natural throttling through semaphore limits

### Data Integrity
- **Message Acknowledgment**: Only after successful QuestDB write
- **Transaction Safety**: Atomic batch writes to QuestDB
- **Data Validation**: Sanitization and required field checks
- **Schema Migrations**: Backup and rollback capabilities

## Development and Testing

### Project Structure (Clean Architecture)
```
TelemetryService.Domain/     # Core business models and Protocol Buffers
TelemetryService.Application/ # Business logic and DTOs
TelemetryService.Infrastructure/ # External dependencies (DB, messaging)
TelemetryService.API/        # HTTP Web API entry point  
TelemetryService.Worker/     # Console application entry point
```

### Testing Strategy
- **Unit Tests**: Core business logic and data transformation
- **Integration Tests**: RabbitMQ and QuestDB connectivity
- **Performance Tests**: Memory usage and throughput validation
- **Messaging Tests**: Protocol Buffer serialization/deserialization

### Build and Deployment
- **Multi-stage Dockerfile**: Optimized for production deployment
- **Health Checks**: Container readiness and liveness probes
- **Resource Limits**: Memory and CPU constraints for Pi5 deployment
- **Dependency Management**: Proper service startup ordering

This architecture delivers a robust, high-performance telemetry processing system optimized for the constraints and capabilities of Raspberry Pi 5 hardware while maintaining the ability to scale with higher-end systems.