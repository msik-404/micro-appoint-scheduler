package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var DBName = os.Getenv("DB_NAME")

const CollName string = "orders"

func getURI() string {
	return fmt.Sprintf("mongodb://%s:%s@scheduler-db:27017", os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"))
}

func ConnectDB() (*mongo.Client, error) {
	// Use the SetServerAPIOptions() method to set the Stable API version to 1
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(getURI()).SetServerAPIOptions(serverAPI)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	// Create a new client and connect to the server
	return mongo.Connect(ctx, opts)
}

func CreateDBIndexes(client *mongo.Client) ([]string, error) {
	db := client.Database(DBName)
	coll := db.Collection(CollName)
	index := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "employee_id", Value: 1},
				{Key: "start_time", Value: 1},
				{Key: "end_time", Value: -1},
				{Key: "is_canceled", Value: 1},
			},
		},
        {
            Keys: bson.D{
                {Key: "_id", Value: -1},
                {Key: "customer_id", Value: -1},
                {Key: "order_time", Value: -1},
            },
        },
        {
            Keys: bson.D{
                {Key: "_id", Value: -1},
                {Key: "company_id", Value: -1},
                {Key: "order_time", Value: -1},
            },
        },
        {
            Keys: bson.D{
                {Key: "_id", Value: -1},
                {Key: "customer_id", Value: -1},
                {Key: "order_time", Value: -1},
				{Key: "is_canceled", Value: 1},
            },
        },
        {
            Keys: bson.D{
                {Key: "_id", Value: -1},
                {Key: "company_id", Value: -1},
                {Key: "order_time", Value: -1},
				{Key: "is_canceled", Value: 1},
            },
        },
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return coll.Indexes().CreateMany(ctx, index)
}
