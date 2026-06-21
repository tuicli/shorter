package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/tuicli/shorter/internal/app"
	"github.com/tuicli/shorter/internal/domain"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindByOriginalURLs(ctx context.Context, urls []string) (map[string]domain.ShortLink, error) {
	out := map[string]domain.ShortLink{}
	if len(urls) == 0 {
		return out, nil
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, code, original_url, title, status, created_by_telegram_id, updated_by_telegram_id,
		       disabled_at, disabled_by_telegram_id, deleted_at, deleted_by_telegram_id, created_at, updated_at
		FROM short_links
		WHERE original_url = ANY($1)
	`, pq.Array(urls))
	if err != nil {
		return nil, fmt.Errorf("query original urls: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		link, err := scanLink(rows)
		if err != nil {
			return nil, err
		}
		out[link.OriginalURL] = link
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate original urls: %w", err)
	}

	return out, nil
}

func (r *Repository) InsertLink(ctx context.Context, input domain.NewShortLink) (domain.ShortLink, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.ShortLink{}, fmt.Errorf("begin insert link: %w", err)
	}
	defer rollback(tx)

	link, err := scanLink(tx.QueryRowContext(ctx, `
		INSERT INTO short_links (code, original_url, title, created_by_telegram_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, code, original_url, title, status, created_by_telegram_id, updated_by_telegram_id,
		          disabled_at, disabled_by_telegram_id, deleted_at, deleted_by_telegram_id, created_at, updated_at
	`, input.Code, input.OriginalURL, input.Title, input.CreatedByTelegramID))
	if err != nil {
		return domain.ShortLink{}, mapConstraintError(err)
	}

	if err := insertEvent(ctx, tx, app.Event{
		ShortLinkID:     link.ID,
		EventType:       "created",
		ActorTelegramID: input.CreatedByTelegramID,
		Metadata:        "{}",
	}); err != nil {
		return domain.ShortLink{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.ShortLink{}, fmt.Errorf("commit insert link: %w", err)
	}

	return link, nil
}

func (r *Repository) ListLatest(ctx context.Context, page int, limit int) (app.RepositoryPage, error) {
	return r.list(ctx, `
		WHERE status IN ('active', 'disabled')
		ORDER BY created_at DESC, id DESC
	`, []any{}, page, limit)
}

func (r *Repository) Search(ctx context.Context, query string, page int, limit int) (app.RepositoryPage, error) {
	pattern := "%" + escapeLike(query) + "%"
	return r.list(ctx, `
		WHERE status IN ('active', 'disabled')
		  AND (code ILIKE $1 ESCAPE '\' OR title ILIKE $1 ESCAPE '\' OR original_url ILIKE $1 ESCAPE '\')
		ORDER BY created_at DESC, id DESC
	`, []any{pattern}, page, limit)
}

func (r *Repository) GetByID(ctx context.Context, id int64) (domain.ShortLink, bool, error) {
	return r.get(ctx, "WHERE id = $1", id)
}

func (r *Repository) GetByCode(ctx context.Context, code string) (domain.ShortLink, bool, error) {
	return r.get(ctx, "WHERE code = $1", code)
}

func (r *Repository) SetStatus(ctx context.Context, id int64, status domain.LinkStatus, adminID int64) (domain.ShortLink, bool, error) {
	if !status.Valid() {
		return domain.ShortLink{}, false, domain.ErrInvalidStatus
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.ShortLink{}, false, fmt.Errorf("begin set status: %w", err)
	}
	defer rollback(tx)

	link, err := scanLink(tx.QueryRowContext(ctx, `
		UPDATE short_links
		SET status = $2,
		    updated_by_telegram_id = $3,
		    updated_at = now(),
		    disabled_at = CASE
		        WHEN $2 = 'disabled' THEN now()
		        WHEN $2 = 'active' THEN NULL
		        ELSE disabled_at
		    END,
		    disabled_by_telegram_id = CASE
		        WHEN $2 = 'disabled' THEN $3
		        WHEN $2 = 'active' THEN NULL
		        ELSE disabled_by_telegram_id
		    END,
		    deleted_at = CASE WHEN $2 = 'deleted' THEN now() ELSE deleted_at END,
		    deleted_by_telegram_id = CASE WHEN $2 = 'deleted' THEN $3 ELSE deleted_by_telegram_id END
		WHERE id = $1 AND status <> 'deleted'
		RETURNING id, code, original_url, title, status, created_by_telegram_id, updated_by_telegram_id,
		          disabled_at, disabled_by_telegram_id, deleted_at, deleted_by_telegram_id, created_at, updated_at
	`, id, string(status), adminID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ShortLink{}, false, nil
	}
	if err != nil {
		return domain.ShortLink{}, false, fmt.Errorf("update link status: %w", err)
	}

	if err := insertEvent(ctx, tx, app.Event{
		ShortLinkID:     link.ID,
		EventType:       eventTypeForStatus(status),
		ActorTelegramID: adminID,
		Metadata:        "{}",
	}); err != nil {
		return domain.ShortLink{}, false, err
	}

	if err := tx.Commit(); err != nil {
		return domain.ShortLink{}, false, fmt.Errorf("commit set status: %w", err)
	}

	return link, true, nil
}

func (r *Repository) ExportLinks(ctx context.Context, limit int) ([]domain.ShortLink, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, code, original_url, title, status, created_by_telegram_id, updated_by_telegram_id,
		       disabled_at, disabled_by_telegram_id, deleted_at, deleted_by_telegram_id, created_at, updated_at
		FROM short_links
		ORDER BY id ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query export links: %w", err)
	}
	defer rows.Close()

	links := []domain.ShortLink{}
	for rows.Next() {
		link, err := scanLink(rows)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate export links: %w", err)
	}

	return links, nil
}

func (r *Repository) RecordEvent(ctx context.Context, event app.Event) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO link_events (short_link_id, event_type, actor_telegram_id, metadata)
		VALUES ($1, $2, $3, COALESCE(NULLIF($4, ''), '{}')::jsonb)
	`, nullableID(event.ShortLinkID), event.EventType, nullableID(event.ActorTelegramID), event.Metadata)
	if err != nil {
		return fmt.Errorf("record link event: %w", err)
	}
	return nil
}

func (r *Repository) list(ctx context.Context, whereOrder string, args []any, page int, limit int) (app.RepositoryPage, error) {
	countQuery := "SELECT count(*) FROM short_links " + strings.Split(whereOrder, "ORDER BY")[0]
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return app.RepositoryPage{}, fmt.Errorf("count links: %w", err)
	}

	offset := (page - 1) * limit
	queryArgs := append(append([]any{}, args...), limit, offset)
	limitParam := len(args) + 1
	offsetParam := len(args) + 2
	query := fmt.Sprintf(`
		SELECT id, code, original_url, title, status, created_by_telegram_id, updated_by_telegram_id,
		       disabled_at, disabled_by_telegram_id, deleted_at, deleted_by_telegram_id, created_at, updated_at
		FROM short_links
		%s
		LIMIT $%d OFFSET $%d
	`, whereOrder, limitParam, offsetParam)

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return app.RepositoryPage{}, fmt.Errorf("query links: %w", err)
	}
	defer rows.Close()

	links := []domain.ShortLink{}
	for rows.Next() {
		link, err := scanLink(rows)
		if err != nil {
			return app.RepositoryPage{}, err
		}
		links = append(links, link)
	}
	if err := rows.Err(); err != nil {
		return app.RepositoryPage{}, fmt.Errorf("iterate links: %w", err)
	}

	return app.RepositoryPage{Links: links, Total: total}, nil
}

