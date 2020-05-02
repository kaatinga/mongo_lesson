package main

import (
	"context"
	"errors"
	"github.com/julienschmidt/httprouter"
	my "github.com/kaatinga/assets"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

// Welcome is the homepage of the service
func Welcome(_ http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	hd := r.Context().Value("hd").(*HandlerData)

	if hd.data.Status == 0 {
		hd.data.Title = "Добро пожаловать"
		hd.data.Text = "Вы на главной странице блога."
	}
}

// == Blog handlers ==

// UserForm shows the user form in case of updating or creating new user
func BlogForm(w http.ResponseWriter, r *http.Request, actions httprouter.Params) {

	var (
		hd   = r.Context().Value("hd").(*HandlerData)
		post BlogPost
	)

	hd.mainAction = actions.ByName("action")
	switch hd.mainAction {
	case "": // значит новый пост
		hd.data.PostURL = "/post/"
		hd.data.Title = newPost
		setFormCookie(w, "addPost", "ok") // устанавливаем сессию формы
	case "update":

		hex := actions.ByName("id")
		var err error

		post.ID, err = primitive.ObjectIDFromHex(hex)
		if err != nil {
			hd.data.setError(http.StatusBadRequest, errors.New("incorrect blog post id"))
			return
		}

		hd.data.PostURL = strings.Join([]string{"/editpost/update/", hex}, "")
		hd.data.Title = editPost
		setFormCookie(w, "editPost", hex) // устанавливаем сессию формы

		err = post.getData(hd.db, hd.ctx)
		if err != nil {
			hd.data.setError(http.StatusBadRequest, errors.New("ошибка чтения данных из бд"))
			return
		}
	}

	hd.data.PageData = post
	hd.data.Template = filepath.Join("..", "ui", "html", "post.html") // путь к контенту страницы
}

// CreateUpdateUser creates or updates users
func ProcessPost(w http.ResponseWriter, r *http.Request, actions httprouter.Params) {
	var (
		hd       = r.Context().Value("hd").(*HandlerData)
		blogPost BlogPost
		err      error // ошибки
		ok       bool  // простые ошибки
		hex      string
		objectID primitive.ObjectID
	)

	blogPost.Author = strings.TrimSpace(r.FormValue("author"))
	blogPost.Title = strings.TrimSpace(r.FormValue("title"))
	blogPost.Content = strings.TrimSpace(r.FormValue("blogpost"))

	action := actions.ByName("action")
	if action == "update" {
		sublog("Updating a post...")

		hd.data.Title = editPost

		hex = actions.ByName("id")

		// проверяем есть ли кука для этого запроса
		_, err = checkFormCookie(w, r, "editPost", hex)
		if err != nil {
			hd.data.setError(400, err)
			return
		}

		objectID, err = primitive.ObjectIDFromHex(hex)
		if err != nil {
			hd.data.setError(400, errors.New("неверный ID записи в блоге"))
			return
		}

		hd.data.PostURL = strings.Join([]string{"/editpost/update/", hex}, "")

		ok, err = hd.Exist(objectID)
		if err != nil {
			hd.data.setError(400, err)
			return
		}

		if !ok {
			hd.data.setError(400, errors.New("no such an ID in the db"))
			return
		}
	} else {
		sublog("Creating a post...")

		// проверяем есть ли кука для этого запроса
		_, err = checkFormCookie(w, r, "addPost", "ok")
		if err != nil {
			hd.data.setError(400, err)
			return
		}

		hd.data.PostURL = "/post/"
		hd.data.Title = newPost
	}

	// проверка ошибок которые возвращаются пользователю
	switch {
	case len(blogPost.Author) < 5:
		blogPost.AuthorError = nameTooShort
	case len(blogPost.Title) < 12:
		blogPost.TitleError = nameTooShort
	case len(blogPost.Content) < 32:
		blogPost.TextError = nameTooShort
	case !my.CheckName(blogPost.Author):
		blogPost.AuthorError = onlyRussian
	default: // если ошибок не найдено
		hd.whereToRedirect = "afterForm"

		switch action {
		case "update":
			// обновляем данные пользователя в БД
			if err != nil {
				hd.data.setError(503, err)
				return
			}
			err = hd.UpdateBlogPost(objectID, blogPost.Author, blogPost.Title, blogPost.Content)
			if err != nil {
				hd.data.setError(503, err)
				return
			}

			hd.AddToLog(strings.Join([]string{"Пользователь<b>", blogPost.Author, "</b>обновил запись с id"}, " "), blogPost.Author)
			hd.formID = "postUpdated"
			hd.formValue = hex
			return

		// новый пост
		default:
			// запись
			post := &Post{
				Title:   blogPost.Title,
				Author:  blogPost.Author,
				Content: blogPost.Content,
			}
			err = post.Insert(hd.ctx, hd.db)
			if err != nil {
				hd.data.setError(503, err)
				return
			}

			hd.AddToLog(strings.Join([]string{"В блог добавлена новая запись пользователя:<b>", blogPost.Author, "</b>"}, " "), blogPost.Author)
			hd.formID = "postCreated"
			hd.formValue = blogPost.Author
			return
		}
	}

	if hd.data.Status == 0 { // Добавляем данные с ошибками формы только в случае ошибки ввода данных
		switch action {
		case "update":
			setFormCookie(w, "editPost", hex) // устанавливаем сессию формы
		default:
			setFormCookie(w, "addPost", "ok") // устанавливаем сессию формы
		}
		hd.data.PageData = blogPost
		hd.data.Template = filepath.Join("..", "ui", "html", "post.html")
	}
}

// DeletePost deletes a post in the blog
func DeletePost(w http.ResponseWriter, r *http.Request, actions httprouter.Params) {
	hd := r.Context().Value("hd").(*HandlerData)

	sublog("Deleting a blog post...")
	hd.data.Title = "Удаление записи в блоге"

	postID, err := primitive.ObjectIDFromHex(actions.ByName("id"))
	if err != nil {
		hd.data.setError(500, errors.New("неверный ID записи в блоге"))
		return
	}

	err = hd.DeleteFromDB(postID)
	if err != nil {
		hd.data.setError(500, err)
		return
	}

	hd.whereToRedirect = "afterForm"
	hd.formID = "postDeleted"
	hd.formValue = "ok"
	hd.AddToLog(strings.Join([]string{"Роль удалёна, ID <b>", postID.String(), "</b>"}, ""), "unknown")
}

// PRG pattern, защита от повторных POST-запросов
func AfterForm(w http.ResponseWriter, r *http.Request, actions httprouter.Params) {
	var (
		hd     = r.Context().Value("hd").(*HandlerData)
		id1    = actions.ByName("id1")
		postID string
		err    error
	)

	// проверяем сессию формы и возвращаем postID, так как этот параметр небезопасный
	postID, err = checkFormCookie(w, r, id1, actions.ByName("id2"))
	if err != nil {
		hd.data.setError(400, err)
		return
	}

	switch id1 {
	case "postCreated":

		hd.data.Title = "Создание новой записи"
		hd.data.Text = strings.Join([]string{"Добавлена новая запись в блог:<b>", postID, "</b>"}, " ")

	case "postUpdated":

		hd.data.Title = "Обновление записи"
		hd.data.Text = strings.Join([]string{"Запись<b>", postID, "</b>обновлена"}, " ")

	case "postDeleted":

		hd.data.Title = "Удаление записи"
		hd.data.Text = strings.Join([]string{"Запись<b>", postID, "</b>удалена"}, " ")

	default:

		hd.data.setError(400, errors.New("unknown action"))

	}
}

type PostList struct {
	Posts []Post
	Paginator
}

type BlogPost struct {
	Post
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
	hd := r.Context().Value("hd").(*HandlerData)

	sublog("Post list is requested")

	var err error
	var postList PostList

	currentPage, _ := my.StUint16(action.ByName("page"))
	err = postList.FillOut(currentPage, "", (*hd).db)
	if err != nil {
		hd.data.setError(503, err)
		return
	}

	postList.Posts, err = hd.GetPosts()
	if err != nil {
		hd.data.setError(400, err)
		return
	}

	//добавляем данные о записях в блоге в структуру данных страницы
	hd.data.PageData = postList
	hd.data.Title = "Блог"
	hd.data.Template = filepath.Join("..", "ui", "html", "posts.html") // уникальный темплейт страницы
}

// == Others ==

func faviconHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	log.Println("favicon.ico будет передан по запросу")
	http.ServeFile(w, r, "../ui/static/img/favicon.ico")
}

func ErrorStatus(status uint16) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var data ViewData
		data.setError(status, nil)

		//чертим страницу
		data.Render(w)
	}
}

// Exist checks existence in the database
func (hd *HandlerData) Exist(_ primitive.ObjectID) (ok bool, err error) {

	subsublog("Checking existence of an item in the database...")

	return true, nil
}

// UpdateBlogPost tries to update a post in the database using the given data
func (hd *HandlerData) UpdateBlogPost(id primitive.ObjectID, author, title, content string) (err error) {

	post := Post{
		Mongo:   Mongo{ID: id},
		Title:   title,
		Content: content,
		Author: author,
	}

	err = post.Update(hd.ctx, hd.db)
	if err != nil {
		return err
	}

	return nil
}

func (post *BlogPost) getData(localDB *mongo.Database, ctx context.Context) (err error) {

	// чтение
	tmpPost := &Post{}

	tmpPost, err = GetPost(ctx, localDB, post.ID)
	if err != nil {
		return err
	}

	post.Post = *tmpPost

	return nil
}
