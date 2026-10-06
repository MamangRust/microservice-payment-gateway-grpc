package clickhouse

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

//go:embed schema.sql
var schemaFS embed.FS

// EnsureDatabase creates the database named by CLICKHOUSE_DATABASE if it does
// not exist yet. It must run before NewClient, whose Ping selects that database
// and therefore fails while the database is still missing.
func EnsureDatabase(l logger.LoggerInterface) error {
	dbName := viper.GetString("CLICKHOUSE_DATABASE")
	if dbName == "" {
		l.Debug("CLICKHOUSE_DATABASE is not set, skipping database creation")
		return nil
	}

	conn, err := openConn(l, "")
	if err != nil {
		return err
	}
	defer func() {
		if cerr := conn.Close(); cerr != nil {
			l.Error("Failed to close ClickHouse connection", zap.Error(cerr))
		}
	}()

	stmt := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", strings.ReplaceAll(dbName, "`", "``"))
	if err := conn.Exec(context.Background(), stmt); err != nil {
		l.Error("Failed to create ClickHouse database",
			zap.String("database", dbName),
			zap.Error(err),
		)
		return fmt.Errorf("failed to create clickhouse database %q: %w", dbName, err)
	}

	l.Debug("ClickHouse database ensured", zap.String("database", dbName))
	return nil
}

// ApplySchema executes the embedded schema.sql against conn. Every statement is
// idempotent (CREATE TABLE IF NOT EXISTS), so both the stats reader and writer
// can safely run it on startup even when the ClickHouse volume already exists.
func ApplySchema(ctx context.Context, conn clickhouse.Conn, l logger.LoggerInterface) error {
	content, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read embedded clickhouse schema: %w", err)
	}

	for _, stmt := range splitStatements(string(content)) {
		if err := conn.Exec(ctx, stmt); err != nil {
			l.Error("Failed to apply ClickHouse schema statement", zap.Error(err))
			return fmt.Errorf("failed to apply clickhouse schema: %w", err)
		}
	}

	l.Debug("ClickHouse schema applied")
	return nil
}

// splitStatements breaks a SQL script into individual statements. The ClickHouse
// native protocol executes one statement per Exec, so a multi-statement script
// cannot be sent as-is.
func splitStatements(sql string) []string {
	var stripped strings.Builder
	for _, line := range strings.Split(sql, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		stripped.WriteString(line)
		stripped.WriteString("\n")
	}

	var stmts []string
	for _, raw := range strings.Split(stripped.String(), ";") {
		if stmt := strings.TrimSpace(raw); stmt != "" {
			stmts = append(stmts, stmt)
		}
	}
	return stmts
}
