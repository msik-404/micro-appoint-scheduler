package models

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/msik-404/micro-appoint-scheduler/internal/database"
)

type Order struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	CustomerID primitive.ObjectID `bson:"customer_id,omitempty"`
	CompanyID  primitive.ObjectID `bson:"company_id,omitempty"`
	ServiceID  primitive.ObjectID `bson:"service_id,omitempty"`
	EmployeeID primitive.ObjectID `bson:"employee_id,omitempty"`
	OrderTime  primitive.DateTime `bson:"order_time,omitempty"`
	IsCanceled bool               `bson:"is_canceled"`
	StartTime  primitive.DateTime `bson:"start_time,omitempty"`
	EndTime    primitive.DateTime `bson:"end_time,omitempty"`
}

// todo needs fixing
func FindManyOrders(
	ctx context.Context,
	client *mongo.Client,
	customerID *primitive.ObjectID,
	companyID *primitive.ObjectID,
	nPerPage *int64,
	startDate *primitive.DateTime,
	startValue *primitive.ObjectID,
	isCanceled *bool,
) (*mongo.Cursor, error) {
	db := client.Database(database.DBName)
	coll := db.Collection(database.CollName)

	opts := options.Find()
	var n int64 = 30
	if nPerPage != nil {
		if *nPerPage > 0 {
			n = *nPerPage
		}
	}
	opts.SetLimit(n)
	projection := bson.D{
		{Key: "start_time", Value: 0},
		{Key: "end_time", Value: 0},
	}
	if customerID != nil {
		projection = append(projection, bson.E{Key: "customer_id", Value: 0})
	} else if companyID != nil {
		projection = append(projection, bson.E{Key: "company_id", Value: 0})
	}
	opts.SetProjection(projection)

	filter := bson.M{}

	if isCanceled != nil {
		filter["is_canceled"] = *isCanceled
	}

	if customerID != nil {
		filter["customer_id"] = customerID
	} else if companyID != nil {
		filter["company_id"] = companyID
	}
	if startValue != nil {
		opts.SetSort(bson.M{
			"order_id":   -1,
			"order_time": -1,
		})
		filter["order_id"] = bson.M{"lt": startValue}
		filter["order_time"] = bson.M{"lte": startDate}
	}
	return coll.Find(ctx, filter, opts)
}

var BookedError = errors.New("this Date is already booked")

func IsBookedTimeFrame(
	ctx context.Context,
	client *mongo.Client,
	employeeID primitive.ObjectID,
	startTime primitive.DateTime,
	endTime primitive.DateTime,
) error {
	db := client.Database(database.DBName)
	coll := db.Collection(database.CollName)
	matches, err := coll.CountDocuments(ctx, bson.M{
		"employee_id": employeeID,
		"start_time":  bson.M{"$lt": endTime},
		"end_time":    bson.M{"$gt": startTime},
		"is_canceled": false,
	})
	if err != nil {
		return err
	}
	if matches > 0 {
		return BookedError
	}
	return nil
}

// todo: check if this employee can perfom this service and
// whether this service and employee is in this company
func (order *Order) InsertOneOrder(
	ctx context.Context,
	client *mongo.Client,
) (*mongo.InsertOneResult, error) {
	session, err := client.StartSession()
	if err != nil {
		return nil, err
	}
	defer session.EndSession(ctx)
	db := client.Database(database.DBName)
	coll := db.Collection(database.CollName)
	callback := func(mongoCtx mongo.SessionContext) (any, error) {
		err := IsBookedTimeFrame(
			ctx,
			client,
			order.EmployeeID,
			order.StartTime,
			order.EndTime,
		)
		if err != nil {
			return nil, err
		}
		return coll.InsertOne(ctx, order)
	}
	transactionResult, err := session.WithTransaction(ctx, callback)
	if err != nil {
		return nil, err
	}
	result := transactionResult.(*mongo.InsertOneResult)
	return result, nil
}

func CancelOneOrder(
	ctx context.Context,
	client *mongo.Client,
	customerID primitive.ObjectID,
	id primitive.ObjectID,
) (*mongo.UpdateResult, error) {
	db := client.Database(database.DBName)
	coll := db.Collection(database.CollName)
	filter := bson.M{
		"_id":     id,
		"customer_id": customerID,
	}
	update := bson.M{"$set": bson.M{"is_canceled": true}}
	return coll.UpdateOne(ctx, filter, update)
}
