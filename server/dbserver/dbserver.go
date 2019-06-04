package dbserver

import (
	"context"
	"time"

	"github.com/goofin/scopby/models"
	"github.com/zeebo/errs"
	"github.com/zeebo/mon"
)

var Err = errs.Class("dbserver")

type Server struct {
	db *models.DB
}

func New(db *models.DB) *Server {
	return &Server{db: db}
}

func (s *Server) CreateUser(ctx context.Context, name string, timezone int) (err error) {
	defer mon.Start().Stop(&err)

	return Err.Wrap(s.db.CreateNoReturn_User(ctx,
		models.User_Name(name),
		models.User_Timezone(timezone)))
}

func (s *Server) GetUser(ctx context.Context, name string) (
	user *models.User, err error) {
	defer mon.Start().Stop(&err)

	user, err = s.db.Get_User_By_Name(ctx,
		models.User_Name(name))
	return user, Err.Wrap(err)
}

func (s *Server) AddSnacks(ctx context.Context, name string, snacks int) (
	err error) {
	defer mon.Start().Stop(&err)

	rx := s.db.NewRx()
	defer func() {
		if err == nil {
			err = rx.Commit()
		} else {
			rx.Rollback()
		}
	}()

	user, err := rx.Get_User_By_Name(ctx,
		models.User_Name(name))
	if err != nil {
		return Err.Wrap(err)
	}

	return Err.Wrap(rx.UpdateNoReturn_User_By_Name(ctx,
		models.User_Name(name),
		models.User_Update_Fields{
			Snacks: models.User_Snacks(user.Snacks + snacks),
		}))
}

func (s *Server) CreateMission(ctx context.Context, name string, desc string,
	seconds int, snacks int) (err error) {
	defer mon.Start().Stop(&err)

	return Err.Wrap(s.db.CreateNoReturn_Mission(ctx,
		models.Mission_User(name),
		models.Mission_Description(desc),
		models.Mission_Seconds(seconds),
		models.Mission_Snacks(snacks),
		models.Mission_Create_Fields{}))
}

func (s *Server) GetMissions(ctx context.Context, name string) (
	missions []*models.Mission, err error) {
	defer mon.Start().Stop(&err)

	missions, err = s.db.All_Mission_By_User(ctx,
		models.Mission_User(name))
	return missions, Err.Wrap(err)
}

func (s *Server) CompleteMission(ctx context.Context, id int64) (err error) {
	defer mon.Start().Stop(&err)

	rx := s.db.NewRx()
	defer func() {
		if err == nil {
			err = rx.Commit()
		} else {
			rx.Rollback()
		}
	}()

	mission, err := rx.Get_Mission_By_Id(ctx,
		models.Mission_Id(id))
	if err != nil {
		return Err.Wrap(err)
	}

	user, err := rx.Get_User_By_Name(ctx,
		models.User_Name(mission.User))
	if err != nil {
		return Err.Wrap(err)
	}

	err = rx.UpdateNoReturn_User_By_Name(ctx,
		models.User_Name(mission.User),
		models.User_Update_Fields{
			Snacks: models.User_Snacks(user.Snacks + mission.Snacks),
		})
	if err != nil {
		return Err.Wrap(err)
	}

	err = rx.UpdateNoReturn_Mission_By_Id(ctx,
		models.Mission_Id(id),
		models.Mission_Update_Fields{
			LastComplete: models.Mission_LastComplete(time.Now()),
		})
	if err != nil {
		return Err.Wrap(err)
	}

	return nil
}

func (s *Server) DeleteMission(ctx context.Context, id int64) (err error) {
	defer mon.Start().Stop(&err)

	_, err = s.db.Delete_Mission_By_Id(ctx,
		models.Mission_Id(id))
	return Err.Wrap(err)
}
