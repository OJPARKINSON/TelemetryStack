package containers

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

func SpinUpQuestDB(t *testing.T, ctx context.Context, nw *testcontainers.DockerNetwork) *testcontainers.DockerContainer {

	container, err := testcontainers.Run(
		ctx,
		"questdb/questdb:latest",
		testcontainers.WithName("e2e-questdb"),
		testcontainers.WithExposedPorts("9000:9000", "8812:8812", "9009:9009", "9003:9003"),
		testcontainers.WithEnv(map[string]string{
			"QDB_SHARED_WORKER_COUNT":                   "12",
			"QDB_CAIRO_MAX_UNCOMMITTED_ROWS":            "2000000",   // Increased to 2M for larger batches
			"QDB_CAIRO_COMMIT_LAG":                      "120000",    // 2 min commit lag (increased from default 60s)
			"QDB_LINE_TCP_ENABLED":                      "true",
			"QDB_LINE_TCP_CONNECTION_POOL_CAPACITY":     "64",        // Match sender pool size
			"QDB_LINE_TCP_NET_CONNECTION_LIMIT":         "256",
			"QDB_LINE_TCP_RECV_BUFFER_SIZE":             "1048576",   // 1MB TCP receive buffer
			"QDB_HTTP_CONNECTION_POOL_INITIAL_CAPACITY": "64",
			"QDB_PG_NET_CONNECTION_LIMIT":               "128",
			"QDB_CAIRO_COMMIT_MODE":                     "nosync",    // Async commits for max throughput
			"QDB_CAIRO_WAL_ENABLED_DEFAULT":             "true",
			"JAVA_OPTS":                                 "-Xmx6g -Xms4g -XX:+UseG1GC -XX:MaxGCPauseMillis=50 -XX:ParallelGCThreads=8", // Reduced GC pause target
		}),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("8812/tcp"),
		),
		network.WithNetwork([]string{"questdb"}, nw),
	)
	if err != nil {
		t.Fatalf("Error running QuestDB server: %v", err)
	}

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate QuestDB container: %v", err)
		}
	})

	return container
}
