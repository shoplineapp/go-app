//go:build mongodb
// +build mongodb

package mongodb

import (
	"fmt"
	"time"

	"github.com/shoplineapp/go-app/plugins"
	"github.com/shoplineapp/go-app/plugins/env"
	"github.com/shoplineapp/go-app/plugins/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/net/context"

	"github.com/kamva/mgm/v3"
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
	client *mongo.Client
	db     *mongo.Database
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

func (s MongoStore) Collection(name string) *mgm.Collection {
	return mgm.NewCollection(s.db, name)
}

func (s *MongoStore) Connect(protocol string, username string, password string, hosts string, databaseName string, params string, opts ...*options.ClientOptions) {
	connectURL := generateConnectURL(protocol, username, password, hosts, databaseName, params)

	opts = append(opts, options.Client().ApplyURI(connectURL))
	opts = append(opts, options.Client().SetConnectTimeout(MONGODB_CONNECTION_TIMEOUT))
	opts = append(opts, options.Client().SetTimeout(MONGODB_QUERY_TIMEOUT))

	ctx := context.Background()
	client, err := mongo.Connect(ctx, opts...)
	if err != nil {
		panic(err)
	}

	ctxWithCancel, cancel := context.WithTimeout(ctx, MONGODB_CONNECTION_TIMEOUT)
	defer cancel()

	// The Client.Ping method can be used to verify that the deployment is successfully connected and
	// the Client was correctly configured.
	err = client.Ping(ctxWithCancel, nil)
	if err != nil {
		panic(err)
	}

	s.client = client
	s.db = client.Database(databaseName)
}

func NewMongoStore(env *env.Env, logger *logger.Logger) *MongoStore {

	store := &MongoStore{
		env:    env,
		logger: logger,
	}
	return store
}
