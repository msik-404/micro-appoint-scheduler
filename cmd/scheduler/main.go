package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc"

	"github.com/msik-404/micro-appoint-scheduler/internal/database"
	mygrpc "github.com/msik-404/micro-appoint-scheduler/internal/grpc"
	"github.com/msik-404/micro-appoint-scheduler/internal/models"
	"github.com/msik-404/micro-appoint-scheduler/internal/schedulerpb"
)

func main() {
	conns, err := mygrpc.New()
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
	// grpc server runs on different goroutine
	go func() {
		port := 50051
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			panic(err)
		}
		s := grpc.NewServer()
		schedulerpb.RegisterApiServer(
			s,
			&schedulerpb.Server{Client: mongoClient, Conns: conns},
		)
		if err := s.Serve(lis); err != nil {
			panic(err)
		}
	}()
	// run rabbitmq listener server on main goroutine
	conn, err := amqp.Dial("amqp://guest:guest@rabbit:5672/")
	if err != nil {
		panic(err)
	}
	ch, err := conn.Channel()
	requestQueue, err := ch.QueueDeclare(
		"request-cancel", // name
		false,            // durable
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		panic(err)
	}
	msgs, err := ch.Consume(
		requestQueue.Name, // queue
		"",                // consumer
		false,             // auto-ack
		false,             // exclusive
		false,             // no-local
		false,             // no-wait
		nil,               // args
	)
	// wait here for new messages FOREVER
	for msg := range msgs {
		var request models.CancelRequest
		err := json.Unmarshal(msg.Body, &request)
		if err != nil {
            log.Print(err.Error())
            continue
		}
        customerID, err := primitive.ObjectIDFromHex(request.CustomerID)
		if err != nil {
            log.Print(err.Error())
            continue
		}
        orderID, err := primitive.ObjectIDFromHex(request.OrderID)
		if err != nil {
            log.Print(err.Error())
            continue
		}
        _, err = models.CancelOneOrder(
            context.Background(), 
            mongoClient, 
            customerID, 
            orderID,
        )
		if err != nil {
            log.Print(err.Error())
            continue
		}
		msg.Ack(false)
	}
}
