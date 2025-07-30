package preset

func RegisterContragentAddressesPresets() {
	Registry["contragent.addresses"] = Preset{
		Table: "contragents",
		Fields: []FieldDef{
			{Source: "contragents.id", Alias: "id", Type: "int"},

			// Связь один-ко-многим: массив адресов
			{		
				Alias:        "contragent_addresses",
				Type:         "has_many",
				NestedPreset: "contragent_address.edit", // отдельный вложенный пресет
				PKField: "id",
				FKField: "contragent_id",
				Internal: true, // скрываем FKField в ответах API
				Sorts: []string{"address_area_id ASC"},
			},
			// Связь один-к-один 
			{		
				Alias:        "real_address",
				Type:         "has_one",
				NestedPreset: "contragent_address.card", // отдельный вложенный пресет
				PKField: "id",
				FKField: "contragent_id",
				Where: "real_address.used = true AND real_address.kind = 0", 
			},
		},
	}



}