package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gernest/alien"
	"github.com/goofin/scopby/server"
	"github.com/zeebo/mon"
)

type resources struct {
	srv server.Server
}

func New(srv server.Server) http.Handler {
	r := &resources{srv: srv}

	mux := alien.New()
	mux.NotFoundHandler(http.HandlerFunc(wrapHandler(r.notFound)))
	mux.Get("/users/:name", wrapHandler(r.getUser))
	mux.Put("/users/:name", wrapHandler(r.putUser))
	mux.Patch("/users/:name", wrapHandler(r.patchUser))
	mux.Get("/users/:name/missions", wrapHandler(r.getMissions))
	mux.Put("/users/:name/mission", wrapHandler(r.putMission))
	mux.Post("/mission/:id/complete", wrapHandler(r.postMissionComplete))
	mux.Delete("/mission/:id", wrapHandler(r.deleteMission))

	return mux
}

func (r *resources) notFound(ctx context.Context, req *http.Request) (
	data interface{}, err error) {
	defer mon.Start().Stop(&err)

	return nil, Err.New("not found")
}

func (r *resources) getUser(ctx context.Context, req *http.Request) (
	data interface{}, err error) {
	defer mon.Start().Stop(&err)

	params := alien.GetParams(req)
	user, err := r.srv.GetUser(ctx,
		params.Get("name"))
	return jsonUser(user), Err.Wrap(err)
}

func (r *resources) putUser(ctx context.Context, req *http.Request) (
	data interface{}, err error) {
	defer mon.Start().Stop(&err)

	var putParams struct {
		Timezone int `json:"timezone"`
	}
	if err := json.NewDecoder(req.Body).Decode(&putParams); err != nil {
		return nil, Err.Wrap(err)
	}

	params := alien.GetParams(req)
	return nil, Err.Wrap(r.srv.CreateUser(ctx,
		params.Get("name"),
		putParams.Timezone))
}

func (r *resources) patchUser(ctx context.Context, req *http.Request) (
	data interface{}, err error) {
	defer mon.Start().Stop(&err)

	var patchParams struct {
		Snacks int `json:"snacks"`
	}
	if err := json.NewDecoder(req.Body).Decode(&patchParams); err != nil {
		return nil, Err.Wrap(err)
	}

	params := alien.GetParams(req)
	return nil, Err.Wrap(r.srv.AddSnacks(ctx,
		params.Get("name"),
		patchParams.Snacks))
}

func (r *resources) getMissions(ctx context.Context, req *http.Request) (
	data interface{}, err error) {
	defer mon.Start().Stop(&err)

	params := alien.GetParams(req)
	missions, err := r.srv.GetMissions(ctx,
		params.Get("name"))
	return jsonMissions(missions), Err.Wrap(err)
}

func (r *resources) putMission(ctx context.Context, req *http.Request) (
	data interface{}, err error) {
	defer mon.Start().Stop(&err)

	var putParams struct {
		Description string `json:"description"`
		Seconds     int    `json:"seconds"`
		Snacks      int    `json:"snacks"`
	}
	if err := json.NewDecoder(req.Body).Decode(&putParams); err != nil {
		return nil, Err.Wrap(err)
	}

	params := alien.GetParams(req)
	return nil, Err.Wrap(r.srv.CreateMission(ctx,
		params.Get("name"),
		putParams.Description,
		putParams.Seconds,
		putParams.Snacks))
}

func (r *resources) postMissionComplete(ctx context.Context, req *http.Request) (
	data interface{}, err error) {
	defer mon.Start().Stop(&err)

	params := alien.GetParams(req)
	id, err := strconv.ParseInt(params.Get("id"), 10, 64)
	if err != nil {
		return nil, Err.Wrap(err)
	}
	return nil, Err.Wrap(r.srv.CompleteMission(ctx,
		id))
}

func (r *resources) deleteMission(ctx context.Context, req *http.Request) (
	data interface{}, err error) {
	defer mon.Start().Stop(&err)

	params := alien.GetParams(req)
	id, err := strconv.ParseInt(params.Get("id"), 10, 64)
	if err != nil {
		return nil, Err.Wrap(err)
	}
	return nil, Err.Wrap(r.srv.DeleteMission(ctx,
		id))
}
