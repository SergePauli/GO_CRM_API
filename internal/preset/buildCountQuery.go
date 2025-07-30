package preset

import (
	"log"
	"strings"

	"github.com/Masterminds/squirrel"
)

// collectJoinsAndAliasMap собирает JOIN-ы и алиасы для пресета
// Возвращает обновленный builder и карту alias -> source
// Используется для построения SQL-запросов с учетом JOIN-ов и алиас
// В отличие от collectColumnsAndJoins, не добавляет колонки в запрос
func collectJoinsAndAliasMap(builder squirrel.SelectBuilder, preset Preset, prefix string) (squirrel.SelectBuilder, map[string]string) {
	aliasToSource := make(map[string]string)

	

	// Теперь обрабатываем поля
	for _, f := range preset.Fields {
		switch {
		case f.Type == "preset" && f.NestedPreset != "":
			nested, err := GetPreset(f.NestedPreset)
			if err != nil {
				log.Printf("⚠️ Invalid nested preset: %s", f.NestedPreset)
				continue
			}
			nestedPrefix := prefix + f.Alias + "_"

			var nestedMap map[string]string
			builder, nestedMap = collectJoinsAndAliasMap(builder, nested, nestedPrefix)

			for k, v := range nestedMap {
				aliasToSource[k] = v
			}

		case f.Type != "computed":
			alias := f.Alias
			if alias == "" {
				alias = f.Source
			}
			fullAlias := prefix + alias
			aliasToSource[fullAlias] = f.Source
		}
	}

	return builder, aliasToSource
}

// BuildCountQuery создает SQL-запрос для подсчета количества записей в таблице пресета
// с учетом фильтров. Возвращает Squirrel SelectBuilder для дальнейшей настройки.
// Фильтры должны быть в формате "field__op": value, где op - это операция сравнения
// (eq, in, lt, lte, gt, gte, start, end, cnt).
// Например: "name__eq": "John", "age__gt": 30
// Возвращает ошибку, если пресет не найден или некорректен.
func (p Preset) BuildCountQuery(filters map[string]interface{}) (squirrel.SelectBuilder, error) {
	builder := squirrel.Select("COUNT(*)").PlaceholderFormat(squirrel.Dollar).From(p.Table)

	
	// Получаем JOIN-ы и alias → source
	var aliasToSource map[string]string
	builder, aliasToSource = collectJoinsAndAliasMap(builder, p, "")
	// WHERE ...
	for rawKey, val := range filters {
		parts := strings.SplitN(rawKey, "__", 2)
		fieldName := parts[0]
		op := "eq"
		if len(parts) == 2 {
			op = parts[1]
		}

		col, ok := aliasToSource[fieldName]
		if !ok {
			log.Printf("⚠️ Unknown filter field: %s", fieldName)
			continue
		}

		switch op {
		case "eq":
			builder = builder.Where(squirrel.Eq{col: val})
		case "in":
			builder = builder.Where(squirrel.Eq{col: val})
		case "lt":
			builder = builder.Where(squirrel.Lt{col: val})
		case "lte":
			builder = builder.Where(squirrel.LtOrEq{col: val})
		case "gt":
			builder = builder.Where(squirrel.Gt{col: val})
		case "gte":
			builder = builder.Where(squirrel.GtOrEq{col: val})
		case "start":
			if s, ok := val.(string); ok {
				builder = builder.Where(squirrel.Like{col: s + "%"})
			}
		case "end":
			if s, ok := val.(string); ok {
				builder = builder.Where(squirrel.Like{col: "%" + s})
			}
		case "cnt":
			if s, ok := val.(string); ok {
				builder = builder.Where(squirrel.Like{col: "%" + s + "%"})
			}
		default:
			log.Printf("⚠️ Unknown filter op: %s", op)
		}
	}

	return builder, nil
}
