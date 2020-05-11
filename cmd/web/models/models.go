package models

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

