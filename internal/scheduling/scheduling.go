package scheduling

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/msik-404/micro-appoint-scheduler/internal/grpc"
	"github.com/msik-404/micro-appoint-scheduler/internal/grpc/employees/employeespb"
	"github.com/msik-404/micro-appoint-scheduler/internal/models"
)

const CoroutinesAmount int = 4

func SplitWork[T any](collection []T, workerAmount int) [][]T {
	itemsPerWorker := len(collection) / workerAmount
	slices := make([][]T, 0, workerAmount)
	for i := 0; i < workerAmount; i++ {
		start := i * itemsPerWorker
		end := (i + 1) * itemsPerWorker
		if i == CoroutinesAmount-1 {
			end = len(collection)
		}
		slices = append(slices, collection[start:end])
	}
	return slices
}

func toIsoWeekDay(goWeekDay time.Weekday) int32 {
	switch goWeekDay {
	case 0:
		return 6
	default:
		return int32(goWeekDay) - 1
	}
}

func GetAllTimeSlots(
	ctx context.Context,
	conns *grpc.GRPCConns,
	request *employeespb.TimeSlotsRequest,
) (*employeespb.TimeSlotsReply, error) {
	client := employeespb.NewApiClient(conns.GetEmployeesConn())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return client.FindManyTimeSlots(ctx, request)
}

type TimeFrame struct {
	From int32
	To   int32
}

type EmployeeTimeSlot struct {
	ID        string
	Name      string
	Surname   string
	TimeSlots []TimeFrame
}

func checkAvability(
	ctx context.Context,
	client *mongo.Client,
	date time.Time,
	employeeTimeSlot *employeespb.EmployeeTimeSlots,
) (*EmployeeTimeSlot, error) {
	if employeeTimeSlot.EmployeeInfo == nil {
		return nil, errors.New("EmployeeInfo field should be set")
	}
	employeeInfo := employeeTimeSlot.EmployeeInfo
	if employeeInfo.Id == nil {
		return nil, errors.New("EmployeeID field must be set")
	}
	employeeAvaliableTimeSlots := EmployeeTimeSlot{
		ID:      employeeInfo.GetId(),
		Name:    employeeInfo.GetName(),
		Surname: employeeInfo.GetSurname(),
	}
	for _, timeSlot := range employeeTimeSlot.TimeSlots {
		employeeID, err := primitive.ObjectIDFromHex(employeeInfo.GetId())
		if err != nil {
			return nil, errors.New("EmployeeID must be set to proper objectID hex")
		}
		if timeSlot.From == nil || timeSlot.To == nil {
			return nil, errors.New("Both TimeSlot values should be set")
		}
		var fromTime, toTime int32 = *timeSlot.From, *timeSlot.To
		if fromTime >= toTime {
			return nil, errors.New("from time must be smaller than to time")
		}
		startTime := date.Add(time.Minute * time.Duration(fromTime))
		endTime := date.Add(time.Minute * time.Duration(toTime))
		isBooked, err := models.IsBookedTimeFrame(
			ctx,
			client,
			employeeID,
			primitive.NewDateTimeFromTime(startTime),
			primitive.NewDateTimeFromTime(endTime),
		)
		if err != nil {
			return nil, err
		}
		if isBooked == false {
			avaliableTimeSlot := TimeFrame{
				From: fromTime,
				To:   toTime,
			}
			employeeAvaliableTimeSlots.TimeSlots = append(
				employeeAvaliableTimeSlots.TimeSlots,
				avaliableTimeSlot,
			)
		}
	}
	return &employeeAvaliableTimeSlots, nil
}

func callback(
	ctx context.Context,
	errs chan error,
	ch chan []*EmployeeTimeSlot,
	client *mongo.Client,
	date time.Time,
	inputSlice []*employeespb.EmployeeTimeSlots,
) {
	outputSlice := make([]*EmployeeTimeSlot, 0, len(inputSlice))
	for _, employeeTimeSlot := range inputSlice {
		avaliableTimeSlot, err := checkAvability(ctx, client, date, employeeTimeSlot)
		if err != nil {
			errs <- err
			ch <- outputSlice
			return
		}
		outputSlice = append(outputSlice, avaliableTimeSlot)
	}
	errs <- nil
	ch <- outputSlice
}

func GetAllAvaliableTimesSlots(
	client *mongo.Client,
	conns *grpc.GRPCConns,
	companyID string,
	serviceID string,
	serviceDuration int32,
	date time.Time,
	startValue *string,
	nPerPage *int64,
) ([]*EmployeeTimeSlot, error) {
	weekDay := toIsoWeekDay(date.Weekday())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	message := employeespb.TimeSlotsRequest{
		CompanyId:       &companyID,
		ServiceId:       &serviceID,
		ServiceDuration: &serviceDuration,
		Day:             &weekDay,
		StartValue:      startValue,
		NPerPage:        nPerPage,
	}
	reply, err := GetAllTimeSlots(ctx, conns, &message)
	if err != nil {
		return nil, err
	}
	employeeTimeSlots := reply.GetEmployeeTimeSlots()
	workerAmount := CoroutinesAmount
	if len(employeeTimeSlots) < CoroutinesAmount {
		workerAmount = len(employeeTimeSlots)
	}
	slices := SplitWork(employeeTimeSlots, workerAmount)
	// this will not be a deadlock, because every goroutine will finish
	errsChan := make(chan error)
	resultsChan := make(chan []*EmployeeTimeSlot)
	for _, slice := range slices {
		go callback(ctx, errsChan, resultsChan, client, date, slice)
	}
	var errs []error
	var avaliableTimeSlot []*EmployeeTimeSlot
	for range slices {
		err := <-errsChan
		if err != nil {
			errs = append(errs, err)
		}
		timeSlots := <-resultsChan
		for _, timeSlot := range timeSlots {
			avaliableTimeSlot = append(avaliableTimeSlot, timeSlot)
		}
	}
	// if errs is not empty
	for _, err = range errs {
		return nil, err
	}
	return avaliableTimeSlot, nil
}
