package database

import (
	"context"
	"fmt"
	"time"

	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func NewClient(logger logger.LoggerInterface) (*pgxpool.Pool, error) {
	return NewClientWithPrefix(logger, "DB")
}

func NewClientA(logger logger.LoggerInterface) (*pgxpool.Pool, error) {
	return NewClientWithPrefix(logger, "DB_A")
}

func NewClientB(logger logger.LoggerInterface) (*pgxpool.Pool, error) {
	return NewClientWithPrefix(logger, "DB_B")
}

func NewClientC(logger logger.LoggerInterface) (*pgxpool.Pool, error) {
	return NewClientWithPrefix(logger, "DB_C")
}

func NewClientD(logger logger.LoggerInterface) (*pgxpool.Pool, error) {
	return NewClientWithPrefix(logger, "DB_D")
}

func NewClientWithPrefix(logger logger.LoggerInterface, prefix string) (*pgxpool.Pool, error) {
	if prefix == "" {
		prefix = "DB"
	}
	dbDriver := viper.GetString(fmt.Sprintf("%s_DRIVER", prefix))
	if dbDriver == "" {
		dbDriver = viper.GetString("DB_DRIVER")
	}

	if dbDriver != "postgres" && dbDriver != "pgx" {
		logger.Error("pgxpool only supports PostgreSQL", zap.String("DB_DRIVER", dbDriver))
		return nil, fmt.Errorf("pgxpool only supports PostgreSQL, got: %s", dbDriver)
	}

	hostKey := fmt.Sprintf("%s_HOST", prefix)
	portKey := fmt.Sprintf("%s_PORT", prefix)
	userKey := fmt.Sprintf("%s_USERNAME", prefix)
	nameKey := fmt.Sprintf("%s_NAME", prefix)
	passKey := fmt.Sprintf("%s_PASSWORD", prefix)

	// Fallback to default if prefix-specific keys are not set
	host := viper.GetString(hostKey)
	if host == "" { host = viper.GetString("DB_HOST") }
	port := viper.GetString(portKey)
	if port == "" { port = viper.GetString("DB_PORT") }
	user := viper.GetString(userKey)
	if user == "" { user = viper.GetString("DB_USERNAME") }
	dbname := viper.GetString(nameKey)
	if dbname == "" { dbname = viper.GetString("DB_NAME") }
	password := viper.GetString(passKey)
	if password == "" { password = viper.GetString("DB_PASSWORD") }

	connStr := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable",
		host, port, user, dbname, password,
	)

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		logger.Error("Failed to parse database config", zap.Error(err))
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	maxOpenConns := viper.GetInt(fmt.Sprintf("%s_MAX_OPEN_CONNS", prefix))
	if maxOpenConns <= 0 {
		maxOpenConns = viper.GetInt("DB_MAX_OPEN_CONNS")
		if maxOpenConns <= 0 {
			maxOpenConns = 100
		}
	}
	config.MaxConns = int32(maxOpenConns)

	minIdleConns := viper.GetInt(fmt.Sprintf("%s_MIN_IDLE_CONNS", prefix))
	if minIdleConns <= 0 {
		minIdleConns = viper.GetInt("DB_MIN_IDLE_CONNS")
		if minIdleConns <= 0 {
			minIdleConns = 50
		}
	}
	config.MinConns = int32(minIdleConns)

	connMaxLifetime := viper.GetDuration(fmt.Sprintf("%s_CONN_MAX_LIFETIME", prefix))
	if connMaxLifetime == 0 {
		connMaxLifetime = viper.GetDuration("DB_CONN_MAX_LIFETIME")
		if connMaxLifetime == 0 {
			connMaxLifetime = time.Hour
		}
	}
	config.MaxConnLifetime = connMaxLifetime

	connMaxIdleTime := viper.GetDuration(fmt.Sprintf("%s_CONN_MAX_IDLE_TIME", prefix))
	if connMaxIdleTime == 0 {
		connMaxIdleTime = viper.GetDuration("DB_CONN_MAX_IDLE_TIME")
		if connMaxIdleTime == 0 {
			connMaxIdleTime = 30 * time.Minute
		}
	}
	config.MaxConnIdleTime = connMaxIdleTime

	healthCheckPeriod := viper.GetDuration(fmt.Sprintf("%s_HEALTH_CHECK_PERIOD", prefix))
	if healthCheckPeriod == 0 {
		healthCheckPeriod = viper.GetDuration("DB_HEALTH_CHECK_PERIOD")
		if healthCheckPeriod == 0 {
			healthCheckPeriod = time.Minute
		}
	}
	config.HealthCheckPeriod = healthCheckPeriod

	ctx := context.Background()
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		logger.Error("Failed to create connection pool", zap.Error(err), zap.String("prefix", prefix))
		return nil, fmt.Errorf("failed to create connection pool for %s: %w", prefix, err)
	}

	if err := pool.Ping(ctx); err != nil {
		logger.Error("Failed to ping database", zap.Error(err), zap.String("prefix", prefix))
		pool.Close()
		return nil, fmt.Errorf("failed to ping database %s: %w", prefix, err)
	}

	logger.Debug("Database connection pool established successfully",
		zap.String("prefix", prefix),
		zap.String("DB_DRIVER", dbDriver),
		zap.Int32("MaxConns", config.MaxConns),
		zap.Int32("MinConns", config.MinConns),
		zap.Duration("MaxConnLifetime", config.MaxConnLifetime),
		zap.Duration("MaxConnIdleTime", config.MaxConnIdleTime),
		zap.Duration("HealthCheckPeriod", config.HealthCheckPeriod),
	)

	return pool, nil
}
