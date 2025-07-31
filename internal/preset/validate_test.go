package preset

import "testing"

// Заглушка: простой набор пресетов с циклом
func initFakeCyclicPresets() {
	Registry = map[string]Preset{
		"a": {
			Fields: []FieldDef{
				{Type: "belongs_to", NestedPreset: "b"},
			},
		},
		"b": {
			Fields: []FieldDef{
				{Type: "belongs_to", NestedPreset: "c"},
			},
		},
		"c": {
			Fields: []FieldDef{
				{Type: "belongs_to", NestedPreset: "a"}, // цикл!
			},
		},
		"safe": {
			Fields: []FieldDef{
				{Type: "belongs_to", NestedPreset: "leaf"},
			},
		},
		"leaf": {
			Fields: []FieldDef{
				{Type: "string", Source: "name"},
			},
		},
	}
}

func TestPresetCycles(t *testing.T) {
	initFakeCyclicPresets()

	ValidateAndPrunePresets()

	if _, ok := Registry["a"]; ok {
		t.Errorf("Preset 'a' should have been removed due to cycle")
	}
	if _, ok := Registry["b"]; ok {
		t.Errorf("Preset 'b' should have been removed due to cycle")
	}
	if _, ok := Registry["c"]; ok {
		t.Errorf("Preset 'c' should have been removed due to cycle")
	}

	if _, ok := Registry["safe"]; !ok {
		t.Errorf("Preset 'safe' should NOT have been removed")
	}
	if _, ok := Registry["leaf"]; !ok {
		t.Errorf("Preset 'leaf' should NOT have been removed")
	}
}
