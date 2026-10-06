package tests

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	chdriver "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/clickhouse"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	role "github.com/MamangRust/microservice-payment-gateway-grpc/pb/role"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pb/user"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/auth"
	clickhousepkg "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/clickhouse"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/hash"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	carddb "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/database/schema"
	card_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/repository"
	merchantdb "github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/database/schema"
	merchant_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/repository"
	roledb "github.com/MamangRust/microservice-payment-gateway-grpc/service/role/database/schema"
	role_handler "github.com/MamangRust/microservice-payment-gateway-grpc/service/role/handler"
	role_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/role/repository"
	role_service "github.com/MamangRust/microservice-payment-gateway-grpc/service/role/service"
	saldodb "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/database/schema"
	saldo_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/repository"
	userdb "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/database/schema"
	user_handler "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/handler"
	user_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/repository"
	user_service "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/service"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/cache"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/observability"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	goredis "github.com/redis/go-redis/v9"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"os"
	"path/filepath"
)

type TestSuite struct {
	PGContainer    *postgres.PostgresContainer
	RedisContainer *redis.RedisContainer
	CHContainer    *clickhouse.ClickHouseContainer
	DBURL          string
	RedisURL       string
	CHURL          string
	Ctx            context.Context
	RootDir        string

	UserAdapter     adapter.UserAdapter
	CardAdapter     adapter.CardAdapter
	MerchantAdapter adapter.MerchantAdapter
	SaldoAdapter    adapter.SaldoAdapter

	// Local gRPC Clients for Auth/Identity testing
	UserQueryClient   user.UserQueryServiceClient
	UserCommandClient user.UserCommandServiceClient
	RoleQueryClient   role.RoleQueryServiceClient
	RoleCommandClient role.RoleCommandServiceClient

	// Aliases for convenience (using concrete types to implement both Query and Command interfaces)
	UserClient *LocalUserClient
	RoleClient *LocalRoleClient

	// Shared resources
	Logger        logger.LoggerInterface
	CacheStore    *cache.CacheStore
	Observability observability.TraceLoggerObservability
	Hashing       hash.HashPassword
	TokenManager  auth.TokenManager
}

func SetupTestSuite() (*TestSuite, error) {
	ctx := context.Background()

	// Setup PostgreSQL
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	dbURL, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("failed to get postgres connection string: %w", err)
	}

	// Setup Redis
	redisContainer, err := redis.Run(ctx, "redis:7-alpine")
	if err != nil {
		return nil, fmt.Errorf("failed to start redis container: %w", err)
	}

	redisURL, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get redis connection string: %w", err)
	}

	// Setup ClickHouse
	chContainer, err := clickhouse.Run(ctx,
		"clickhouse/clickhouse-server:24.3-alpine",
		clickhouse.WithDatabase("testdb"),
		clickhouse.WithUsername("testuser"),
		clickhouse.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForHTTP("/ping").WithPort("8123/tcp").WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start clickhouse container: %w", err)
	}

	chURL, err := chContainer.ConnectionString(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get clickhouse connection string: %w", err)
	}

	ts := &TestSuite{
		PGContainer:    pgContainer,
		RedisContainer: redisContainer,
		CHContainer:    chContainer,
		DBURL:          dbURL,
		RedisURL:       redisURL,
		CHURL:          chURL,
		Ctx:            ctx,
	}

	// Find project root
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get cwd: %w", err)
	}

	root := cwd
	for {
		if _, err := os.Stat(filepath.Join(root, "justfile")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			return nil, fmt.Errorf("could not find justfile in any parent directory")
		}
		root = parent
	}
	ts.RootDir = root

	// Initialize Adapters
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open pgxpool: %w", err)
	}
	userQueries := userdb.New(pool)
	cardQueries := carddb.New(pool)
	merchantQueries := merchantdb.New(pool)
	saldoQueries := saldodb.New(pool)
	roleQueries := roledb.New(pool)

	userRepo := user_repo.NewUserQueryRepository(userQueries)
	ts.UserAdapter = NewLocalUserAdapter(userRepo)

	cardQueryRepo := card_repo.NewCardQueryRepository(cardQueries)
	cardCommandRepo := card_repo.NewCardCommandRepository(cardQueries)
	ts.CardAdapter = NewLocalCardAdapter(cardQueryRepo, cardCommandRepo)

	merchantRepo := merchant_repo.NewMerchantQueryRepository(merchantQueries)
	ts.MerchantAdapter = NewLocalMerchantAdapter(merchantRepo)

	saldoRepos := saldo_repo.NewRepositories(saldoQueries, nil, nil)
	ts.SaldoAdapter = NewLocalSaldoAdapter(saldoRepos)

	// Initialize Logging, Cache and Observability for local services
	logger.ResetInstance()
	lp := sdklog.NewLoggerProvider()
	ts.Logger, _ = logger.NewLogger("test-integration", lp)

	redisOpts, err := goredis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}
	redisClient := goredis.NewClient(redisOpts)
	cacheMetrics, _ := observability.NewCacheMetrics("test-integration")
	ts.CacheStore = cache.NewCacheStore(redisClient, ts.Logger, cacheMetrics)
	ts.Observability, _ = observability.NewObservability("test-integration", ts.Logger)

	// Initialize Local gRPC Clients for Auth testing
	ts.Hashing = hash.NewHashingPassword()
	ts.TokenManager, _ = auth.NewManager("test-secret-key")

	userRepos := user_repo.NewRepositories(&user_repo.Deps{Db: userQueries})
	userService := user_service.NewService(&user_service.Deps{
		Repositories: userRepos,
		Hash:         ts.Hashing,
		Logger:       ts.Logger,
		Cache:        ts.CacheStore,
	})
	userHandler := user_handler.NewHandler(userService)
	uClient := &LocalUserClient{Handler: userHandler}
	ts.UserQueryClient = uClient
	ts.UserCommandClient = uClient

	roleRepos := role_repo.NewRepositories(roleQueries)
	roleService := role_service.NewService(&role_service.Deps{
		Repositories: roleRepos,
		Logger:       ts.Logger,
		Cache:        ts.CacheStore,
	})
	roleHandler := role_handler.NewHandler(roleService)
	rClient := &LocalRoleClient{Handler: roleHandler}
	ts.RoleQueryClient = rClient
	ts.RoleCommandClient = rClient

	ts.UserClient = uClient
	ts.RoleClient = rClient

	return ts, nil
}

