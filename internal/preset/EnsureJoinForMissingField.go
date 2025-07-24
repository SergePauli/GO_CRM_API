package preset

import (
	"log"
	"strings"

	"github.com/Masterminds/squirrel"
)

func EnsureJoinForMissingField(
	fieldName string,
	builder squirrel.SelectBuilder,
	preset Preset,
	prefix string,
	aliasToSource map[string]string,
) squirrel.SelectBuilder {
	parentAlias := strings.TrimSuffix(prefix, "_")

	for _, f := range preset.Fields {
		alias := f.Alias
		if alias == "" {
			alias = f.Source
		}
		fullAlias := prefix + alias

		switch {
		case f.Type == "preset" && f.NestedPreset != "":
			if strings.HasPrefix(fieldName, fullAlias+"_") {
				nested, err := GetPreset(f.NestedPreset)
				if err != nil {
					log.Printf("Invalid nested preset: %s", f.NestedPreset)
					return builder
				}

				nestedAlias := fullAlias
				if _, exists := aliasToSource[nestedAlias]; !exists {
					parentFK := f.FKField
					if parentFK == "" {
							parentFK = alias + "_id" // или fullAlias + "_id", если уникальность нужна
					}
					nestedPK := f.PKField
					if nestedPK == "" {
						nestedPK = "id"
					}
					nestedTable := nested.Table
					if nestedTable == "" {
						log.Printf("Nested preset %s has no table", f.NestedPreset)
						return builder
					}

					joinExpr := nestedTable + " AS " + nestedAlias +" ON " + parentAlias + "." + parentFK + " = " + nestedAlias + "." + nestedPK
					builder = builder.LeftJoin(joinExpr)
					aliasToSource[nestedAlias] = nestedAlias
				}

				// Рекурсивно для вложенного пресета
				builder = EnsureJoinForMissingField(fieldName, builder, nested, fullAlias+"_", aliasToSource)
			}

		case f.Type == "has_many" && f.NestedPreset != "":
			if parentAlias == "" {
					parentAlias = preset.Table
			}
			if strings.HasPrefix(fieldName, fullAlias+"_") {
				nested, err := GetPreset(f.NestedPreset)
				if err != nil {
					log.Printf("Invalid has_many nested preset: %s", f.NestedPreset)
					return builder
				}

				nestedAlias := fullAlias
				if _, exists := aliasToSource[nestedAlias]; !exists {
					parentPK := f.PKField
					if parentPK == "" {
						parentPK = "id"
					}
					nestedFK := f.FKField
					if nestedFK == "" {
						nestedFK = preset.Table + "_id"
					}
					nestedTable := nested.Table
					if nestedTable == "" {
						log.Printf("Nested has_many preset %s has no table", f.NestedPreset)
						return builder
					}

					joinExpr := nestedTable + " AS " + nestedAlias + " ON " + nestedAlias + "." + nestedFK + " = " + parentAlias + "." + parentPK
					builder = builder.LeftJoin(joinExpr)
					aliasToSource[nestedAlias] = nestedAlias
				}

				// Рекурсивно обрабатываем вложенный has_many
				builder = EnsureJoinForMissingField(fieldName, builder, nested, fullAlias+"_", aliasToSource)
			}

		default:
			// fieldName должен совпадать с alias поля
			if fieldName == fullAlias && f.Source != "" {
				// Определим, нужно ли подставить alias
				if !strings.Contains(f.Source, ".") {
					aliasToSource[fullAlias] = parentAlias + "." + f.Source
				} else {
					// Разбиваем f.Source на table и column
					parts := strings.SplitN(f.Source, ".", 2)
					if len(parts) == 2 {
						aliasToSource[fullAlias] = parentAlias + "." + parts[1]
					} else {
						aliasToSource[fullAlias] = parentAlias + "." + f.Source
					}		
				}
			}	
		}	
	}

	return builder
}
