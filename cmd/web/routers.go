package main

import (
	"context"
	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"net/url"
	"strings"
)

type Adapter func(httprouter.Handle) httprouter.Handle

func Adapt(next httprouter.Handle, adapters ...Adapter) httprouter.Handle {
	for _, adapter := range adapters {
		next = adapter(next)
	}
	return next
}

func InitPage(db *mongo.Database, ctx context.Context) Adapter {
	return func(next httprouter.Handle) httprouter.Handle {
		return func(w http.ResponseWriter, r *http.Request, actions httprouter.Params) {
			var hd HandlerData
			hd.db = db
			hd.ctx = ctx

			// рендер работает отложенно с проверкой условия
			defer func() {

				switch {
				case hd.formID != "" && hd.data.Status == 0: // в случае если параметр не пустой, нужно делать редирект

					value := url.QueryEscape(hd.formValue)

					if hd.whereToRedirect == "" || value == "" {
						subSubLogRed("Ошибка! Значения пустые")
						hd.data.setError(500, nil)
						hd.data.Render(w)
					}

					subSubLogGreen("Time to redirect!")

					setFormCookie(w, hd.formID, value)
					urlToRedirect := strings.Join([]string{"/", hd.whereToRedirect, "/", hd.formID, "/", value}, "")

					http.Redirect(w, r, urlToRedirect, 303)

				case !hd.noRender: // на случай если файлы хэндлятся, проверяем

					subSubLogGreen("Time to render!")
					hd.data.Render(w)

				}
			}()

			hd.data.URL = r.URL.String()
			hd.data.Method = r.Method

			if hd.data.Status == 0 {

				hd.data.MenuList = make([]MenuData, 0, 3)
				hd.data.MenuList = []MenuData{
					0: {"/posts/1", "Блог", false},
					1: {"/post/", "Новая запись", false},
					2: {"/log/1", "Журнал событий", false},
				}
			}

			hd.data.LastModified = compileDate

			if r.URL.String() == "/" || hd.data.Status == 0 { // если главная страница или не ошибка
				// передаём data используя контекст и запускаем следуюзий хэндлер если всё ок
				ctx := context.WithValue(r.Context(), "hd", &hd)
				next(w, r.WithContext(ctx), actions)
			} else {
				subSubLogYellow("Следующий хэндлер исключён")
			}
		}
	}
}

type HandlerData struct {
	db                   *mongo.Database
	data                 ViewData
	noRender             bool
	formID               string
	formValue            string
	whereToRedirect      string
	additionalRedirectID string
	mainAction           string
	ctx                  context.Context
}

// routes
func SetUpHandlers(m *Middleware) {
	sublog("Setting up handlers...")

	// главная страница
	m.router.GET("/", Adapt(Welcome, InitPage(m.db, m.ctx)))

	// блог
	m.router.GET("/post/", Adapt(BlogForm, InitPage(m.db, m.ctx)))
	m.router.GET("/post/:action/:id", Adapt(BlogForm, InitPage(m.db, m.ctx)))
	m.router.POST("/editpost/:action/:id", Adapt(ProcessPost, InitPage(m.db, m.ctx)))
	m.router.POST("/post/:action/:id", Adapt(ProcessPost, InitPage(m.db, m.ctx)))
	m.router.GET("/deletepost/:id", Adapt(DeletePost, InitPage(m.db, m.ctx)))
	m.router.GET("/posts/:page", Adapt(ListPosts, InitPage(m.db, m.ctx)))

	// Лог
	m.router.GET("/log/:page", Adapt(ListLog, InitPage(m.db, m.ctx)))

	// После формы
	m.router.GET("/afterForm/:id1/:id2", Adapt(AfterForm, InitPage(m.db, m.ctx)))

	//Ошибки
	m.router.NotFound = ErrorStatus(http.StatusNotFound)                 // переопределяем страницу для 404
	m.router.MethodNotAllowed = ErrorStatus(http.StatusMethodNotAllowed) // переопределяем страницу для 405

	// Создаём файлсервер для работы с папкой "../ui/static", относительно main.go.
	// Регистрируем хандлер для созданного файлсервера статики.
	m.router.ServeFiles("/static/*filepath", http.Dir("static")) // Relative path is not supported!

	// Обработчик favicon.ico
	m.router.GET("/favicon.ico", faviconHandler)
}