func (ts *TestSuite) RunMigrations(serviceNames ...string) error {
	var relPaths []string
	for _, name := range serviceNames {
		relPaths = append(relPaths, filepath.Join("service", name, "database", "migration"))
	}
	return ts.RunAllMigrations(ts.RootDir, relPaths)
}

func (ts *TestSuite) RunServiceMigrations(serviceName string) error {
	return ts.RunMigrations(serviceName)
}

func (ts *TestSuite) RunAllMigrations(root string, relPaths []string) error {
	tempDir, err := os.MkdirTemp("", "migrations-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	for _, relPath := range relPaths {
		srcDir := filepath.Join(root, relPath)
		entries, err := os.ReadDir(srcDir)
		if err != nil {
			return fmt.Errorf("failed to read migrations from %s: %w", srcDir, err)
		}

		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sql" {
				srcFile := filepath.Join(srcDir, entry.Name())
				destFile := filepath.Join(tempDir, entry.Name())

				content, err := os.ReadFile(srcFile)
				if err != nil {
					return fmt.Errorf("failed to read migration file %s: %w", srcFile, err)
				}
				if err := os.WriteFile(destFile, content, 0644); err != nil {
					return fmt.Errorf("failed to write migration file %s: %w", destFile, err)
				}
			}
		}
	}

	db, err := goose.OpenDBWithDriver("pgx", ts.DBURL)
	if err != nil {
		return fmt.Errorf("failed to open db: %w", err)
	}
	defer db.Close()

	if err := goose.Up(db, tempDir); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}

func (ts *TestSuite) Teardown() {
	if ts.PGContainer != nil {
		if err := ts.PGContainer.Terminate(ts.Ctx); err != nil {
			log.Printf("failed to terminate postgres container: %v", err)
		}
	}
	if ts.RedisContainer != nil {
		if err := ts.RedisContainer.Terminate(ts.Ctx); err != nil {
			log.Printf("failed to terminate redis container: %v", err)
		}
	}
	if ts.CHContainer != nil {
		if err := ts.CHContainer.Terminate(ts.Ctx); err != nil {
			log.Printf("failed to terminate clickhouse container: %v", err)
		}
	}
}

// OpenStatsConn opens a ClickHouse connection to the testcontainer and applies
// the stats schema so the stats reader/writer test suites have their tables
// available. The connection string comes from the testcontainer (CHURL); we do
// not route through viper, whose CLICKHOUSE_HOST default would not resolve
// inside this environment.
func (ts *TestSuite) OpenStatsConn() (chdriver.Conn, error) {
	u, err := url.Parse(ts.CHURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse clickhouse url: %w", err)
	}

	auth := chdriver.Auth{Database: strings.TrimPrefix(u.Path, "/")}
	if u.User != nil {
		auth.Username = u.User.Username()
		auth.Password, _ = u.User.Password()
	}

	conn, err := chdriver.Open(&chdriver.Options{
		Addr: []string{u.Host},
		Auth: auth,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open clickhouse connection: %w", err)
	}

	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping clickhouse: %w", err)
	}

	if err := clickhousepkg.ApplySchema(context.Background(), conn, ts.Logger); err != nil {
		return nil, err
	}

	return conn, nil
}

// StatsTables lists every ClickHouse table used by the stats fixtures.
var StatsTables = []string{
	"transaction_events",
	"topup_events",
	"transfer_events",
	"withdraw_events",
	"saldo_events",
	"card_events",
	"merchant_events",
}

// TruncateStatsTables removes all rows from the stats tables so each suite run
// starts from a clean, deterministic state.
func TruncateStatsTables(ctx context.Context, conn chdriver.Conn) error {
	for _, table := range StatsTables {
		if err := conn.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE IF EXISTS %s", table)); err != nil {
			return fmt.Errorf("failed to truncate %s: %w", table, err)
		}
	}
	return nil
}
