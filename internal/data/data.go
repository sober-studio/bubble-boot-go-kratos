package data

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/biz"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/conf"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/data/model"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/data/query"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/idgen"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/idgen/snowflake"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewRedis,
	NewIDGenerator,
	NewRedisCaptchaStore,
	wire.Bind(new(biz.Transaction), new(*Data)),
)

// Data .
// 注意：所有需要关闭的资源必须在 cleanup 中显式处理
type Data struct {
	db  *gorm.DB
	rdb *redis.Client
	//query *query.Query // GORM Gen 生成的查询对象
	//oss   biz.OSS
}

// NewData .
func NewData(c *conf.Data, logger log.Logger, db *gorm.DB, rdb *redis.Client) (*Data, func(), error) {
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
		// GORM 不需要手动关闭连接池（由 Go 标准库 sql.DB 管理）
		// 关闭 Redis 连接
		if err := rdb.Close(); err != nil {
			log.NewHelper(logger).Error(err)
		}
	}

	return &Data{
		db:  db,
		rdb: rdb,
		//query: query.Use(db), // 初始化 GORM Gen 查询实例
		//oss:   oss,
	}, cleanup, nil
}

// NewDB 初始化数据库 (GORM)
func NewDB(c *conf.Data, l log.Logger) *gorm.DB {
	var dialector gorm.Dialector
	switch c.Database.Driver {
	case "postgres":
		dialector = postgres.Open(c.Database.Source)
	case "mysql":
		// 预留 MySQL 适配
		dialector = mysql.Open(c.Database.Source)
	default:
		// 默认使用 Postgres
		dialector = postgres.Open(c.Database.Source)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		// 使用自定义的 Kratos 日志适配器 (前面步骤中定义的 NewGormLogger)
		Logger: NewGormLogger(l),
	})
	if err != nil {
		log.NewHelper(l).Fatalf("failed opening connection to database: %v", err)
	}

	// 初始化完成后调用 AutoMigrate
	// if err := db.AutoMigrate(); err != nil {
	// 	log.NewHelper(l).Error(err)
	// }

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		log.NewHelper(l).Fatalf("failed to get sql DB from gorm: %v", err)
	}

	// 设置最大空闲连接数
	if c.Database.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(int(c.Database.MaxIdleConns))
	} else {
		sqlDB.SetMaxIdleConns(10) // 默认值
	}

	// 设置最大打开连接数
	if c.Database.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(int(c.Database.MaxOpenConns))
	} else {
		sqlDB.SetMaxOpenConns(100) // 默认值
	}

	// 设置连接最大生命周期
	if c.Database.ConnMaxLifetime != nil {
		sqlDB.SetConnMaxLifetime(c.Database.ConnMaxLifetime.AsDuration())
	} else {
		sqlDB.SetConnMaxLifetime(time.Hour) // 默认值
	}

	return db
}

// NewRedis 初始化 Redis 客户端
func NewRedis(c *conf.Data, l log.Logger) *redis.Client {
	var readTimeout, writeTimeout time.Duration
	if c.Redis.ReadTimeout != nil {
		readTimeout = c.Redis.ReadTimeout.AsDuration()
	}
	if c.Redis.WriteTimeout != nil {
		writeTimeout = c.Redis.WriteTimeout.AsDuration()
	}

	poolSize := 10
	if c.Redis.PoolSize > 0 {
		poolSize = int(c.Redis.PoolSize)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         c.Redis.Addr,
		Password:     c.Redis.Password,
		DB:           int(c.Redis.Database),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		PoolSize:     poolSize,
	})

	// 连通性检查与重试
	ctx := context.Background()
	helper := log.NewHelper(l)

	// 简单重试 3 次
	for i := 0; i < 3; i++ {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		if err := rdb.Ping(pingCtx).Err(); err == nil {
			cancel()
			return rdb
		}
		cancel()
		helper.Infof("failed connecting to redis, retrying... (%d/3)", i+1)
		time.Sleep(1 * time.Second)
	}

	// 最后一次尝试，如果失败则 Fatal
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		helper.Fatalf("failed connecting to redis: %v", err)
	}

	return rdb
}

// contextTxKey 事务在 Context 中的 Key
type contextTxKey struct{}

// InTx 事务包装器实现 (biz.Transaction 接口)
func (d *Data) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 将事务对象注入 Context
		ctx = context.WithValue(ctx, contextTxKey{}, tx)
		return fn(ctx)
	})
}

// getDB 内部私有方法：处理事务判断
func (d *Data) getDB(ctx context.Context) *gorm.DB {
	tx, ok := ctx.Value(contextTxKey{}).(*gorm.DB)
	if ok {
		return tx
	}
	return d.db.WithContext(ctx)
}

// DB 返回原始 GORM 实例（支持事务）
func (d *Data) DB(ctx context.Context) *gorm.DB {
	return d.getDB(ctx)
}

// Q 返回 GORM Gen 类型安全查询实例（支持事务）
func (d *Data) Q(ctx context.Context) *query.Query {
	db := d.db
	tx, ok := ctx.Value(contextTxKey{}).(*gorm.DB)
	if ok {
		db = tx
	}
	return query.Use(db.WithContext(ctx))
}

// RDB 返回 Redis 客户端
func (d *Data) RDB() *redis.Client {
	return d.rdb
}

// NewIDGenerator 初始化 ID 生成器
func NewIDGenerator(app *conf.Application) idgen.IDGenerator {
	g := snowflake.NewSnowflake(app.WorkerId)
	model.SetIDGenerator(g)
	return g
}
