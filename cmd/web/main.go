package main

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"mongo/logger"
	"mongo/models"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/julienschmidt/httprouter"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	myFormatDateTime = "02.01.2006 15:04"

	// все фразы тут
	newPost  = "Создание новой записи в блоге"
	editPost = "Редактирование записи в блоге"

	// Ошибки
	nameTooShort = "Введённая строка слишком короткая"
	onlyRussian  = "Только русские буквы и пробел разрешены"
)

var (
	moscow      *time.Location // время
	compileDate string
)

func main() {

	logger.LoggerInit(os.Stdout) // указываем куда выводить логи

	// Getting environment settings for the network port and all the necessary paths
	logger.Info.Println("Loading environment settings...")

	var myEnvs EnvironmentSettings
	myEnvs.load() // Все ошибки проверяются внутри и завершают программу если что не так

	var (
		err    error
		server *Middleware
	)

	// Устанавливаем сдвиг времени
	moscow, _ = time.LoadLocation("Europe/Moscow")

	logger.Info.Println("Starting the web server...")

	var ctx context.Context
	ctx = context.Background()

	// объявляем роутер
	server = newMiddleware(httprouter.New(), ctx)

	// Establishing connection to the database
	var client *mongo.Client
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(myEnvs.MongoAddress))
	if err != nil {
		logger.SubInfo.Println("Ошибка установки соединения")
		os.Exit(1)
	}

	server.db = client.Database("blog")
	logger.SubInfo.Println("Connection is established!")

	// анонсируем хандлеры
	server.SetUpHandlers()

	webServer := http.Server{
		Addr:              net.JoinHostPort("", myEnvs.Port),
		Handler:           server,
		ReadTimeout:       1 * time.Minute,
		ReadHeaderTimeout: 15 * time.Second,
		WriteTimeout:      1 * time.Minute,
	}

	logger.SubInfo.Println("Launching the service on the port:", myEnvs.Port, "...")
	go func() {
		err = webServer.ListenAndServe()
		if err != nil {
			logger.Info.Println(err)
		}
	}()

	logger.SubSubInfo.Println("The server was launched!")

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	<-interrupt

	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()

	err = webServer.Shutdown(timeout)
	if err != nil {
		logger.Info.Println(err)
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

	logger.SubSubInfo.Println("A filter is applied to the list")

	return
}

func setFormCookie(w http.ResponseWriter, cookieName, cookieValue string) {

	logger.Yellow(logger.SubSubInfo, "Устанавливаем временную сессию формы")
	formCookie := &http.Cookie{
		Name:     cookieName,
		Value:    cookieValue,
		Path:     "/",
		MaxAge:   300,                     // 5 минут
		Secure:   false,                   // yet 'false' as TLS is not used
		HttpOnly: true,                    // 'true' secures from XSS attacks
		SameSite: http.SameSiteStrictMode, // base CSRF attack protection
	}

	http.SetCookie(w, formCookie)
	logger.Yellow(logger.SubSubInfo, "Сессия формы успешно установлена")
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

	logger.Yellow(logger.SubSubInfo, "A Form Cookie Detected")

	cookieValue, err = url.QueryUnescape(FormCookie.Value)
	if err != nil {
		return "", err
	}

	cookieMustHaveValue, err = url.QueryUnescape(cookieMustHaveValue)
	if err != nil {
		return "", err
	}

	logger.Yellow(logger.SubSubInfo, strings.Join([]string{"The cookie form ID (after processing) is", cookieValue}, ""))
	logger.Yellow(logger.SubSubInfo, strings.Join([]string{"The cookie form ID (after processing) must be", cookieMustHaveValue}, ""))

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
	logger.Yellow(logger.SubSubInfo, "Временная сессия формы удалена")

	return cookieMustHaveValue, nil
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

func GetPost(ctx context.Context, db *mongo.Database, id primitive.ObjectID) (*models.Post, error) {
	var p models.Post
	coll := db.Collection(p.GetMongoCollectionName())
	res := coll.FindOne(ctx, bson.M{"_id": id})
	if err := res.Decode(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

// Модель данных для параметров окружения
type EnvironmentSettings struct {
	Port         string `env:"PORT"`
	MongoAddress string `env:"MONGO_ADDR"`
}

func (e *EnvironmentSettings) load() {
	t := reflect.TypeOf(*e)        // объект для работы с типами
	v := reflect.ValueOf(e).Elem() // объект для работы со значениями

	numberOfFields := t.NumField() // вынимаем размер структуры
	logger.SubInfo.Println("Количество полей в структуре:", strconv.Itoa(numberOfFields))

	var field, tag string // временные переменные для цикла обработки параметров окружения ниже

	for i := 0; i < numberOfFields; i++ {
		field = t.Field(i).Name // имя поля
		logger.SubInfo.Println("Текущее название поля в структуре EnvironmentSettings:", field)
		tag = t.Field(i).Tag.Get("env") // значение тега env
		logger.SubInfo.Println("Переменная окружения:", tag)

		envPar, ok := os.LookupEnv(tag)
		if !ok {
			logger.Red(logger.SubInfo, strings.Join([]string{"Ошибка чтения параметров окружения:", field}, ""))
			os.Exit(1)
		}
		logger.SubInfo.Println("Значение:", envPar)

		v.FieldByName(field).SetString(envPar)
	}
}
