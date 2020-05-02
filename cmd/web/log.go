package main

import (
	"github.com/julienschmidt/httprouter"
	my "github.com/kaatinga/assets"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"path/filepath"
)

type LogList struct {
	Messages []LogMessage
	Filter
	Paginator
}

func (logList *LogList) getLog(_ *mongo.Database) (err error) {

	//

	return nil
}

type LogMessage struct {
	ID    uint16
	Event string
	Name  string
	Date  string
}

func (hd *HandlerData) AddToLog(event, author string) {
	//TODO:
}

func ListLog(_ http.ResponseWriter, r *http.Request, action httprouter.Params) {
	hd := r.Context().Value("hd").(*HandlerData)

	sublog("Audit log list is requested")
	hd.data.Title = "Журнал событий"

	var logList LogList
	var err error

	// обработка данных фильтра
	getParameters := r.URL.Query()
	err = logList.PrepareDateFilter(getParameters)
	if err != nil {
		hd.data.setError(400, err)
		return
	}
	//log.Println(logList.DateFilter)

	logList.Where = logList.Filter.ComposeWhere("")

	currentPage, _ := my.StUint16(action.ByName("page"))

	err = logList.FillOut(currentPage, logList.Filter.getString, (*hd).db)
	if err != nil {
		hd.data.setError(503, err)
		return
	}

	err = logList.getLog((*hd).db)
	if err != nil {
		hd.data.setError(503, err)
		return
	}

	//добавляем данные о ролях в структуру данных
	hd.data.PageData = logList
	hd.data.Template = filepath.Join("..", "ui", "html", "log.html") // темплейт страницы
}
