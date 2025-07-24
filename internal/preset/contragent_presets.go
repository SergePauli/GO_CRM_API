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
				Sorts: []string{"address_area_id ASC"},
			},
		},
	}



}