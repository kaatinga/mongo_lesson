package main

import (
<<<<<<< HEAD
=======
	"github.com/kaatinga/mongo_lesson/cmd/web/models"

>>>>>>> origin/master
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

	testCases := []string{
		`

<!DOCTYPE html>
<html lang="ru">

<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Учебный блог — Добро пожаловать</title>
    <link rel="stylesheet" href="/static/css/normalize.css">
    <link rel="stylesheet" href="/static/css/style.css">
    <link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Roboto+Condensed&display=swap&subset=cyrillic">
</head>

<body>
    <header>
        <input id="burger" type="checkbox" name="hamb" value="hamb"><label for="burger"><img src="/static/img/close.svg" alt="✖"><span>☰</span></label>
        <a href="/" class="logo" title="Перейти на главную страницу"><img src="/static/img/logo.svg"></a>
        
        <ul class="menu">
            
            <li><a href="/posts/1" >Блог</a></li>
            <li><a href="/post/" >Новая запись</a></li>
            <li><a href="/log/1" >Журнал событий</a></li>
        </ul>
        
    </header>
    

    <main>
        <h2>Добро пожаловать</h2>
        
<p>Вы на главной странице блога.</p>

    </main>
    <footer>
        <div>&copy; Михаил Онищенко aka Kaatinga
            <p style="color: gray"></p>
        </div>
        <div>Версия от: 12.05.2020
            <p style="color: gray">
                Страница запроса: /<br>
                Темплейт: ..\ui\html\index.html<br>
                Код ответа сервера: 200<br>
                URL для отправки данных: </p>

        </div>
    </footer>

</body>

</html>

`,
	}

	for _, tcase := range testCases {

		reader := bytes.NewReader([]byte(tcase))
		req, _ := http.NewRequest("GET", "/", reader)
		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			Adapt(Welcome, InitPage(db, ctx))
		})

		handler.ServeHTTP(rr, req)
		if resp := rr.Body.String(); resp != tcase {
			t.Errorf("got %s, excpected %s", tcase, resp)
		}
	}
}
