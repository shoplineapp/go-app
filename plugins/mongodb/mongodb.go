//go:build mongodb
// +build mongodb

package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/shoplineapp/go-app/plugins"
	"github.com/shoplineapp/go-app/plugins/env"
	"github.com/shoplineapp/go-app/plugins/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewMongoStore)
}

var MONGODB_CONNECTION_TIMEOUT = 10 * time.Second
var MONGODB_PING_INTERNAL = 30 * time.Second
var MONGODB_QUERY_TIMEOUT = 20 * time.Second

type MongoStore struct {
	env    *env.Env
	logger *logger.Logger

	client   *mongo.Client
	database *mongo.Database
}

func generateConnectURL(protocol string, username string, password string, hosts string, databaseName string, params string) string {
	var paramsString string
	var credentials string

	if protocol == "" {
		protocol = "mongodb"
	}

	if params != "" {
		paramsString = fmt.Sprintf("?%s", params)
	} else {
		paramsString = ""
	}

	if username != "" && password != "" {
		credentials = fmt.Sprintf("%s:%s@", username, password)
	} else {
		credentials = ""
	}

	return fmt.Sprintf("%s://%s%s/%s%s", protocol, credentials, hosts, databaseName, paramsString)
}

func (s *MongoStore) Connect(protocol string, username string, password string, hosts string, databaseName string, params string) {
	connectURL := generateConnectURL(protocol, username, password, hosts, databaseName, params)

	ctx, cancel := context.WithTimeout(context.Background(), MONGODB_CONNECTION_TIMEOUT)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connectURL))

	if err != nil {
		s.logger.Error(err)
		panic(err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), MONGODB_PING_INTERNAL)
	defer cancel()

	err = client.Ping(ctx, readpref.Primary())

	if err != nil {
		s.logger.Error(err)
		panic(err)
	}

	database := client.Database(databaseName)

	s.logger.Info(fmt.Sprintf("MongoDB connection with %s established", hosts))
	s.client = client
	s.database = database
}

func (s MongoStore) FindOne(collectionName string, filter interface{}, doc interface{}) error {
	collection := s.database.Collection(collectionName)
	ctx, cancel := context.WithTimeout(context.Background(), MONGODB_QUERY_TIMEOUT)
	defer cancel()
	result := collection.FindOne(ctx, filter)

	if err := result.Err(); err == mongo.ErrNoDocuments {
		return err
	}

	err := result.Decode(doc)
	if err != nil {
		return err
	}

	return nil
}

func NewMongoStore(env *env.Env, logger *logger.Logger) *MongoStore {
	store := &MongoStore{
		env:    env,
		logger: logger,
	}
	return store
}
