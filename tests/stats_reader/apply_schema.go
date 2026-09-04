package stats_reader_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	chdriver "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// applySchema reads the clickhouse schema.sql and executes each statement.
func applySchema(ctx context.Context, conn chdriver.Conn, log logger.LoggerInterface) error {
	paths := []string{
		"../../pkg/clickhouse/schema.sql",
		"../pkg/clickhouse/schema.sql",
		"pkg/clickhouse/schema.sql",
	}

	var data []byte
	var err error
	for _, p := range paths {
		data, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	if err != nil {
		cwd, _ := os.Getwd()
		for {
			if _, serr := os.Stat(filepath.Join(cwd, "justfile")); serr == nil {
				break
			}
			parent := filepath.Dir(cwd)
			if parent == cwd {
				break
			}
			cwd = parent
		}
		p := filepath.Join(cwd, "pkg/clickhouse/schema.sql")
		data, err = os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("could not find schema.sql: %w", err)
		}
	}

	for _, stmt := range strings.Split(string(data), ";") {
		// Strip comment lines from each statement
		var lines []string
		for _, line := range strings.Split(stmt, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "--") {
				continue
			}
			lines = append(lines, line)
		}
		stmt = strings.TrimSpace(strings.Join(lines, "\n"))
		if stmt == "" {
			continue
		}
		if err := conn.Exec(ctx, stmt); err != nil {
			log.Error("Failed to apply ClickHouse schema statement",
				zap.Error(err),
				zap.String("statement", stmt),
			)
			return fmt.Errorf("apply statement: %w", err)
		}
	}

	viper.SetDefault("CLICKHOUSE_DATABASE", "default")
	return nil
}
