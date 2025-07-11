package model

type Address struct {
	ID     int    `db:"id"`
	Value  string `db:"value"`
	AreaID int    `db:"area_id"`
}