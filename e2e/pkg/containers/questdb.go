package containers

import (
	"context"
	"fmt"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func SpinUpQuestDB(t *testing.T, ctx context.Context) *testcontainers.DockerContainer {
	container, err := testcontainers.Run(
		ctx,
		"questdb/questdb:latest",
		testcontainers.WithExposedPorts("9000:9000", "8812:8812", "9009:9009", "9003:9003"),
		testcontainers.WithEnv(map[string]string{
			"JAVA_OPTS":                     "-Xmx2g -Xms1g -XX:+UseG1GC -XX:MaxGCPauseMillis=100",
			"DB_HTTP_ENABLED":               "true",
			"QDB_PG_ENABLED":                "true",
			"QDB_LINE_TCP_ENABLED":          "true",
			"QDB_METRICS_ENABLED":           "true",
			"QDB_CAIRO_WAL_ENABLED_DEFAULT": "true",
		}),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("8812/tcp"),
			// wait.ForLog("Ready to accept connections"),
		),
	)
	if err != nil {
		fmt.Println("Error running questDB server, ", err)
	}

	return container
}
