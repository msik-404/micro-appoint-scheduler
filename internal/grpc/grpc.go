package grpc

import (
    "google.golang.org/grpc"

	"github.com/msik-404/micro-appoint-scheduler/internal/grpc/employees"
)

type GRPCConns struct {
	EmployeeSConn *grpc.ClientConn
}

func New() (*GRPCConns, error) {
	conns := GRPCConns{}
	employeesConn, err := grpc.Dial(employees.ConnString, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	conns.EmployeeSConn = employeesConn
	return &conns, nil
}

func (conns *GRPCConns) GetEmployeesConn() *grpc.ClientConn {
	return conns.EmployeeSConn
}
