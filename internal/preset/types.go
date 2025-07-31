package preset

// Preset описывает структуру пресета для SQL-запросов
type Preset struct {
	Table     string
	Fields    []FieldDef	
}


type FieldDef struct {
	Source       string                          // SQL-путь: "addresses.id", "areas.name" или "relation.preset"
	Alias        string                          // JSON-ключ, по умолчанию = Source
	Type         string                          // "int", "string", "bool", "computed", "belongs_to", "has_many", "has_one", "preset"
	NestedPreset string                          // имя вложенного пресета
	Formatter    func(map[string]any) any        // используется только при Type == "computed"
	PKField     	string // для has_many: поле первичного ключа в основной таблице, если не "id"
	FKField      	string // для has_many: внешний ключ в подтаблице
	Sorts 				[]string // для has_many: сортировка по полям вложенного пресета
	Where					string // условие для фильтрации, например "areas.id = 1"
	Internal			bool // если true, поле не будет возвращаться в ответах API
}



