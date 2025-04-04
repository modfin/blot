// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0

package db

type Fragment struct {
	ID              int     `db:"id" json:"id"`
	Label           string    `db:"label" json:"label"`
	Name            string    `db:"name" json:"name"`
	Content         string    `db:"content" json:"content"`
	EmbeddingModel  string    `db:"embedding_model" json:"embedding_model"`
	EmbeddingVector []float64 `db:"embedding_vector" json:"embedding_vector"`
	CreatedAt       int     `db:"created_at" json:"created_at"`
	UpdatedAt       int     `db:"updated_at" json:"updated_at"`
}
