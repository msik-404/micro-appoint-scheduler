package main

import (
	"context"
	"fmt"
	"net"
	"time"

    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
    "google.golang.org/grpc"

	"github.com/msik-404/micro-appoint-scheduler/internal/database"
	mygrpc "github.com/msik-404/micro-appoint-scheduler/internal/grpc"
	"github.com/msik-404/micro-appoint-scheduler/internal/models"
	"github.com/msik-404/micro-appoint-scheduler/internal/schedulerpb"
	"github.com/msik-404/micro-appoint-scheduler/internal/scheduling"
)

func main() {
	_, err := mygrpc.New()
	if err != nil {
		panic(err)
	}
	mongoClient, err := database.ConnectDB()
	if err != nil {
		panic(err)
	}
	_, err = database.CreateDBIndexes(mongoClient)
	if err != nil {
		panic(err)
	}
	port := 50051
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}
	s := grpc.NewServer()
	schedulerpb.RegisterApiServer(s, &schedulerpb.Server{Client: mongoClient})
	if err := s.Serve(lis); err != nil {
		panic(err)
	}
}

func test(conns *mygrpc.GRPCConns, mongoClient *mongo.Client) {
	companyIDHex := "64984e94f6a5569ad78b9c18"
	serviceIDHex := "64984f4bf6a5569ad78b9c19"

	companyID, _ := primitive.ObjectIDFromHex(companyIDHex)
	serviceID, _ := primitive.ObjectIDFromHex(serviceIDHex)
	employeeID, _ := primitive.ObjectIDFromHex("649850b5d01e68201e68ca00")
	nullDate := time.Date(2023, 6, 26, 00, 00, 00, 00, time.UTC)
	order := models.Order{
		UserID:     primitive.NewObjectID(),
		CompanyID:  companyID,
		ServiceID:  serviceID,
		EmployeeID: employeeID,
		OrderTime:  primitive.NewDateTimeFromTime(time.Now().UTC()),
		IsCanceled: false,
		StartTime:  primitive.NewDateTimeFromTime(nullDate.Add(time.Minute * 830)),
		EndTime:    primitive.NewDateTimeFromTime(nullDate.Add(time.Minute * 880)),
	}
	_, err := order.InsertOneOrder(context.Background(), mongoClient)
	if err != nil {
	}

	results, err := scheduling.GetAllAvaliableTimesSlots(
		mongoClient,
		conns,
		companyIDHex,
		serviceIDHex,
		50,
		nullDate,
		nil,
		nil,
	)
	if err != nil {
		panic(err)
	}
	for _, i := range results {
		fmt.Printf("dostępne daty: %+v\n", i)
	}
}
