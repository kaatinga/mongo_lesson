package main

import (
	"context"
	"github.com/swaggo/http-swagger"
	"mongo/logger"
	"mongo/models"
	"net/http"
	"net/url"
	"strings"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/mongo"
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
			var hd models.HandlerData
			hd.Db = db
			hd.Ctx = ctx

			// рендер работает отложенно с проверкой условия
			defer func() {

				switch {
				case hd.FormID != "" && hd.Data.Status == 0: // в случае если параметр не пустой, нужно делать редирект

					value := url.QueryEscape(hd.FormValue)

					if hd.WhereToRedirect == "" || value == "" {
						logger.SubSubLogRed("Ошибка! Значения пустые")
						hd.Data.SetError(500, nil)
						hd.Data.Render(w)
					}

					logger.SubSubLogGreen("Time to redirect!")

					setFormCookie(w, hd.FormID, value)
					urlToRedirect := strings.Join([]string{"/", hd.WhereToRedirect, "/", hd.FormID, "/", value}, "")

					http.Redirect(w, r, urlToRedirect, 303)

				case !hd.NoRender: // на случай если файлы хэндлятся, проверяем

					logger.SubSubLogGreen("Time to render!")
					hd.Data.Render(w)

				}
			}()

			hd.Data.URL = r.URL.String()
			hd.Data.Method = r.Method

			if hd.Data.Status == 0 {

				hd.Data.MenuList = make([]models.MenuData, 0, 3)
				hd.Data.MenuList = []models.MenuData{
					0: {"/posts/1", "Блог", false},
					1: {"/post/", "Новая запись", false},
					2: {"/log/1", "Журнал событий", false},
				}
			}

			hd.Data.LastModified = compileDate

			if r.URL.String() == "/" || hd.Data.Status == 0 { // если главная страница или не ошибка
				// передаём data используя контекст и запускаем следуюзий хэндлер если всё ок
				ctx := context.WithValue(r.Context(), "hd", &hd)
				next(w, r.WithContext(ctx), actions)
			} else {
				logger.SubSubLogYellow("Следующий хэндлер исключён")
			}
		}
	}
}

// routes
func (m *Middleware) SetUpHandlers() {
	logger.SubLog("Setting up handlers...")

	// swagger хандлер
	m.router.HandlerFunc("GET", "/swagger/*filepath", httpSwagger.Handler(httpSwagger.URL("/swagger.json")))

	// главная страница
	m.router.GET("/", Adapt(Welcome, InitPage(m.db, m.ctx)))

	// swagger
	m.router.GET("/swagger.json", SwaggerJSON)

	// блог
	m.router.GET("/post/", Adapt(BlogForm, InitPage(m.db, m.ctx)))
	m.router.GET("/post/:action/:id", Adapt(BlogForm, InitPage(m.db, m.ctx)))
	m.router.POST("/editpost/:action/:id", Adapt(ProcessPost, InitPage(m.db, m.ctx)))
	m.router.POST("/post/", Adapt(ProcessPost, InitPage(m.db, m.ctx)))
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
