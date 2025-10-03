package repositories

import (
	"context"

	"github.com/uptrace/bun"
)

// Repository defines the base CRUD operations for all repositories
type Repository[T any] interface {
	Create(ctx context.Context, entity T) error
	GetByID(ctx context.Context, id string) (T, error)
	Update(ctx context.Context, entity T) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]T, error)
	Count(ctx context.Context) (int64, error)
}

// BaseRepository provides a generic implementation of Repository interface
type BaseRepository[T any] struct {
	db        *bun.DB
	tableName string
}

// NewBaseRepository creates a new base repository
func NewBaseRepository[T any](db *bun.DB, tableName string) *BaseRepository[T] {
	return &BaseRepository[T]{
		db:        db,
		tableName: tableName,
	}
}

// Create inserts a new entity
func (r *BaseRepository[T]) Create(ctx context.Context, entity T) error {
	_, err := r.db.NewInsert().Model(&entity).Exec(ctx)
	return err
}

// GetByID retrieves an entity by ID
func (r *BaseRepository[T]) GetByID(ctx context.Context, id string) (T, error) {
	var entity T
	err := r.db.NewSelect().Model(&entity).Where("id = ?", id).Scan(ctx)
	return entity, err
}

// Update updates an existing entity
func (r *BaseRepository[T]) Update(ctx context.Context, entity T) error {
	// Extract ID from entity - this assumes the entity has an ID field
	// For now, we'll use a generic approach that works with our models
	_, err := r.db.NewUpdate().Model(&entity).WherePK().Exec(ctx)
	return err
}

// Delete removes an entity by ID
func (r *BaseRepository[T]) Delete(ctx context.Context, id string) error {
	var entity T
	_, err := r.db.NewDelete().Model(&entity).Where("id = ?", id).Exec(ctx)
	return err
}

// List retrieves entities with pagination
func (r *BaseRepository[T]) List(ctx context.Context, limit, offset int) ([]T, error) {
	var entities []T
	query := r.db.NewSelect().Model(&entities)

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Scan(ctx)
	return entities, err
}

// Count returns the total number of entities
func (r *BaseRepository[T]) Count(ctx context.Context) (int64, error) {
	var entity T
	count, err := r.db.NewSelect().Model(&entity).Count(ctx)
	return int64(count), err
}