func (r *Repository) get(ctx context.Context, where string, args ...any) (domain.ShortLink, bool, error) {
	link, err := scanLink(r.db.QueryRowContext(ctx, `
		SELECT id, code, original_url, title, status, created_by_telegram_id, updated_by_telegram_id,
		       disabled_at, disabled_by_telegram_id, deleted_at, deleted_by_telegram_id, created_at, updated_at
		FROM short_links
		`+where+`
	`, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ShortLink{}, false, nil
	}
	if err != nil {
		return domain.ShortLink{}, false, fmt.Errorf("get link: %w", err)
	}
	return link, true, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanLink(row scanner) (domain.ShortLink, error) {
	var link domain.ShortLink
	var status string
	var updatedBy sql.NullInt64
	var disabledAt sql.NullTime
	var disabledBy sql.NullInt64
	var deletedAt sql.NullTime
	var deletedBy sql.NullInt64

	err := row.Scan(
		&link.ID,
		&link.Code,
		&link.OriginalURL,
		&link.Title,
		&status,
		&link.CreatedByTelegramID,
		&updatedBy,
		&disabledAt,
		&disabledBy,
		&deletedAt,
		&deletedBy,
		&link.CreatedAt,
		&link.UpdatedAt,
	)
	if err != nil {
		return domain.ShortLink{}, err
	}

	link.Status = domain.LinkStatus(status)
	if updatedBy.Valid {
		link.UpdatedByTelegramID = updatedBy.Int64
	}
	if disabledAt.Valid {
		link.DisabledAt = &disabledAt.Time
	}
	if disabledBy.Valid {
		link.DisabledByTelegramID = disabledBy.Int64
	}
	if deletedAt.Valid {
		link.DeletedAt = &deletedAt.Time
	}
	if deletedBy.Valid {
		link.DeletedByTelegramID = deletedBy.Int64
	}

	return link, nil
}

func insertEvent(ctx context.Context, tx *sql.Tx, event app.Event) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO link_events (short_link_id, event_type, actor_telegram_id, metadata)
		VALUES ($1, $2, $3, COALESCE(NULLIF($4, ''), '{}')::jsonb)
	`, nullableID(event.ShortLinkID), event.EventType, nullableID(event.ActorTelegramID), event.Metadata)
	if err != nil {
		return fmt.Errorf("insert link event: %w", err)
	}
	return nil
}

func nullableID(id int64) sql.NullInt64 {
	if id == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: id, Valid: true}
}

func rollback(tx *sql.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}

func mapConstraintError(err error) error {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return err
	}
	if pqErr.Code != "23505" {
		return err
	}

	switch pqErr.Constraint {
	case "short_links_code_key":
		return domain.ErrCodeExists
	case "short_links_original_url_key":
		return domain.ErrOriginalURLExists
	default:
		return err
	}
}

func eventTypeForStatus(status domain.LinkStatus) string {
	switch status {
	case domain.StatusActive:
		return "enabled"
	case domain.StatusDisabled:
		return "disabled"
	case domain.StatusDeleted:
		return "deleted"
	default:
		return "status_changed"
	}
}

func escapeLike(query string) string {
	query = strings.ReplaceAll(query, `\`, `\\`)
	query = strings.ReplaceAll(query, `%`, `\%`)
	query = strings.ReplaceAll(query, `_`, `\_`)
	return query
}
