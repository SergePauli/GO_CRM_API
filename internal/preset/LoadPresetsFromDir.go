package preset

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

//

type Model struct {
	Table   string   `yaml:"table"`
	Presets []ImportPreset `yaml:"presets"`
}

type ImportPreset struct {
	Name   string     `yaml:"name"`   // example: "contragent.card"
	Fields []ImportFieldDef `yaml:"fields"` // fields in this preset
}

type ImportFieldDef struct {
	Source       string   `yaml:"source"`        // example: "addresses.id"
	Alias        string   `yaml:"alias"`         // optional override
	Type         string   `yaml:"type"`          // "int", "string", "has_many", "belongs_to", etc.
	NestedPreset string   `yaml:"nested_preset"` // name of another preset
	Select       string   `yaml:"select"`        // for computed SQL expression
	PKField      string   `yaml:"pk_field"`
	FKField      string   `yaml:"fk_field"`
	Sorts        []string `yaml:"sorts"`
	Where        string   `yaml:"where"`
	Internal     bool     `yaml:"internal"`
}

func canRegisterPreset(p Preset) bool {
	for _, f := range p.Fields {
		if isNestedField(f.Type) && f.NestedPreset != "" {
			if _, ok := Registry[f.NestedPreset]; !ok {
				return false
			}
		}
	}
	return true
}

func isNestedField(fieldType string) bool {
	switch fieldType {
	case "has_one", "has_many", "belongs_to":
		return true
	default:
		return false
	}
}

func convertImportPreset(table string, imp ImportPreset) Preset {
	fields := make([]FieldDef, 0, len(imp.Fields))
	for _, f := range imp.Fields {
		fields = append(fields, FieldDef{
			Source:       f.Source,
			Alias:        f.Alias,
			Type:         f.Type,
			NestedPreset: f.NestedPreset,
			Select:       f.Select,
			PKField:      f.PKField,
			FKField:      f.FKField,
			Sorts:        f.Sorts,
			Where:        f.Where,
			Internal:     f.Internal,
		})
	}

	return Preset{
		Table:  table,
		Fields: fields,
	}
}

type RegistryMap map[string]Preset // key: "contragent.card"



// LoadPresetsFromDir загружает пресеты из указанной директории

func LoadPresetsFromDir(dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.yml"))
	if err != nil {
		return fmt.Errorf("failed to read preset dir: %w", err)
	}

	candidates := map[string]Preset{}
	tableNames := map[string]string{} // shortName → fullTableName

	for _, file := range files {
		ymlData, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", file, err)
		}

		var model Model
		if err := yaml.Unmarshal(ymlData, &model); err != nil {
			return fmt.Errorf("failed to parse YAML %s: %w", file, err)
		}

		// Extract filename base (without .yml)
		base := strings.TrimSuffix(filepath.Base(file), ".yml")
		tableNames[base] = model.Table

		for _, p := range model.Presets {
			key := fmt.Sprintf("%s.%s", base, p.Name)

			for i := range p.Fields {
				// Optional enhancement: infer Source
				if p.Fields[i].Source == "" && p.Fields[i].Alias != "" {
					p.Fields[i].Source = fmt.Sprintf("%s.%s", model.Table, p.Fields[i].Alias)
				}
			}

			candidates[key] = convertImportPreset(model.Table, p)
				
		}
	}

	// Регистрируем с учётом зависимостей
	registered := true
	round := 0
	for registered {
		registered = false
		for name, preset := range candidates {
			if canRegisterPreset(preset) {
				Registry[name] = preset
				delete(candidates, name)
				log.Printf("✅ Registered: %s", name)
				registered = true
			}
		}
		round++
		log.Printf("⏳ Round %d: %d candidates remaining", round, len(candidates))
	}

	if len(candidates) > 0 {
		for name := range candidates {
			log.Printf("⚠️ Cannot register: %s — unresolved dependency", name)
		}
		return fmt.Errorf("some presets could not be registered")
	}

	return nil
}