package main

import (
	"go.mongodb.org/mongo-driver/mongo"
	"strconv"
)

func (p *Paginator) FillOut(currentPage uint16, parameters string, _ *mongo.Database) (err error) {

	//TODO: заглушка
	p.Total = 20

	// подготовка данных для пагинации
	if p.Total > 20 {
		maxPages := p.Total / 20

		// добавляем единичку если есть хвост
		if p.Total%20 > 0 {
			maxPages++
		}

		// проверка допустимого диапазона чисел и исправление currentPage
		if currentPage < 1 {
			currentPage = 1
		} else if currentPage > maxPages {
			currentPage = maxPages
		}

		// если текущая страница больше одного, тогда пересчитываем Offset
		if currentPage > 1 {
			p.Offset = (currentPage * 20) - 20
		}

		// генерируем пагинацию
		if currentPage != 1 {
			p.Append("Предыдущая", strconv.Itoa(int(currentPage-1)), parameters, false)
		}

		for i := 1; i <= int(maxPages); i++ {
			if maxPages > 12 && i == 6 {
				p.Append("...", "", parameters, false)
			}

			if maxPages > 6 && i > 5 && i < int(maxPages-4) {
				continue
			}

			p.Append("", strconv.Itoa(i), parameters, i == int(currentPage))
		}

		if currentPage != maxPages {
			p.Append("Следующая", strconv.Itoa(int(currentPage+1)), parameters, false)
		}
	}
	return // завершаем работу если страниц 1
}
