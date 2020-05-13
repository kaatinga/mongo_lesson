package main

import (
	"context"
	"github.com/julienschmidt/httprouter"
	"github.com/kaatinga/mongo_lesson/cmd/web/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"net/http"
	"time"
)

// Middleware wraps julien's router http methods
type Middleware struct {
	router *httprouter.Router
	db     *mongo.Database
	ctx    context.Context
}

// newMiddleware returns pointer of Middleware
func newMiddleware(r *httprouter.Router, ctx context.Context) *Middleware {
	var db *mongo.Database
	return &Middleware{r, db, ctx}
}

// мидлвейр для всех хэндлеров
func (rw *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("-------------------", time.Now().In(moscow).Format(http.TimeFormat), "A request is received -------------------")
	log.Println("The request is from", r.RemoteAddr, "| Method:", r.Method, "| URI:", r.URL.String())

	if r.Method == "POST" {
		// проверяем размер POST данных
		r.Body = http.MaxBytesReader(w, r.Body, 10000)
		err := r.ParseForm()
		if err != nil {
			logger.SubLogRed("POST data is exceeded the limit")
			http.Error(w, http.StatusText(400), 400)
			return
		}
	}

	rw.router.ServeHTTP(w, r)
}
