package db

import (
	"context"
	"time"

	"github.com/haroldcamargo/english/backend/internal/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Connect(ctx context.Context, cfg config.Config) (*mongo.Client, *mongo.Database, error) {
	clientOpts := options.Client().ApplyURI(cfg.MongoURI)
	clientOpts.SetMaxPoolSize(128)
	clientOpts.SetMinPoolSize(8)
	clientOpts.SetMaxConnecting(16)
	clientOpts.SetRetryWrites(true)
	clientOpts.SetConnectTimeout(5 * time.Second)
	clientOpts.SetServerSelectionTimeout(5 * time.Second)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, nil, err
	}
	return client, client.Database(cfg.MongoDB), nil
}
