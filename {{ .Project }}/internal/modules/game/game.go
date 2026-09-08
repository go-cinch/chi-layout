package game

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
{{- if ne .Computed.database_driver_final "mysql" }}
	"strconv"
{{- end }}
	"strings"
	"time"
	"unicode/utf8"

	"{{ .Computed.module_name_final }}/internal/common/pagination"
	"{{ .Computed.module_name_final }}/internal/infra/db"
)

var (
	ErrNotFound = errors.New("game not found")
	ErrIDs      = errors.New("provide 1-100 positive game ids")
	ErrInvalid  = errors.New("invalid game: positive id and name of 1-100 characters required")
)

type Game struct {
	ID          int64  `json:"id"`
	CreatedAt   int64  `json:"createdAt"`
	UpdatedAt   int64  `json:"updatedAt"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Input struct {
	Name        string `json:"name" example:"Demo Game"`
	Description string `json:"description" example:"A simple game"`
}

// UpdateInput distinguishes omitted fields from explicitly supplied empty values.
type UpdateInput struct {
	Name        *string `json:"name,omitempty" example:"Updated Game"`
	Description *string `json:"description,omitempty" example:"Updated description"`
}

type Module struct {
	store  *db.Store
	limits pagination.Limits
}

func New(store *db.Store, limits pagination.Limits) *Module {
	return &Module{store: store, limits: limits}
}

func (m *Module) Get(ctx context.Context, id int64) (*Game, error) {
	if id <= 0 {
		return nil, ErrInvalid
	}
	const query = `SELECT id, created_at, updated_at, name, description FROM t_game WHERE id = {{ if eq .Computed.database_driver_final "mysql" }}?{{ else }}$1{{ end }}`
	var value Game
	var createdAt, updatedAt time.Time
	err := m.store.SQL(ctx).QueryRowContext(ctx, query, id).Scan(&value.ID, &createdAt, &updatedAt, &value.Name, &value.Description)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query game: %w", err)
	}
	value.CreatedAt = createdAt.UnixMilli()
	value.UpdatedAt = updatedAt.UnixMilli()
	return &value, nil
}

func (m *Module) Create(ctx context.Context, input Input) (*Game, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || utf8.RuneCountInString(input.Name) > 100 {
		return nil, ErrInvalid
	}
	var value *Game
	err := m.store.Tx(ctx, func(ctx context.Context) error {
		var id int64
		now := time.Now().UTC().Truncate(time.Microsecond)
{{ if eq .Computed.database_driver_final "mysql" }}
		result, err := m.store.SQL(ctx).ExecContext(ctx, `INSERT INTO t_game (created_at, updated_at, name, description) VALUES (?, ?, ?, ?)`, now, now, input.Name, input.Description)
		if err != nil {
			return err
		}
		id, err = result.LastInsertId()
{{ else }}
		err := m.store.SQL(ctx).QueryRowContext(ctx, `INSERT INTO t_game (created_at, updated_at, name, description) VALUES ($1, $2, $3, $4) RETURNING id`, now, now, input.Name, input.Description).Scan(&id)
{{ end }}
		if err != nil {
			return err
		}
		value, err = m.Get(ctx, id)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("create game: %w", err)
	}
	return value, nil
}

// Update changes only supplied fields; an empty description clears it.
func (m *Module) Update(ctx context.Context, id int64, input UpdateInput) (*Game, error) {
	if id <= 0 || (input.Name == nil && input.Description == nil) {
		return nil, ErrInvalid
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || utf8.RuneCountInString(name) > 100 {
			return nil, ErrInvalid
		}
		input.Name = &name
	}
	var value *Game
	err := m.store.Tx(ctx, func(ctx context.Context) error {
		const query = `UPDATE t_game SET name = {{ if eq .Computed.database_driver_final "mysql" }}COALESCE(?, name), description = COALESCE(?, description), updated_at = ? WHERE id = ?{{ else }}COALESCE($1, name), description = COALESCE($2, description), updated_at = $3 WHERE id = $4{{ end }}`
		_, err := m.store.SQL(ctx).ExecContext(ctx, query, input.Name, input.Description, time.Now().UTC().Truncate(time.Microsecond), id)
		if err != nil {
			return err
		}
		// Reading inside the transaction distinguishes unchanged rows from missing rows in MySQL.
		value, err = m.Get(ctx, id)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("update game: %w", err)
	}
	return value, nil
}

// Delete removes matching rows in one statement. An entirely missing set returns
// ErrNotFound; missing IDs within a partially matching set are ignored.
func (m *Module) Delete(ctx context.Context, ids ...int64) error {
	if len(ids) == 0 || len(ids) > 100 {
		return ErrIDs
	}
	args := make([]any, 0, len(ids))
	placeholders := make([]string, 0, len(ids))
	seen := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return ErrIDs
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		args = append(args, id)
{{ if eq .Computed.database_driver_final "mysql" }}
		placeholders = append(placeholders, "?")
{{ else }}
		placeholders = append(placeholders, "$"+strconv.Itoa(len(args)))
{{ end }}
	}
	query := "DELETE FROM t_game WHERE id IN (" + strings.Join(placeholders, ",") + ")"
	result, err := m.store.SQL(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete game: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete game result: %w", err)
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

// Nil pagination fields select defaults; explicit values are preserved.
type ListInput struct {
	Name     string
	Page     *int32
	PageSize *int32
}
type ListResult struct {
	Items    []Game `json:"items"`
	Total    int64  `json:"total"`
	Page     int32  `json:"p"`
	PageSize int32  `json:"s"`
}

func (m *Module) List(ctx context.Context, input ListInput) (*ListResult, error) {
	page, size, inRange := m.limits.Normalize(input.Page, input.PageSize)
	value := &ListResult{Items: make([]Game, 0), Page: page, PageSize: size}
	if !inRange {
		return value, nil
	}
	filter := "%" + strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(strings.TrimSpace(input.Name)) + "%"
	const countQuery = `SELECT COUNT(*) FROM t_game WHERE LOWER(name) LIKE LOWER({{ if eq .Computed.database_driver_final "mysql" }}?{{ else }}$1{{ end }}) ESCAPE '!'`
	if err := m.store.SQL(ctx).QueryRowContext(ctx, countQuery, filter).Scan(&value.Total); err != nil {
		return nil, fmt.Errorf("count games: %w", err)
	}
	offset := int64(page-1) * int64(size)
	if offset >= value.Total {
		return value, nil
	}
	const query = `SELECT id, created_at, updated_at, name, description FROM t_game WHERE LOWER(name) LIKE LOWER({{ if eq .Computed.database_driver_final "mysql" }}?{{ else }}$1{{ end }}) ESCAPE '!' ORDER BY id DESC LIMIT {{ if eq .Computed.database_driver_final "mysql" }}? OFFSET ?{{ else }}$2 OFFSET $3{{ end }}`
	rows, err := m.store.SQL(ctx).QueryContext(ctx, query, filter, size, offset)
	if err != nil {
		return nil, fmt.Errorf("list games: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item Game
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&item.ID, &createdAt, &updatedAt, &item.Name, &item.Description); err != nil {
			return nil, fmt.Errorf("scan game: %w", err)
		}
		item.CreatedAt = createdAt.UnixMilli()
		item.UpdatedAt = updatedAt.UnixMilli()
		value.Items = append(value.Items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list games: %w", err)
	}
	return value, nil
}
