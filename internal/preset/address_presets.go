package preset

func RegisterAddressPresets() {
	Registry["address.card"] = Preset{
		Table: "addresses",
		Fields: []FieldDef{
			{Source: "addresses.id", Alias: "id", Type: "int"},
			{Source: "addresses.value", Alias: "value", Type: "string"},
			{Source: "areas", Alias: "area", Type: "preset", NestedPreset: "area.card"},	
			{
				Type:      "computed",
				Alias:     "sum_id",
				Formatter: func(data map[string]any) any {
					id, _ := data["id"].(int64)
					aid, _ := data["area_id"].(int64)
					return id + aid
				},
			},	
		},
		Joins: []JoinSpec{
			{Type: "LEFT JOIN", Expr: "areas ON areas.id = addresses.area_id"},
		},	
		
	}

	Registry["address.edit"] = Preset{
		Table: "addresses",
		Fields: []FieldDef{
			{Source: "addresses.id", Alias: "id", Type: "int"},
			{Source: "addresses.area_id", Alias: "area_id", Type: "int"},
			{Source: "addresses.value", Alias: "value", Type: "string"},
			{Source: "areas", Alias: "area", Type: "preset", NestedPreset: "area.card"},
		},
		Joins: []JoinSpec{
			{Type: "LEFT JOIN", Expr: "areas ON areas.id = addresses.area_id"},
		},
		
	}
}