package main

import (
<<<<<<< HEAD
	"context"
	"errors"
=======
	"./logger"
	"./models"
	"context"
	"errors"
	"github.com/julienschmidt/httprouter"
	my "github.com/kaatinga/assets"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
>>>>>>> origin/master
	"log"
	"mongo/logger"
	"mongo/models"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/julienschmidt/httprouter"
	my "github.com/kaatinga/assets"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Welcome is the homepage of the service
func Welcome(_ http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	hd := r.Context().Value("hd").(*models.HandlerData)

	if hd.Data.Status == 0 {
		hd.Data.Title = "Добро пожаловать"
		hd.Data.Text = "Вы на главной странице блога."
	}
}

// == Blog handlers ==

// UserForm shows the user form in case of updating or creating new user
func BlogForm(w http.ResponseWriter, r *http.Request, actions httprouter.Params) {

	var (
		hd   = r.Context().Value("hd").(*models.HandlerData)
		post BlogPost
	)

	hd.MainAction = actions.ByName("action")
	switch hd.MainAction {
	case "": // значит новый пост
		hd.Data.PostURL = "/post/"
		hd.Data.Title = newPost
		setFormCookie(w, "addPost", "ok") // устанавливаем сессию формы
	case "update":

		hex := actions.ByName("id")
		var err error

		post.Post.ID, err = primitive.ObjectIDFromHex(hex)
		if err != nil {
			hd.Data.SetError(http.StatusBadRequest, errors.New("incorrect blog post id"))
			return
		}

		hd.Data.PostURL = strings.Join([]string{"/editpost/update/", hex}, "")
		hd.Data.Title = editPost
		setFormCookie(w, "editPost", hex) // устанавливаем сессию формы

		err = post.getData(hd.Db, hd.Ctx)
		if err != nil {
			hd.Data.SetError(http.StatusBadRequest, errors.New("ошибка чтения данных из бд"))
			return
		}
	}

	hd.Data.PageData = post
	hd.Data.Template = filepath.Join("..", "ui", "html", "post.html") // путь к контенту страницы
}

// CreateUpdateUser creates or updates users
func ProcessPost(w http.ResponseWriter, r *http.Request, actions httprouter.Params) {
	var (
		hd       = r.Context().Value("hd").(*models.HandlerData)
		blogPost BlogPost
		err      error // ошибки
		ok       bool  // простые ошибки
		hex      string
		objectID primitive.ObjectID
	)

	blogPost.Post.Author = strings.TrimSpace(r.FormValue("author"))
	blogPost.Post.Title = strings.TrimSpace(r.FormValue("title"))
	blogPost.Post.Content = strings.TrimSpace(r.FormValue("blogpost"))

	action := actions.ByName("action")
	if action == "update" {
		logger.SubLog("Updating a post...")

		hd.Data.Title = editPost

		hex = actions.ByName("id")

		// проверяем есть ли кука для этого запроса
		_, err = checkFormCookie(w, r, "editPost", hex)
		if err != nil {
			hd.Data.SetError(400, err)
			return
		}

		objectID, err = primitive.ObjectIDFromHex(hex)
		if err != nil {
			hd.Data.SetError(400, errors.New("неверный ID записи в блоге"))
			return
		}

		hd.Data.PostURL = strings.Join([]string{"/editpost/update/", hex}, "")

		ok, err = hd.Exist(objectID)
		if err != nil {
			hd.Data.SetError(400, err)
			return
		}

		if !ok {
			hd.Data.SetError(400, errors.New("no such an ID in the db"))
			return
		}
	} else {
		logger.SubLog("Creating a post...")

		// проверяем есть ли кука для этого запроса
		_, err = checkFormCookie(w, r, "addPost", "ok")
		if err != nil {
			hd.Data.SetError(400, err)
			return
		}

		hd.Data.PostURL = "/post/"
		hd.Data.Title = newPost
	}

	// проверка ошибок которые возвращаются пользователю
	switch {
	case len(blogPost.Post.Author) < 5:
		blogPost.AuthorError = nameTooShort
	case len(blogPost.Post.Title) < 12:
		blogPost.TitleError = nameTooShort
	case len(blogPost.Post.Content) < 32:
		blogPost.TextError = nameTooShort
	case !my.CheckName(blogPost.Post.Author):
		blogPost.AuthorError = onlyRussian
	default: // если ошибок не найдено
		hd.WhereToRedirect = "afterForm"

		switch action {
		case "update":

			// обновляем данные пользователя в БД
			if err != nil {
				hd.Data.SetError(503, err)
				return
			}
			err = hd.UpdateBlogPost(objectID, blogPost.Post.Author, blogPost.Post.Title, blogPost.Post.Content)
			if err != nil {
				hd.Data.SetError(503, err)
				return
			}

			hd.AddToLog(strings.Join([]string{"Пользователь<b>", blogPost.Post.Author, "</b>обновил запись с id"}, " "), blogPost.Post.Author)
			hd.FormID = "postUpdated"
			hd.FormValue = hex
			return

		// новый пост
		default:
			// запись
			post := &models.Post{
<<<<<<< HEAD
				Title:   blogPost.Post.Title,
				Author:  blogPost.Post.Author,
				Content: blogPost.Post.Content,
=======
				Title:   blogPost.Title,
				Author:  blogPost.Author,
				Content: blogPost.Content,
>>>>>>> origin/master
			}
			_, err = post.Insert(hd.Ctx, hd.Db)
			if err != nil {
				hd.Data.SetError(503, err)
				return
			}

			hd.AddToLog(strings.Join([]string{"В блог добавлена новая запись пользователя:<b>", blogPost.Post.Author, "</b>"}, " "), blogPost.Post.Author)
			hd.FormID = "postCreated"
			hd.FormValue = blogPost.Post.Author
			return
		}
	}

	if hd.Data.Status == 0 { // Добавляем данные с ошибками формы только в случае ошибки ввода данных
		switch action {
		case "update":
			setFormCookie(w, "editPost", hex) // устанавливаем сессию формы
		default:
			setFormCookie(w, "addPost", "ok") // устанавливаем сессию формы
		}
		hd.Data.PageData = blogPost
		hd.Data.Template = filepath.Join("..", "ui", "html", "post.html")
	}
}

// DeletePost deletes a post in the blog
func DeletePost(w http.ResponseWriter, r *http.Request, actions httprouter.Params) {
	hd := r.Context().Value("hd").(*models.HandlerData)

	logger.SubLog("Deleting a blog post...")
	hd.Data.Title = "Удаление записи в блоге"

	postID, err := primitive.ObjectIDFromHex(actions.ByName("id"))
	if err != nil {
		hd.Data.SetError(500, errors.New("неверный ID записи в блоге"))
		return
	}

	err = hd.DeleteFromDB(postID)
	if err != nil {
		hd.Data.SetError(500, err)
		return
	}

	hd.WhereToRedirect = "afterForm"
	hd.FormID = "postDeleted"
	hd.FormValue = "ok"
	hd.AddToLog(strings.Join([]string{"Роль удалёна, ID <b>", postID.String(), "</b>"}, ""), "unknown")
}

// PRG pattern, защита от повторных POST-запросов
func AfterForm(w http.ResponseWriter, r *http.Request, actions httprouter.Params) {
	var (
		hd     = r.Context().Value("hd").(*models.HandlerData)
		id1    = actions.ByName("id1")
		postID string
		err    error
	)

	// проверяем сессию формы и возвращаем postID, так как этот параметр небезопасный
	postID, err = checkFormCookie(w, r, id1, actions.ByName("id2"))
	if err != nil {
		hd.Data.SetError(400, err)
		return
	}

	switch id1 {
	case "postCreated":

		hd.Data.Title = "Создание новой записи"
		hd.Data.Text = strings.Join([]string{"Добавлена новая запись в блог:<b>", postID, "</b>"}, " ")

	case "postUpdated":

		hd.Data.Title = "Обновление записи"
		hd.Data.Text = strings.Join([]string{"Запись<b>", postID, "</b>обновлена"}, " ")

	case "postDeleted":

		hd.Data.Title = "Удаление записи"
		hd.Data.Text = strings.Join([]string{"Запись<b>", postID, "</b>удалена"}, " ")

	default:

		hd.Data.SetError(400, errors.New("unknown action"))

	}
}

type PostList struct {
	Posts []models.Post
	Paginator
}

type BlogPost struct {
<<<<<<< HEAD
	Post models.Post
=======
	models.Post
>>>>>>> origin/master
	PostErrors
}

type PostErrors struct {
	AuthorError string
	TitleError  string
	TextError   string
}

// == Blog list ==

// ListRoles shows the list of the roles
func ListPosts(_ http.ResponseWriter, r *http.Request, action httprouter.Params) {
	hd := r.Context().Value("hd").(*models.HandlerData)

	logger.SubLog("Post list is requested")

	var err error
	var postList PostList

	currentPage, _ := my.StUint16(action.ByName("page"))
	err = postList.FillOut(currentPage, "", (*hd).Db)
	if err != nil {
		hd.Data.SetError(503, err)
		return
	}

	postList.Posts, err = hd.GetPosts()
	if err != nil {
		hd.Data.SetError(400, err)
		return
	}

	//добавляем данные о записях в блоге в структуру данных страницы
	hd.Data.PageData = postList
	hd.Data.Title = "Блог"
	hd.Data.Template = filepath.Join("..", "ui", "html", "posts.html") // уникальный темплейт страницы
}

// == Others ==

func faviconHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	log.Println("favicon.ico будет передан по запросу")
	http.ServeFile(w, r, "../ui/static/img/favicon.ico")
}

func ErrorStatus(status uint16) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var data models.ViewData
		data.SetError(status, nil)

		//чертим страницу
		data.Render(w)
	}
}

func (post *BlogPost) getData(localDB *mongo.Database, ctx context.Context) (err error) {

	// чтение
	tmpPost := &models.Post{}

	tmpPost, err = models.GetPost(ctx, localDB, post.Post.ID)
	if err != nil {
		return err
	}

	post.Post = *tmpPost

	return nil
}

func GetPost(ctx context.Context, db *mongo.Database, id primitive.ObjectID) (*models.Post, error) {
	var p models.Post
	coll := db.Collection(p.GetMongoCollectionName())
	res := coll.FindOne(ctx, bson.M{"_id": id})
	if err := res.Decode(&p); err != nil {
		return nil, err
	}
	return &p, nil
}
