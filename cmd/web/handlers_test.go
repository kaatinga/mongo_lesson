package main

import (
	"bytes"
	"context"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"mongo/models"
	"net/http"
	"net/http/httptest"
	"testing"
)

var (
	ctx      = context.Background()
	client   *mongo.Client
	objectID primitive.ObjectID
	db       *mongo.Database
)

func init() {

	var err error

	// Establishing connection to the database
	client, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatalln("Ошибка установки соединения")
	}

	db = client.Database("blog")
}

func TestPost_Insert(t *testing.T) {

	var err error

	t.Run("insert", func(t *testing.T) {

		p := models.Post{
			Mongo:   models.Mongo{ID: objectID},
			Title:   "Test",
			Author:  "Test",
			Content: "Test",
		}
		objectID, err = p.Insert(ctx, db)
		if err != nil {
			t.Errorf("Insert() error = %v, wantErr %v", err, nil)
		}
	})
}

func TestHandlerData_Exist(t *testing.T) {

	log.Println(objectID)

	var err error

	t.Run("must exist", func(t *testing.T) {
		var hd models.HandlerData
		hd.Db = db
		hd.Ctx = ctx

		var exist bool
		exist, err = hd.Exist(objectID)

		if err != nil {
			t.Errorf("Exist() error = %v, wantErr %v", err, nil)
			return
		}

		if exist != true {
			t.Errorf("Exist() got = %v, want %v", exist, true)
		}
	})
}

func TestHandlerData_UpdateBlogPost(t *testing.T) {

	var err error

	t.Run("Must be ok", func(t *testing.T) {
		var hd models.HandlerData
		hd.Db = db

		err = hd.UpdateBlogPost(objectID, "test_done", "test_done", "test_done")
		if err != nil {
			t.Errorf("UpdateBlogPost() error = %v, wantErr %v", err, nil)
		}
	})
}

func TestPost_Delete(t *testing.T) {

	var err error

	t.Run("delete", func(t *testing.T) {

		p := models.Post{
			Mongo: models.Mongo{ID: objectID},
		}

		err = p.Delete(ctx, db)

		if err != nil {
			t.Errorf("Delete() error = %v, wantErr %v", err, nil)
		}
	})
}

func TestWelcome(t *testing.T) {

	var mustBe string

	reader := bytes.NewReader([]byte(mustBe))
	req, _ := http.NewRequest("GET", "/", reader)
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Adapt(Welcome, InitPage(db, ctx))
	})

	handler.ServeHTTP(rr, req)

	resp := rr.Body.String()
	if resp != mustBe {
		t.Errorf("got %s, excpected %s", mustBe, resp)
	}

}
