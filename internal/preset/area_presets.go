package preset

func RegisterAreaPresets() {
	Registry["area.card"] = Preset{
		Table: "areas",
		Fields: []FieldDef{
			{Source: "areas.id", Alias: "id", Type: "int"},
			{Source: "areas.name", Alias: "name", Type: "string"},
		},
	}
}