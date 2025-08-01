package preset

import (
	"fmt"
	"log"
	"strings"
)

var Registry = map[string]Preset{}
// RegisterPresets регистрирует все пресеты в реестре
func InitAllPresets(presetDir string) {
	err := LoadPresetsFromDir(presetDir)
	if err != nil {
		log.Fatalf("Failed to load presets: %v", err)
	}
	ValidateAndPrunePresets() // <- проверка на циклы
}
// EnsurePresetLoaded проверяет, что пресет с данным именем загружен
func EnsurePresetLoaded(name string) error {	
	if _, ok := Registry[name]; !ok {
		return fmt.Errorf("preset %q still not found", name)
	}
	return nil
}
// GetPreset возвращает пресет по ключу
func GetPreset(key string) (Preset, error) {
	p, ok := Registry[key]
	if !ok {
		return Preset{}, fmt.Errorf("preset %q not found", key)
	}
	return p, nil
}

// HasHasManyRelations проверяет, есть ли в пресете поля типа "has_many"
// Это нужно для оптимизации запросов и избежания лишних JOIN'ов
func (p Preset) HasHasManyRelations() bool {
	for _, f := range p.Fields {
		if f.Type == "has_many" {
			return true
		}
	}
	return false
}

// Определяет, есть ли в фильтрах поля, лежащие в has_many ветках
func FiltersTouchHasMany(filters map[string]interface{}, aliasToHasMany map[string]bool) bool {
	for rawKey := range filters {
		field := strings.SplitN(rawKey, "__", 2)[0]
		if aliasToHasMany[field] {
			return true
		}
	}
	return false
}

// ValidateAndPrunePresets проверяет все пресеты на циклы и удаляет некорректные
// Если пресет ссылается на себя или есть циклическая зависимость, он удаляется из реестра
// Логирует предупреждения о найденных циклах
func ValidateAndPrunePresets() {
	invalid := map[string]bool{}

	for name := range Registry {
		visited := map[string]bool{}
		if hasCycle(name, visited, []string{}) {
			invalid[name] = true
		}
	}

	// Удаление после обхода всех пресетов
	for name := range invalid {
		delete(Registry, name)
		log.Printf("⚠️ Removed cyclic preset: %s", name)
	}
}

func hasCycle(name string, visited map[string]bool, path []string) bool {
	if visited[name] {
		log.Printf("⛔ Cycle detected: %s", strings.Join(append(path, name), " → "))
		return true
	}
	visited[name] = true

	p, ok := Registry[name]
	if !ok {
		log.Printf("⚠️ Preset not found: %s", name)
		return false
	}

	for _, f := range p.Fields {
		if (f.Type == "belongs_to" || f.Type == "has_many") && f.NestedPreset != "" {
			if hasCycle(f.NestedPreset, copyMap(visited), append(path, name)) {
				return true
			}
		}
	}
	return false
}
// copyMap создает глубокую копию карты visited, чтобы избежать мутаций
// Это нужно, чтобы не нарушать логику обхода при рекурсивных вызовах
// Используется в hasCycle для сохранения состояния visited на каждом уровне рекурсии
// Возвращает новую карту с теми же ключами и значениями
// Это позволяет избежать проблем с изменением visited в разных ветках рекурсии
// и гарантирует, что каждый вызов будет работать с чистым состоянием visited.
func copyMap(src map[string]bool) map[string]bool {
	dst := make(map[string]bool)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
