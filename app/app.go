package app

import (
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"io"
	"io/ioutil"
	"net/http"
)

type Middleware func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) error

func Handle(middlewares ...Middleware) httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		for _, handler := range middlewares {
			err := handler(w, r, ps)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				switch e := err.(type) {
				case ErrorResponse:
					w.WriteHeader(e.StatusCode())
					json.NewEncoder(w).Encode(&struct {
						Message string        `json:"message"`
						Errors  []interface{} `json:"errors"`
					}{Message: e.Error(), Errors: e.Data()})
				default:
					fmt.Println(e.Error())
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, "{\"message\":\"Internal Error\"}")
				}
				return
			}
		}
	})
}

type App struct {
	Router *httprouter.Router
}

func New() App {
	return App{Router: httprouter.New()}
}

func (a *App) DELETE(path string, middlewares ...Middleware) {
	a.Router.DELETE(path, Handle(middlewares...))
}

func (a *App) GET(path string, middlewares ...Middleware) {
	a.Router.GET(path, Handle(middlewares...))
}

func (a *App) HEAD(path string, middlewares ...Middleware) {
	a.Router.HEAD(path, Handle(middlewares...))
}

func (a *App) OPTIONS(path string, middlewares ...Middleware) {
	a.Router.OPTIONS(path, Handle(middlewares...))
}

func (a *App) PATCH(path string, middlewares ...Middleware) {
	a.Router.PATCH(path, Handle(middlewares...))
}

func (a *App) POST(path string, middlewares ...Middleware) {
	a.Router.POST(path, Handle(middlewares...))
}

func (a *App) PUT(path string, middlewares ...Middleware) {
	a.Router.PUT(path, Handle(middlewares...))
}

func ParseRequestBody(r *http.Request, dest interface{}) error {

	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1000000))
	if err != nil {
		return ErrorResponse{http.StatusRequestEntityTooLarge,
			ErrRequestEntityTooLarge, []ErrorResponseDetail{}}
	}
	if err := r.Body.Close(); err != nil {
		return err
	}

	if err := json.Unmarshal(body, dest); err != nil {
		return ErrorResponse{http.StatusUnprocessableEntity,
			ErrUnprocessableEntity, []ErrorResponseDetail{}}
	}

	return nil

}
