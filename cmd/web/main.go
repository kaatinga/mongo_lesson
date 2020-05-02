package main

import (
	"context"
	"errors"
	"github.com/fatih/color"
	"github.com/julienschmidt/httprouter"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	myFormatDateTime = "02.01.2006 15:04"
	port             = "3333"

	// все фразы тут
	newPost     = "Создание новой записи в блоге"
	editPost    = "Редактирование записи в блоге"
	deletedPost = "Запись в блоге удалена"
	editedPost  = "Запись в блоге отредактирована"
	addedPost   = "Новая запись добавлена"

	// Ошибки
	nameTooShort      = "Введённая строка слишком короткая"
	incorrectSymbols  = "Неправильное наименование. Допустимы только русские буквы, цифры, пробел и набор символов '&\"+-»«'"
	beginWithDigit    = "Не допускается начинать наименование с цифры"
	limitFrom0To255   = "Значение допустимо в пределах от 0 до 255"
	limitFrom0To65535 = "Значение допустимо в пределах от 0 до 65535"
	usedAlready       = "Введённая строка содержит наименование которое уже используется в системе"
	onlyRussian       = "Только русские буквы и пробел разрешены"
)

var (
	moscow      *time.Location // время
	compileDate string
)

func main() {

	var (
		err    error
		server *Middleware
	)

	// Устанавливаем сдвиг времени
	moscow, _ = time.LoadLocation("Europe/Moscow")

	log.Println("Starting the web server...")



	var ctx context.Context
	ctx = context.Background()

	// объявляем роутер
	server = newMiddleware(httprouter.New(), ctx)

	// Establishing connection to the database
	var client *mongo.Client
	client, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Println("Ошибка установки соединения")
	}

	server.db = client.Database("blog")
	sublog("Connection is established!")

	// анонсируем хандлеры
	SetUpHandlers(server)

	webServer := http.Server{
		Addr:    net.JoinHostPort("", port),
		Handler: server,
		//TLSConfig:         nil,
		ReadTimeout:       1 * time.Minute,
		ReadHeaderTimeout: 15 * time.Second,
		WriteTimeout:      1 * time.Minute,
		//IdleTimeout:       0,
		//MaxHeaderBytes:    0,
		//TLSNextProto:      nil,
		//ConnState:         nil,
		//ErrorLog:          nil,
		//BaseContext:       nil,
		//ConnContext:       nil,
	}

	sublog("Launching the service on the port:", port, "...")
	go func() {
		err = webServer.ListenAndServe()
		if err != nil {
			log.Println(err)
		}
	}()

	subsublog("The server was launched!")

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	<-interrupt

	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()

	err = webServer.Shutdown(timeout)
	if err != nil {
		log.Println(err)
	}
}

type Filter struct {
	DateFilter string
	From       string
	To         string
	getString  string
}

func (filter *Filter) PrepareDateFilter(getParameters url.Values) (err error) {

	from := getParameters.Get("from")
	to := getParameters.Get("to")

	if from == "" && to == "" {
		return nil
	}

	filter.getString = "?"

	if from != "" {
		_, err = time.Parse("2006-01-02", from)
		if err != nil {
			return err
		}
		filter.getString = strings.Join([]string{filter.getString, "from=", from}, "")
		filter.DateFilter = strings.Join([]string{filter.DateFilter, " date>'", from, "'"}, "")
		filter.From = from
	}

	if to != "" && from != "" {
		filter.getString = strings.Join([]string{filter.getString, "&"}, "")
		filter.DateFilter = strings.Join([]string{filter.DateFilter, " AND"}, "")
	}

	if to != "" {
		_, err = time.Parse("2006-01-02", to)
		if err != nil {
			return err
		}
		filter.getString = strings.Join([]string{filter.getString, "to=", to}, "")
		filter.DateFilter = strings.Join([]string{filter.DateFilter, " date<'", to, "'"}, "")
		filter.To = to
	}

	//log.Println(filter.DateFilter)
	//log.Println(filter.From)
	//log.Println(filter.To)

	return nil
}

