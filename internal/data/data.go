package data

import (
	"context"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"time"
	"users/internal/conf"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.opentelemetry.io/otel"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	gormlogger "gorm.io/gorm/logger"
)

var ProviderSet = wire.NewSet(NewData, NewUsersRepo)

type Data struct {
	// TODO wrapped database client
	client *gorm.DB
}

func openDB(c *conf.Data, logger log.Logger) (*gorm.DB, error) {
	dsn := c.Database.Source
	if dsn == "" {
		return nil, errors.InternalServer("data.openDB", "missing database source")
	}
	driver := c.Database.Driver
	if driver == "" {
		return nil, errors.InternalServer("data.openDB", "missing database driver")
	}

	lg := NewZapGormAdapter(logger)

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // disables implicit prepared statement usage

	}), &gorm.Config{
		Logger: lg,
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	client, err := openDB(c, logger)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
		if err != nil {
			log.NewHelper(logger).Error("failed closing the data resources: %v", err)
		}
	}
	return &Data{client}, cleanup, nil
}

func Migrate(ctx context.Context, c *conf.Data, logger log.Logger) {
	_, span := otel.Tracer("data").Start(ctx, "Migrate")
	defer span.End()
	log.NewHelper(logger).Info("migrating the schema")
	client, err := openDB(c, logger)
	if err != nil {
		log.NewHelper(logger).Error("failed opening database: %v", err)
	}
	err = client.AutoMigrate(&Users{})
	if err != nil {
		log.NewHelper(logger).Error("failed migrating the schema: %v", err)

	}
}

type adaptedGormLogger struct {
	logger *log.Helper
}

func NewZapGormAdapter(logger log.Logger) gormlogger.Interface {
	lg := log.NewHelper(logger)
	return &adaptedGormLogger{lg}
}

func (l *adaptedGormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return l
}

func (l *adaptedGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	args := make([]interface{}, 0, len(data)+1)
	args = append(args, msg)
	args = append(args, data...)
	l.logger.Info(args...)
}

func (l *adaptedGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	args := make([]interface{}, 0, len(data)+1)
	args = append(args, msg)
	args = append(args, data...)
	l.logger.Warn(args...)
}

func (l *adaptedGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	args := make([]interface{}, 0, len(data)+1)
	args = append(args, msg)
	args = append(args, data...)
	l.logger.Error(args...)
}

func (l *adaptedGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	_, span := otel.Tracer("gorm").Start(ctx, "Query")
	defer span.End()
	sql, rowsAffected := fc()
	span.SetAttributes(attribute.String("db.system", "postgresql"))
	span.SetAttributes(attribute.String("db.statement", sql))
	span.SetAttributes(attribute.Int64("db.rows_affected", int64(rowsAffected)))
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
}
