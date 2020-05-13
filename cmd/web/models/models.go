package models

import (
	"context"
	"fmt"
	"log"
	"mongo/logger"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"text/template"
)

// ViewData - модель данных страницы
type ViewData struct {
	Error        string
	Title        string
	Text         string
	Template     string // Путь к файлу шаблона
	Status       uint16
	LastModified string
	URL          string
	PostURL      string
	Method       string
	MenuList     []MenuData
	PageData     interface{} // Разные данные, любые
}

// Функция исполняет типовые действия в случае ошибки. Вызывается из formRequest() в случае любой ошибки
func (data *ViewData) SetError(status uint16, err error) {

	// устанавливаем статус
	data.Status = status

	// записываем ошибку в модель
	if err != nil {
		data.Error = err.Error()

		// выводим ошибку в лог
		logger.SubLogRed(err.Error())
	}
}

// Render - Функция для вывода страницы пользователю
func (data *ViewData) Render(w http.ResponseWriter) {

	// проверяем что есть ошибка и сообщаем в лог
	if data.Status != 0 {

		if data.Title == "" {
			data.Title = strconv.Itoa(int((*data).Status))
		}

		if data.Text == "" {
			data.Text = http.StatusText(int((*data).Status))
		}

		if data.Text == "" { // Может так быть что StatusText() ничего не вернёт, тогда дополняет текст сами
			data.Text = strings.Join([]string{"Ошибка обработки запроса, код ошибки", (*data).Title}, " ")
		}

		w.WriteHeader(int(data.Status)) // Добавляем в заголовок сообщение об ошибке
		logger.Subsublog("The code is not 200, the status code is", strconv.Itoa(int((*data).Status)))
	} else {
		data.Status = 200
	}

	// путь к основному шаблону
	layout := filepath.Join("..", "ui", "html", "base", "base.html")
	authBlock := filepath.Join("..", "ui", "html", "base", "noauth.html") // пока без аутентификации

	var tmpl *template.Template
	var err error

	if data.Template == "" {
		data.Template = filepath.Join("..", "ui", "html", "index.html") // дефолтный контент
	}

	logger.SubLog("Template was used:", (*data).Template)

	tmpl, err = template.ParseFiles(layout, authBlock, (*data).Template)

	if err != nil {
		// Вываливаем в лог кучу хлама для анализа. Нужно переписать и выводить в файл.
		logger.SubLogRed(err.Error())
		// Возвращаем ошибку пользователю
		http.Error(w, http.StatusText(500), 500)
		return
	}

	err = tmpl.ExecuteTemplate(w, "base", data)
	if err != nil {
		logger.SubLogRed(err.Error())
		http.Error(w, http.StatusText(500), 500)
		return
	}

	logger.Subsublog("Ошибки при формировании страницы по шаблону не обнаружено") // если ошибки нет
}

// MenuData - Модель данных ссылки на страницу
type MenuData struct {
	URL      string
	Name     string
	Selected bool
}

type HandlerData struct {
	Db                   *mongo.Database
	Data                 ViewData
	NoRender             bool
	FormID               string
	FormValue            string
	WhereToRedirect      string
	AdditionalRedirectID string
	MainAction           string
	Ctx                  context.Context
}

// Exist checks existence in the database
func (hd *HandlerData) Exist(id primitive.ObjectID) (bool, error) {

	logger.Subsublog("Checking existence of an item in the database...")

	post := Post{
		Mongo: Mongo{ID: id},
	}

	collection := (*hd).Db.Collection(post.GetMongoCollectionName())

	count, err := collection.CountDocuments((*hd).Ctx, bson.D{{"_id", id}})
	if err != nil {
		return false, err
	}

	logger.Subsublog("найдено:", strconv.Itoa(int(count)))

	if count == 0 {
		return false, nil
	}

	return true, nil
}

// UpdateBlogPost tries to update a post in the database using the given data
func (hd *HandlerData) UpdateBlogPost(id primitive.ObjectID, author, title, content string) (err error) {

	post := Post{
		Mongo:   Mongo{ID: id},
		Title:   title,
		Content: content,
		Author:  author,
	}

	err = post.Update(hd.Db)
	if err != nil {
		return err
	}

	return nil
}

func (hd *HandlerData) GetPosts() ([]Post, error) {
	p := Post{}
	coll := (*hd).Db.Collection(p.GetMongoCollectionName())

	cur, err := coll.Find((*hd).Ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	var posts []Post
	if err = cur.All((*hd).Ctx, &posts); err != nil {
		return nil, err
	}

	return posts, nil
}

func (hd *HandlerData) AddToLog(event, author string) {
	//TODO:
}

type Mongo struct {
	ID primitive.ObjectID `bson:"_id"`
}

func (m *Mongo) GetMongoCollectionName() string {
	panic("GetMongoCollectionName not implemented")
	return ""
}

type Post struct {
	Mongo   `inline`
	Title   string `bson:"title"`
	Author  string `bson:"author"`
	Content string `bson:"content"`
}

func (p *Post) GetMongoCollectionName() string {
	return "posts"
}

func (p *Post) Insert(ctx context.Context, db *mongo.Database) (objectID primitive.ObjectID, err error) {
	p.ID = primitive.NewObjectID()

	coll := db.Collection(p.GetMongoCollectionName())

	var result *mongo.InsertOneResult
	result, err = coll.InsertOne(ctx, p)
	if err != nil {
		return
	}

	objectID = result.InsertedID.(primitive.ObjectID)

	return
}

func GetPost(ctx context.Context, db *mongo.Database, id primitive.ObjectID) (*Post, error) {
	var p Post
	coll := db.Collection(p.GetMongoCollectionName())
	res := coll.FindOne(ctx, bson.M{"_id": id})
	if err := res.Decode(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (p *Post) Update(db *mongo.Database) error {
	coll := db.Collection(p.GetMongoCollectionName())

	opts := options.Update().SetUpsert(true)
	filter := bson.D{{"_id", p.ID}}
	update := bson.D{{"$set", bson.D{{"title", p.Title}, {"content", p.Content}, {"author", p.Author}}}}
	result, err := coll.UpdateOne(context.TODO(), filter, update, opts)
	if err != nil {
		log.Println(err)
	}

	if result.MatchedCount != 0 {
		fmt.Println("matched and replaced an existing document")
		return nil
	}

	return err
}

func (p *Post) Delete(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(p.GetMongoCollectionName())
	_, err := coll.DeleteOne(ctx, bson.M{"_id": p.ID})
	return err
}

// DeleteFromDB удаляет сущность из базы
func (hd *HandlerData) DeleteFromDB(what primitive.ObjectID) error {

	postDelete := Post{Mongo: Mongo{ID: what}}
	if err := postDelete.Delete(hd.Ctx, hd.Db); err != nil {
		return err
	}
	return nil
}
