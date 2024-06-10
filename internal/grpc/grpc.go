package grpc

import (
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/state"
)

type Service struct {
	requests     *state.Requests
	environments *state.Environments
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) GetServerReflection(id string) ([]domain.GRPCMethod, error) {
	return []domain.GRPCMethod{
		{
			Name: "Echo",
		},
		{
			Name: "CreateUser",
		},
	}, nil
}
