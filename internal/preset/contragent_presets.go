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
				FKField:      "contragent_addresses.contragent_id",
			},
		},
	}



}