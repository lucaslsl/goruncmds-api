package main

import (
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/julienschmidt/httprouter"
	"github.com/lucaslsl/goruncmds-api/app"
	"github.com/namsral/flag"
	"github.com/rs/cors"
	"github.com/satori/go.uuid"
	"log"
	"net/http"
	"time"
)

type TaskInstruction struct {
	Command             string `json:"command"`
	StopPipelineOnError bool   `json:"stop_pipeline_on_error"`
}

type Task struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Instructions []TaskInstruction `json:"instructions"`
	ServersIDs   []string          `json:"servers_ids"`
	ServersRoles []string          `json:"servers_roles"`
}

var (
	redisURL     = flag.String("redis_url", "redis://localhost:6379", "Redis URL")
	redisChannel = flag.String("redis_channel", "cmds_tasks", "Tasks Channel")
	listenAddr   = flag.String("listen_address", ":8080", "Listen Address")
	authKey      = flag.String("auth_key", "0z02sKnkfLczIlcsi8k5n4f3J7TXuc60", "Authentication Key")
	redisPool    *redis.Pool
)

func newPool(url string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     2,
		IdleTimeout: 240 * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.DialURL(url) },
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

func CreateTask(w http.ResponseWriter, r *http.Request, _ httprouter.Params) error {
	var task Task
	err := app.ParseRequestBody(r, &task)
	if err != nil {
		return err
	}
	taskID := uuid.NewV4().String()
	task.ID = taskID
	taskJson, _ := json.Marshal(task)

	redisConn := redisPool.Get()
	defer redisConn.Close()

	redisKey := "task:" + taskID

	_, err = redisConn.Do("SET", redisKey, string(taskJson))
	if err != nil {
		return err
	}
	_, err = redisConn.Do("SADD", "tasks", redisKey)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(task); err != nil {
		return err
	}
	return nil
}

func RetrieveTask(w http.ResponseWriter, r *http.Request, ps httprouter.Params) error {

	redisConn := redisPool.Get()
	defer redisConn.Close()

	task, err := redis.String(redisConn.Do("GET", "task:"+ps.ByName("taskID")))
	if err != nil {
		return app.ErrorResponse{http.StatusNotFound,
			app.ErrNotFound, []app.ErrorResponseDetail{}}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, task)
	return nil
}

func UpdateTask(w http.ResponseWriter, r *http.Request, ps httprouter.Params) error {

	redisConn := redisPool.Get()
	defer redisConn.Close()

	redisKey := "task:" + ps.ByName("taskID")

	t, err := redis.String(redisConn.Do("GET", redisKey))
	if err != nil {
		return app.ErrorResponse{http.StatusNotFound,
			app.ErrNotFound, []app.ErrorResponseDetail{}}
	}
	var task Task
	err = json.Unmarshal([]byte(t), &task)
	if err != nil {
		return err
	}

	var taskUpdated Task
	err = app.ParseRequestBody(r, &taskUpdated)
	if err != nil {
		return err
	}
	taskUpdated.ID = task.ID

	taskUpdatedJson, _ := json.Marshal(taskUpdated)
	_, err = redisConn.Do("SET", redisKey, string(taskUpdatedJson))
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, string(taskUpdatedJson))
	return nil
}

func DeleteTask(w http.ResponseWriter, r *http.Request, ps httprouter.Params) error {

	redisConn := redisPool.Get()
	defer redisConn.Close()

	redisKey := "task:" + ps.ByName("taskID")

	exists, err := redis.Bool(redisConn.Do("EXISTS", redisKey))
	if err != nil || !exists {
		return app.ErrorResponse{http.StatusNotFound,
			app.ErrNotFound, []app.ErrorResponseDetail{}}
	}

	_, err = redisConn.Do("SREM", "tasks", redisKey)
	if err != nil {
		return err
	}

	_, err = redisConn.Do("DEL", redisKey)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func TaskList(w http.ResponseWriter, r *http.Request, ps httprouter.Params) error {

	redisConn := redisPool.Get()
	defer redisConn.Close()

	members, err := redis.Strings(redisConn.Do("SMEMBERS", "tasks"))
	if err != nil {
		return err
	}

	var tasksToFetch []interface{}
	for _, k := range members {
		tasksToFetch = append(tasksToFetch, k)
	}

	var values []string

	if len(tasksToFetch) > 0 {
		values, err = redis.Strings(redisConn.Do("MGET", tasksToFetch...))
		if err != nil {
			return err
		}
	}

	tasks := []Task{}
	var tmp Task
	for _, t := range values {
		if err = json.Unmarshal([]byte(t), &tmp); err == nil {
			tasks = append(tasks, tmp)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(tasks); err != nil {
		return err
	}
	return nil

}

func RunTask(w http.ResponseWriter, r *http.Request, ps httprouter.Params) error {
	redisConn := redisPool.Get()
	defer redisConn.Close()

	redisKey := "task:" + ps.ByName("taskID")

	task, err := redis.String(redisConn.Do("GET", redisKey))
	if err != nil {
		return app.ErrorResponse{http.StatusNotFound,
			app.ErrNotFound, []app.ErrorResponseDetail{}}
	}

	_, err = redisConn.Do("PUBLISH", *redisChannel, task)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func init() {
	flag.Parse()
}

func main() {
	redisPool = newPool(*redisURL)
	app := app.New()
	app.POST("/api/v1/tasks", IsAuthenticated, CreateTask)
	app.GET("/api/v1/tasks", IsAuthenticated, TaskList)
	app.GET("/api/v1/tasks/:taskID", IsAuthenticated, RetrieveTask)
	app.PATCH("/api/v1/tasks/:taskID", IsAuthenticated, UpdateTask)
	app.DELETE("/api/v1/tasks/:taskID", IsAuthenticated, DeleteTask)
	app.PUT("/api/v1/tasks/:taskID/run", IsAuthenticated, RunTask)
	handler := cors.Default().Handler(app.Router)
	log.Fatal(http.ListenAndServe(*listenAddr, handler))
}
