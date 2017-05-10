package main

import (
	"encoding/json"
	"github.com/garyburd/redigo/redis"
	"github.com/julienschmidt/httprouter"
	"github.com/lucaslsl/goexeccmds-api/app"
	"github.com/namsral/flag"
	"github.com/rs/cors"
	"log"
	"net/http"
	"time"
)

type TaskInstruction struct {
	Command             string `json:"command"`
	StopPipelineOnError bool   `json:"stop_pipeline_on_error"`
}

type Task struct {
	Name         string            `json:"name"`
	Instructions []TaskInstruction `json:"instructions"`
	ServersIDs   []string          `json:"servers_ids"`
	ServersRoles []string          `json:"servers_roles"`
}

var (
	redisAddr    = flag.String("redis_address", "localhost:6379", "Redis Address")
	redisChannel = flag.String("redis_channel", "cmds_tasks", "Tasks Channel")
	listenAddr   = flag.String("listen_address", ":8080", "Listen Address")
	authKey      = flag.String("auth_key", "0z02sKnkfLczIlcsi8k5n4f3J7TXuc60", "Authentication Key")
	redisPool    *redis.Pool
)

func newPool(addr string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     2,
		IdleTimeout: 240 * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.Dial("tcp", addr) },
	}
}

func IsAuthenticated(w http.ResponseWriter, r *http.Request, _ httprouter.Params) error {
	k := r.URL.Query().Get("auth_key")
	if k != *authKey {
		return app.ErrorResponse{http.StatusUnauthorized,
			app.ErrUnauthorized, []app.ErrorResponseDetail{}}
	}
	return nil
}

func RunTask(w http.ResponseWriter, r *http.Request, _ httprouter.Params) error {
	var t Task
	err := app.ParseRequestBody(r, &t)
	if err != nil {
		return err
	}
	tJson, _ := json.Marshal(t)

	redisConn := redisPool.Get()
	defer redisConn.Close()

	_, err = redisConn.Do("PUBLISH", *redisChannel, string(tJson))
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func main() {
	redisPool = newPool(*redisAddr)
	app := app.New()
	app.PUT("/runtask", IsAuthenticated, RunTask)
	handler := cors.Default().Handler(app.Router)
	log.Fatal(http.ListenAndServe(*listenAddr, handler))
}
