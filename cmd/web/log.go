package main

import (
	"./logger"
	"./models"

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
	return nil
}

type LogMessage struct {
	ID    uint16
	Event string
	Name  string
	Date  string
}

func ListLog(_ http.ResponseWriter, r *http.Request, action httprouter.Params) {
	hd := r.Context().Value("hd").(*models.HandlerData)

	logger.SubLog("Audit log list is requested")
	hd.Data.Title = "Журнал событий"

	var logList LogList
	var err error

	// обработка данных фильтра
	getParameters := r.URL.Query()
	err = logList.PrepareDateFilter(getParameters)
	if err != nil {
		hd.Data.SetError(400, err)
		return
	}

	logList.Where = logList.Filter.ComposeWhere("")

	currentPage, _ := my.StUint16(action.ByName("page"))

	err = logList.FillOut(currentPage, logList.Filter.getString, (*hd).Db)
	if err != nil {
		hd.Data.SetError(503, err)
		return
	}

	err = logList.getLog((*hd).Db)
	if err != nil {
		hd.Data.SetError(503, err)
		return
	}

	//добавляем данные о ролях в структуру данных
	hd.Data.PageData = logList
	hd.Data.Template = filepath.Join("..", "ui", "html", "log.html") // темплейт страницы
}
