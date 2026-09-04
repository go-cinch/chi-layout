package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"{{ .Computed.module_name_final }}/internal/infra/db"
	"{{ .Computed.module_name_final }}/internal/modules"
)

var ErrNotFound = errors.New("user not found")

type User struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
}

type Module struct {
	store *db.Store
}

var _ modules.Module = (*Module)(nil)

func New(store *db.Store) *Module {
	return &Module{store: store}
}

func (*Module) Name() string {
	return "/users"
}

func (m *Module) Find(ctx context.Context, id int64) (*User, error) {
	const query = `
SELECT id, created_at, updated_at, name, email
FROM t_user
WHERE id = {{ if eq .Computed.database_driver_final "mysql" }}?{{ else }}$1{{ end }}`

	var value User
	err := m.store.SQL(ctx).QueryRowContext(ctx, query, id).Scan(
		&value.ID,
		&value.CreatedAt,
		&value.UpdatedAt,
		&value.Name,
		&value.Email,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}
	return &value, nil
}
