package models

type Provider struct {
	ID        string `db:"id"`
	CreatedAt string `db:"created_at"`
	UpdatedAt string `db:"updated_at"`
	Name      string `db:"name"`
	Provider  string `db:"provider"`
	Config    string `db:"config"`
	IsDefault bool   `db:"is_default"`
}