func (filter *Filter) ComposeWhere(addWhere string) (resultFilter string) {

	// если всё пусто, то всё пусто
	if filter.DateFilter == "" && addWhere == "" {
		return ""
	}

	// иначе начинаем составлять
	resultFilter = "WHERE "

	if filter.DateFilter != "" {
		resultFilter = strings.Join([]string{resultFilter, filter.DateFilter}, "")
	}

	if addWhere != "" {
		if filter.DateFilter != "" {
			resultFilter = strings.Join([]string{resultFilter, " AND "}, "")
		}
		resultFilter = strings.Join([]string{resultFilter, addWhere}, "")
	}

	subsublog("A filter is applied to the list")

	return
}

func setFormCookie(w http.ResponseWriter, cookieName, cookieValue string) {

	subSubLogYellow("Устанавливаем временную сессию формы")
	formCookie := &http.Cookie{
		Name:     cookieName,
		Value:    cookieValue,
		Path:     "/",
		Expires:  time.Now().Add(5 * time.Minute),
		MaxAge:   300,                     // 5 минут
		Secure:   false,                   // yet 'false' as TLS is not used
		HttpOnly: true,                    // 'true' secures from XSS attacks
		SameSite: http.SameSiteStrictMode, // base CSRF attack protection
	}

	http.SetCookie(w, formCookie)
	subSubLogYellow("Сессия формы успешно установлена")
}

func checkFormCookie(w http.ResponseWriter, r *http.Request, cookieName, cookieMustHaveValue string) (string, error) {

	var (
		FormCookie  *http.Cookie
		err         error
		cookieValue string
	)

	FormCookie, err = r.Cookie(cookieName)
	if err != nil {
		return "", err
	}

	subSubLogYellow("A Form Cookie Detected")

	cookieValue, err = url.QueryUnescape(FormCookie.Value)
	if err != nil {
		return "", err
	}

	cookieMustHaveValue, err = url.QueryUnescape(cookieMustHaveValue)
	if err != nil {
		return "", err
	}

	subSubLogYellow("The cookie form ID (after processing) is", cookieValue)
	subSubLogYellow("The cookie form ID (after processing) must be", cookieMustHaveValue)

	if cookieValue != cookieMustHaveValue {
		return "", errors.New("the Form Cookie is incorrect")
	}

	// удаляем принятую куку теперь чтобы защититься от повторного запроса
	deleteCookie := &http.Cookie{
		Name:   cookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, deleteCookie)
	subSubLogYellow("Временная сессия формы удалена")

	return cookieMustHaveValue, nil
}

// Функция исполняет типовые действия в случае ошибки. Вызывается из formRequest() в случае любой ошибки
func (data *ViewData) setError(status uint16, err error) {

	// устанавливаем статус
	data.Status = status

	// записываем ошибку в модель
	if err != nil {
		data.Error = err.Error()

		// выводим ошибку в лог
		color.Set(color.FgHiRed)
		subsublog(data.Error)
		color.Unset()
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
		subsublog("The code is not 200, the status code is", strconv.Itoa(int((*data).Status)))
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

	sublog("Template was used:", (*data).Template)

	tmpl, err = template.ParseFiles(layout, authBlock, (*data).Template)

	if err != nil {
		// Вываливаем в лог кучу хлама для анализа. Нужно переписать и выводить в файл.
		subLogRed(err.Error())
		// Возвращаем ошибку пользователю
		http.Error(w, http.StatusText(500), 500)
		return
	}

	err = tmpl.ExecuteTemplate(w, "base", data)
	if err != nil {
		subLogRed(err.Error())
		http.Error(w, http.StatusText(500), 500)
		return
	}

	subsublog("Ошибки при формировании страницы по шаблону не обнаружено") // если ошибки нет
}

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

// MenuData - Модель данных ссылки на страницу
type MenuData struct {
	URL      string
	Name     string
	Selected bool
}

type Paginator struct {
	Html      string
	Paginator string
	Total     uint16
	Offset    uint16
	Where     string // Будет хранить дополнительную вставку условия Where для реализации фильтра
}

func (p *Paginator) Append(phrase, iString, parameters string, currentPage bool) {
	if phrase == "" {
		phrase = iString
	}

	if currentPage {
		p.Html = strings.Join([]string{(*p).Html, "<page class=currentpage>", phrase, "</page>"}, "")
		return
	}

	p.Html = strings.Join([]string{(*p).Html, "<page>"}, "")

	if iString != "" {
		p.Html = strings.Join([]string{(*p).Html, "<a href=", iString, parameters, ">", phrase, "</a></page>"}, "")
		return
	}

	p.Html = strings.Join([]string{(*p).Html, phrase, "</page>"}, "")

}
