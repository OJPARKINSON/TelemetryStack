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
			"JAVA_OPTS":                     "-Xmx2g -Xms1g -XX:+UseG1GC -XX:MaxGCPauseMillis=100",
			"QDB_HTTP_ENABLED":              "true",
			"QDB_PG_ENABLED":                "true",
			"QDB_LINE_TCP_ENABLED":          "true",
			"QDB_METRICS_ENABLED":           "true",
			"QDB_CAIRO_WAL_ENABLED_DEFAULT": "true",
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
