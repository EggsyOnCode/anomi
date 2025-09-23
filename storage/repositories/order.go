package repositories

import (
	"context"
	"time"

	"github.com/EggysOnCode/anomi/storage/models"
	"github.com/uptrace/bun"
)

// OrderRepository defines the interface for order-related database operations
type OrderRepository interface {
	Repository[models.Order]
	
	// Domain-specific methods
	GetByUserID(ctx context.Context, userID string) ([]models.Order, error)
	GetByUserIDAndType(ctx context.Context, userID, orderType string) ([]models.Order, error)
	GetByUserIDAndRole(ctx context.Context, userID, role string) ([]models.Order, error)
	GetActiveOrders(ctx context.Context, userID string) ([]models.Order, error)
	GetCanceledOrders(ctx context.Context, userID string) ([]models.Order, error)
	GetByTimeRange(ctx context.Context, start, end time.Time) ([]models.Order, error)
	GetByUserIDAndTimeRange(ctx context.Context, userID string, start, end time.Time) ([]models.Order, error)
}

// OrderRepositoryImpl implements OrderRepository interface
type OrderRepositoryImpl struct {
	*BaseRepository[models.Order]
}

// NewOrderRepository creates a new order repository
func NewOrderRepository(db *bun.DB) OrderRepository {
	return &OrderRepositoryImpl{
		BaseRepository: NewBaseRepository[models.Order](db, "orders"),
	}
}

// GetByUserID retrieves all orders for a specific user
func (r *OrderRepositoryImpl) GetByUserID(ctx context.Context, userID string) ([]models.Order, error) {
	var orders []models.Order
	err := r.db.NewSelect().Model(&orders).Where("user_id = ?", userID).Scan(ctx)
	return orders, err
}

// GetByUserIDAndType retrieves orders for a user filtered by order type
func (r *OrderRepositoryImpl) GetByUserIDAndType(ctx context.Context, userID, orderType string) ([]models.Order, error) {
	var orders []models.Order
	err := r.db.NewSelect().Model(&orders).
		Where("user_id = ? AND order_type = ?", userID, orderType).
		Scan(ctx)
	return orders, err
}

// GetByUserIDAndRole retrieves orders for a user filtered by role
func (r *OrderRepositoryImpl) GetByUserIDAndRole(ctx context.Context, userID, role string) ([]models.Order, error) {
	var orders []models.Order
	err := r.db.NewSelect().Model(&orders).
		Where("user_id = ? AND role = ?", userID, role).
		Scan(ctx)
	return orders, err
}

// GetActiveOrders retrieves non-canceled orders for a user
func (r *OrderRepositoryImpl) GetActiveOrders(ctx context.Context, userID string) ([]models.Order, error) {
	var orders []models.Order
	err := r.db.NewSelect().Model(&orders).
		Where("user_id = ? AND canceled = false", userID).
		Order("created_at DESC").
		Scan(ctx)
	return orders, err
}

// GetCanceledOrders retrieves canceled orders for a user
func (r *OrderRepositoryImpl) GetCanceledOrders(ctx context.Context, userID string) ([]models.Order, error) {
	var orders []models.Order
	err := r.db.NewSelect().Model(&orders).
		Where("user_id = ? AND canceled = true", userID).
		Order("created_at DESC").
		Scan(ctx)
	return orders, err
}

// GetByTimeRange retrieves orders within a time range
func (r *OrderRepositoryImpl) GetByTimeRange(ctx context.Context, start, end time.Time) ([]models.Order, error) {
	var orders []models.Order
	err := r.db.NewSelect().Model(&orders).
		Where("created_at >= ? AND created_at <= ?", start, end).
		Order("created_at DESC").
		Scan(ctx)
	return orders, err
}

// GetByUserIDAndTimeRange retrieves orders for a user within a time range
func (r *OrderRepositoryImpl) GetByUserIDAndTimeRange(ctx context.Context, userID string, start, end time.Time) ([]models.Order, error) {
	var orders []models.Order
	err := r.db.NewSelect().Model(&orders).
		Where("user_id = ? AND created_at >= ? AND created_at <= ?", userID, start, end).
		Order("created_at DESC").
		Scan(ctx)
	return orders, err
}
