package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func resolveAddr() string {
	addr := viper.GetString("CLICKHOUSE_ADDR")
	if addr != "" {
		return addr
	}

	host := viper.GetString("CLICKHOUSE_HOST")
	if host == "" {
		host = "clickhouse"
	}
	port := viper.GetString("CLICKHOUSE_PORT")
	if port == "" {
		port = "9000"
	}
	return fmt.Sprintf("%s:%s", host, port)
}

// openConn dials ClickHouse with the given database selected. An empty database
// lets the driver fall back to "default", which is what EnsureDatabase needs
// before the configured database exists.
func openConn(l logger.LoggerInterface, database string) (clickhouse.Conn, error) {
	addr := resolveAddr()

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: database,
			Username: viper.GetString("CLICKHOUSE_USERNAME"),
			Password: viper.GetString("CLICKHOUSE_PASSWORD"),
		},
		DialTimeout: time.Second * 30,
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	})

	if err != nil {
		l.Error("Failed to open ClickHouse connection", zap.Error(err))
		return nil, fmt.Errorf("failed to open clickhouse connection: %w", err)
	}

	return conn, nil
}

func NewClient(l logger.LoggerInterface) (clickhouse.Conn, error) {
	addr := resolveAddr()

	conn, err := openConn(l, viper.GetString("CLICKHOUSE_DATABASE"))
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(context.Background()); err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			l.Error("ClickHouse exception during ping", 
				zap.Uint32("code", uint32(exception.Code)), 
				zap.String("message", exception.Message),
			)
		}
		return nil, fmt.Errorf("failed to ping clickhouse: %w", err)
	}

	l.Debug("ClickHouse connection established successfully", zap.String("addr", addr))
	return conn, nil
}
