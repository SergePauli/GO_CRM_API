package preset

import (
	"strings"
	"testing"
)


func RegisterTestPresets() map[string]Preset{
	Registry = map[string]Preset{
		"parent": {
			Table: "parents",
			Fields: []FieldDef{
				{Source: "parents.id", Alias: "id", Type: "int"},
				{Alias: "children", Type: "has_many", NestedPreset: "child"},
			},
		},
		"parents.card": {
			Table: "parents",
			Fields: []FieldDef{
				{Source: "parents.id", Alias: "id", Type: "int"},
				{Alias: "children", Type: "has_many", NestedPreset: "children.with_name"},
			},
			Joins: []JoinSpec{},
		},
		"children.with_name": {
			Table: "children",
			Fields: []FieldDef{
				{Source: "children.name", Alias: "child_name", Type: "string"},
			},
			Joins: []JoinSpec{
				{Type: "LEFT JOIN", Expr: "children ON children.parent_id = parents.id"},
			},
		},		
		"toy": {
			Table: "toys",
			Fields: []FieldDef{
				{Source: "toys.child_id", Alias: "child_id", Type: "int"},
				{Source: "toys.name", Alias: "toy_name", Type: "string"},
			},
			Joins: []JoinSpec{
				{Type: "LEFT JOIN", Expr: "toys ON toys.child_id = children.id"},
			},
		},
		"parents.deep": {
			Table: "parents",
			Fields: []FieldDef{
				{Source: "parents.id", Alias: "id", Type: "int"},
				{Alias: "children", Type: "has_many", NestedPreset: "children.with_toys"},
			},
		},
		"children.with_toys": {
			Table: "children",
			Fields: []FieldDef{
				{Source: "children.id", Alias: "id", Type: "int"},
				{Source: "children.name", Alias: "child_name", Type: "string"},
				{Alias: "toys", Type: "has_many", NestedPreset: "toy"},
			},
			Joins: []JoinSpec{
				{Type: "LEFT JOIN", Expr: "children ON children.parent_id = parents.id"},
			},
		},
	}
	return Registry
}

func TestBuildQueryAddsDistinctOnHasManyFilter(t *testing.T) {
	// Регистрация тестовых пресетов
	Registry = RegisterTestPresets()

	filters := map[string]interface{}{
		"children_child_name__cnt": "foo",
	}

	// Выполняем построение SQL-запроса
	q,_ := Registry["parents.card"].BuildQuery(filters, nil, 0, 10)
	sqlStr, _, err := q.ToSql()
	if err != nil {
		t.Fatalf("Failed to build SQL: %v", err)
	}

	if !strings.Contains(sqlStr, "DISTINCT") {
		t.Errorf("Expected DISTINCT in query when filtering has_many, got: %s", sqlStr)
	}
}

func TestBuildQueryAddsDistinctOnHasManySort(t *testing.T) {
	Registry = RegisterTestPresets()

	sorts := []string{"children_child_name ASC"}
	query,_ := Registry["parents.card"].BuildQuery(nil, sorts, 0, 0)
	sql, _, err := query.ToSql()
	if err != nil {
		t.Fatalf("failed to generate SQL: %v", err)
	}

	if !strings.Contains(sql, "DISTINCT") {
		t.Errorf("expected DISTINCT in SQL when sorting on has_many, got: %s", sql)
	}
}
func TestBuildQueryAddsDistinctOnNestedHasMany(t *testing.T) {
	Registry = RegisterTestPresets()
	filters := map[string]interface{}{
		"children_toys_toy_name__cnt": "ball",
	}
	query,_ := Registry["parents.deep"].BuildQuery(filters, nil, 0, 0)
	sql, _, err := query.ToSql()
	if err != nil {
		t.Fatalf("failed to generate SQL: %v", err)
	}

	if !strings.Contains(sql, "DISTINCT") {
		t.Errorf("expected DISTINCT in SQL when filtering on nested has_many, got: %s", sql)
	}
}