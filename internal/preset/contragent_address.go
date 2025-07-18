package preset

func RegisterContragentAddressPresets() {
	Registry["contragent_address.edit"] =  Preset{
		Table: "contragent_addresses",
		Fields: []FieldDef{
			{Source: "contragent_addresses.id", Alias: "id", Type: "int"},
			{Source: "contragent_addresses.used", Alias: "used", Type: "bool"},
			{Source: "contragent_addresses.contragent_id", Alias: "contragent_id", Type: "int"},
			{Source: "contragent_addresses.kind", Alias: "kind", Type: "string"},
			{Source: "addresses", Alias: "address", Type: "preset", NestedPreset: "address.edit"},
		},
		Joins: []JoinSpec{
			{Type: "LEFT JOIN", Expr: "addresses ON addresses.id = contragent_addresses.address_id"},
		},
	}
}
