package db

import (
	"context"
	"fmt"
	"github.com/modfin/blot/internal/db/vec"
)

func (q *Queries) AddFragment(
	ctx context.Context,
	label string,
	name string,
	content string,
	embeddingModel string,
	embeddingVector []float64,
) (Fragment, error) {

	const addFragment = `
INSERT INTO fragments (label, name, content, embedding_model, embedding_vector)
VALUES (?, ?, ?, ?, ?) 
ON CONFLICT (label, name) DO 
	UPDATE 
    SET content = excluded.content, 
		embedding_model = excluded.embedding_model,
		embedding_vector = excluded.embedding_vector
RETURNING id, label, name, content, embedding_model, embedding_vector, created_at, updated_at
`

	row := q.db.QueryRowContext(ctx, addFragment,
		label,
		name,
		content,
		embeddingModel,
		vec.EncodeVector(embeddingVector),
	)

	var i Fragment
	var vecbin []byte
	err := row.Scan(
		&i.ID,
		&i.Label,
		&i.Name,
		&i.Content,
		&i.EmbeddingModel,
		&vecbin,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	if err != nil {
		return Fragment{}, fmt.Errorf("insert fragment: %w", err)
	}
	i.EmbeddingVector, err = vec.DecodeVector(vecbin)
	if err != nil {
		return Fragment{}, fmt.Errorf("decoding embedding vector: %w", err)
	}
	return i, err
}

func (q *Queries) DirtyFragment(ctx context.Context, label string, name string, content string) (bool, error) {

	const dirty = `
	SELECT count(*) = 0
	FROM fragments
	WHERE label = ? AND name = ? AND content = ?
`

	row := q.db.QueryRowContext(ctx, dirty,
		label,
		name,
		content,
	)
	var i bool
	if err := row.Scan(&i); err != nil {
		return false, err
	}
	return i, nil

}

func (q *Queries) KNN(ctx context.Context, vector []float64, label string, limit int) ([]Fragment, error) {

	const kNN = `
SELECT id, label, name, content, embedding_model, embedding_vector, created_at, updated_at
FROM fragments
WHERE label like ?
ORDER BY vec_dist(?, embedding_vector)
LIMIT ?
`

	rows, err := q.db.QueryContext(ctx, kNN,
		label,
		vec.EncodeVector(vector),
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Fragment
	for rows.Next() {
		var i Fragment
		var vecbytes []byte
		if err := rows.Scan(
			&i.ID,
			&i.Label,
			&i.Name,
			&i.Content,
			&i.EmbeddingModel,
			&vecbytes,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}

		i.EmbeddingVector, err = vec.DecodeVector(vecbytes)
		if err != nil {
			return nil, fmt.Errorf("failed decoding embedding vector: %w", err)
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (q *Queries) ListFragments(ctx context.Context) ([]Fragment, error) {

	const listFragments = `
SELECT id, label, name, content, embedding_model, embedding_vector, created_at
FROM fragments
ORDER BY id
`

	rows, err := q.db.QueryContext(ctx, listFragments)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Fragment
	for rows.Next() {
		var i Fragment
		if err := rows.Scan(
			&i.ID,
			&i.Label,
			&i.Name,
			&i.Content,
			&i.EmbeddingModel,
			&i.EmbeddingVector,
			&i.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
