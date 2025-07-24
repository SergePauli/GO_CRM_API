package preset

import (
	"strings"
)

// CheckedSorts проверяет, что сортировка не содержит полей, которые начинаются с алиасов в targetFields
// Возвращает отфильтрованный список сортировок
func CheckedSorts(sorts []string, targetFields map[string]*FieldDef) []string {
	if len(targetFields) == 0 {
		return sorts
	}
	var filtered []string
	for _, sort := range sorts {
		parts := strings.Fields(sort)
		if len(parts) == 0 {
			continue
		}
		field := parts[0]
		// Проверяем, начинается ли поле с одного из алиасов в targetFields
		isDenyed := false
		for prefix := range targetFields {			
			if strings.HasPrefix(field, prefix+"_") {		
				isDenyed = true		
				break
			} 
		}	
		// Если поле не запрещено, добавляем его в отфильтрованный список
		if !isDenyed 	{filtered = append(filtered, sort)}
	}

	return filtered
}
