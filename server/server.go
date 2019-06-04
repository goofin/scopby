package server

import (
	"context"

	"github.com/goofin/scopby/models"
)

type Server interface {
	CreateUser(ctx context.Context, name string, timezone int) error
	GetUser(ctx context.Context, name string) (*models.User, error)
	AddSnacks(ctx context.Context, name string, snacks int) error

	CreateMission(ctx context.Context, name string, desc string, seconds int, snacks int) error
	GetMissions(ctx context.Context, name string) ([]*models.Mission, error)
	CompleteMission(ctx context.Context, id int64) error
	DeleteMission(ctx context.Context, id int64) error
}
