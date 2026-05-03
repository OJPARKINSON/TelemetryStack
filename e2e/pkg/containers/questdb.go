package containers

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

func SpinUpQuestDB(t *testing.T, ctx context.Context, nw *testcontainers.DockerNetwork) *testcontainers.DockerContainer {
	const name = "e2e-questdb"
	removeContainerByName(ctx, name)

	container, err := testcontainers.Run(
		ctx,
		"questdb/questdb:latest",
		testcontainers.WithName(name),
		testcontainers.WithExposedPorts("9002:9000", "8812:8812", "9009:9009", "9003:9003"),
		testcontainers.WithEnv(map[string]string{
			"QDB_SHARED_WORKER_COUNT":                   "12",
			"QDB_CAIRO_MAX_UNCOMMITTED_ROWS":            "2000000",
			"QDB_CAIRO_COMMIT_LAG":                      "120000",
			"QDB_LINE_TCP_ENABLED":                      "true",
			"QDB_LINE_TCP_CONNECTION_POOL_CAPACITY":     "64",
			"QDB_LINE_TCP_NET_CONNECTION_LIMIT":         "256",
			"QDB_LINE_TCP_RECV_BUFFER_SIZE":             "1048576",
			"QDB_HTTP_CONNECTION_POOL_INITIAL_CAPACITY": "64",
			"QDB_PG_NET_CONNECTION_LIMIT":               "128",
			"QDB_CAIRO_COMMIT_MODE":                     "nosync",
			"QDB_CAIRO_WAL_ENABLED_DEFAULT":             "true",
			"JAVA_OPTS":                                 "-Xmx6g -Xms4g -XX:+UseG1GC -XX:MaxGCPauseMillis=50 -XX:ParallelGCThreads=8",
		}),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("8812/tcp"),
		),
		network.WithNetwork([]string{"questdb"}, nw),
	)
	testcontainers.CleanupContainer(t, container)
	if err != nil {
		t.Fatalf("Error running QuestDB server: %v", err)
	}

	return container
}
