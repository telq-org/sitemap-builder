package mongo

import (
	"context"
	"github.com/telq-org/sitemap-builder/pkg/config"
	"github.com/telq-org/sitemap-builder/pkg/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"time"
)

var (
	Threads     *mongo.Collection
	Users       *mongo.Collection
	Tags        *mongo.Collection
	Communities *mongo.Collection
)

func init() {
	const timeout = 10
	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().
		SetWriteConcern(writeconcern.New(
			writeconcern.W(1),
			writeconcern.J(true),
		)).
		SetReadConcern(readconcern.Available()).
		SetReadPreference(readpref.SecondaryPreferred()).
		ApplyURI(config.Env.MongoDB.URL))
	logger.Must(err)

	err = client.Ping(ctx, nil)
	logger.Must(err)

	Threads = client.Database("telq_backend").Collection("threads")
	Users = client.Database("telq_backend").Collection("users")
	Tags = client.Database("telq_backend").Collection("tags")
	Communities = client.Database("telq_backend").Collection("communities")
}
