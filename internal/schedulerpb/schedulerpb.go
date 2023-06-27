package schedulerpb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	mygrpc "github.com/msik-404/micro-appoint-scheduler/internal/grpc"
	"github.com/msik-404/micro-appoint-scheduler/internal/models"
)

type Server struct {
	UnimplementedApiServer
	Client *mongo.Client
	Conns  *mygrpc.GRPCConns
}

func (s *Server) AddOrder(
	ctx context.Context,
	request *AddOrderRequest,
) (*emptypb.Empty, error) {
	customerID, err := primitive.ObjectIDFromHex(request.GetCustomerId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	companyID, err := primitive.ObjectIDFromHex(request.GetCompanyId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	serviceID, err := primitive.ObjectIDFromHex(request.GetServiceId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	employeeID, err := primitive.ObjectIDFromHex(request.GetEmployeeId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if request.StartTime == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"Start time field is required, provide valid unix time",
		)
	}
	startTime := time.Unix(request.GetStartTime(), 0)
	if request.EndTime == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"End time field is required, provide valid unix time",
		)
	}
	endTime := time.Unix(request.GetStartTime(), 0)
	order := models.Order{
		CustomerID:     customerID,
		CompanyID:  companyID,
		ServiceID:  serviceID,
		EmployeeID: employeeID,
		OrderTime:  primitive.NewDateTimeFromTime(time.Now().UTC()),
		IsCanceled: false,
		StartTime:  primitive.NewDateTimeFromTime(startTime),
		EndTime:    primitive.NewDateTimeFromTime(endTime),
	}
	_, err = order.InsertOneOrder(ctx, s.Client)
	if err != nil {
        if err == models.BookedError {
            return nil, status.Error(codes.AlreadyExists, err.Error())
        }
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) FindManyOrders(
	ctx context.Context,
	request *OrdersRequest,
) (*OrdersReply, error) {
	initUserID, userErr := primitive.ObjectIDFromHex(request.GetCustomerId())
	initCompanyID, companyErr := primitive.ObjectIDFromHex(request.GetCompanyId())
	if userErr != nil && companyErr != nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"UserID filed or comapnyID field should have proper hex value",
		)
	} else if userErr == nil && companyErr == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"Only UserID field or companyID field should be filled at the same time",
		)
	}
	var customerID, companyID *primitive.ObjectID
	if userErr == nil {
		customerID = &initUserID
	} else if companyID == nil {
		companyID = &initCompanyID
	}
	var startDate *primitive.DateTime
	if request.StartDate != nil {
		date := primitive.NewDateTimeFromTime(time.Unix(request.GetStartDate(), 0))
		startDate = &date
	}
	var startValue *primitive.ObjectID
	if request.StartValue != nil {
		id, err := primitive.ObjectIDFromHex(request.GetStartValue())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		startValue = &id
	}
	cursor, err := models.FindManyOrders(
		ctx,
		s.Client,
		customerID,
		companyID,
		request.NPerPage,
		startDate,
		startValue,
		request.IsCanceled,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	defer cursor.Close(ctx)
	reply := OrdersReply{}
	for cursor.Next(ctx) {
		var order models.Order
		if err := cursor.Decode(&order); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		id := order.ID.Hex()
		serviceID := order.ServiceID.Hex()
		employeeID := order.EmployeeID.Hex()
		orderTime := order.OrderTime.Time().Unix()
		orderProto := Order{
			Id:         &id,
			CustomerId:     nil,
			CompanyId:  nil,
			ServiceId:  &serviceID,
			EmployeeId: &employeeID,
			OrderTime:  &orderTime,
			IsCanceled: &order.IsCanceled,
		}
		if !order.CustomerID.IsZero() {
			customerID := order.CustomerID.Hex()
			orderProto.CustomerId = &customerID
		} else if !order.CompanyID.IsZero() {
			companyID := order.CompanyID.Hex()
			orderProto.CompanyId = &companyID
		}
		reply.Orders = append(reply.Orders, &orderProto)
	}
	return &reply, nil
}

func (s *Server) CancelOrder(
	ctx context.Context,
	request *CancelRequest,
) (*emptypb.Empty, error) {
	customerID, err := primitive.ObjectIDFromHex(request.GetCustomerId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	id, err := primitive.ObjectIDFromHex(request.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	result, err := models.CancelOneOrder(ctx, s.Client, customerID, id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if result.MatchedCount == 0 {
		return nil, status.Error(
			codes.NotFound,
			"Order with that customerID and ID was not found",
		)
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) FindManyAvaliableTimeSlots(
	ctx context.Context,
	request *AvaliableTimeSlotsRequest,
) (*AvaliableTimeSlotsReply, error) {
	reply, err := GetAllAvaliableTimesSlots(ctx, s.Client, s.Conns, request)
	if err != nil {
		return nil, err
	}
	return reply, nil
}
