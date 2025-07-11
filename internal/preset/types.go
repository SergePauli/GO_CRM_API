package preset

// Preset описывает структуру пресета для SQL-запросов
type Preset struct {
	Table     string
	Fields    []FieldDef
	Joins     []JoinSpec	
}

type FieldDef struct {
	Source       string                          // SQL-путь: "addresses.id", "areas.name" или "relation.preset"
	Alias        string                          // JSON-ключ, по умолчанию = Source
	Type         string                          // "int", "string", "bool", "computed", "array"
	NestedPreset string                          // имя вложенного пресета
	Formatter    func(map[string]any) any        // используется только при Type == "computed"
}

type JoinSpec struct {
	Type string // "JOIN", "LEFT JOIN", "RIGHT JOIN"
	Expr string // "areas ON areas.id = addresses.area_id"
}
