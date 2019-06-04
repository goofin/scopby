package rest

import (
	"time"

	"github.com/goofin/scopby/models"
)

type User struct {
	Name     string `json:"name"`
	Snacks   int    `json:"snacks"`
	Timezone int    `json:"timezone"`
}

func jsonUser(user *models.User) *User {
	if user == nil {
		return nil
	}
	return &User{
		Name:     user.Name,
		Snacks:   user.Snacks,
		Timezone: user.Timezone,
	}
}

type Mission struct {
	Id           int64      `json:"id"`
	Description  string     `json:"description"`
	Seconds      int        `json:"seconds"`
	Due          string     `json:"due"`
	LastComplete *time.Time `json:"last_complete"`
}

func jsonMission(mission *models.Mission) *Mission {
	if mission == nil {
		return nil
	}
	return &Mission{
		Id:          mission.Id,
		Description: mission.Description,
		Seconds:     mission.Seconds,
		Due: new(time.Time).
			Add(time.Second * time.Duration(mission.Seconds)).
			Format("3:04PM"),
		LastComplete: mission.LastComplete,
	}
}

func jsonMissions(missions []*models.Mission) []*Mission {
	if missions == nil {
		return nil
	}
	out := make([]*Mission, len(missions))
	for i := range missions {
		out[i] = jsonMission(missions[i])
	}
	return out
}
