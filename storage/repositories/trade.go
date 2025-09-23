package repositories

import (
	"context"
	"time"

	"github.com/EggysOnCode/anomi/storage/models"
	"github.com/uptrace/bun"
)

// TradeRepository defines the interface for trade-related database operations
type TradeRepository interface {
	Repository[*models.Trade]
	
	// Domain-specific methods
	GetByUserID(ctx context.Context, userID string) ([]*models.Trade, error)
	GetByOrderID(ctx context.Context, orderID string) ([]*models.Trade, error)
	GetByUserIDAndRole(ctx context.Context, userID, role string) ([]*models.Trade, error)
	GetByTimeRange(ctx context.Context, start, end time.Time) ([]*models.Trade, error)
	GetByUserIDAndTimeRange(ctx context.Context, userID string, start, end time.Time) ([]*models.Trade, error)
	GetTradesForOrder(ctx context.Context, orderID string) ([]*models.Trade, error)
	GetTradesByUserIDAndOrderID(ctx context.Context, userID, orderID string) ([]*models.Trade, error)
}

// TradeRepositoryImpl implements TradeRepository interface
type TradeRepositoryImpl struct {
	*BaseRepository[*models.Trade]
}

// NewTradeRepository creates a new trade repository
func NewTradeRepository(db *bun.DB) TradeRepository {
	return &TradeRepositoryImpl{
		BaseRepository: NewBaseRepository[*models.Trade](db, "trades"),
	}
}

// GetByUserID retrieves all trades for a specific user
func (r *TradeRepositoryImpl) GetByUserID(ctx context.Context, userID string) ([]*models.Trade, error) {
	var trades []*models.Trade
	err := r.db.NewSelect().Model(&trades).Where("user_id = ?", userID).Scan(ctx)
	return trades, err
}

// GetByOrderID retrieves all trades for a specific order
func (r *TradeRepositoryImpl) GetByOrderID(ctx context.Context, orderID string) ([]*models.Trade, error) {
	var trades []*models.Trade
	err := r.db.NewSelect().Model(&trades).Where("order_id = ?", orderID).Scan(ctx)
	return trades, err
}

// GetByUserIDAndRole retrieves trades for a user filtered by role
func (r *TradeRepositoryImpl) GetByUserIDAndRole(ctx context.Context, userID, role string) ([]*models.Trade, error) {
	var trades []*models.Trade
	err := r.db.NewSelect().Model(&trades).
		Where("user_id = ? AND role = ?", userID, role).
		Scan(ctx)
	return trades, err
}

// GetByTimeRange retrieves trades within a time range
func (r *TradeRepositoryImpl) GetByTimeRange(ctx context.Context, start, end time.Time) ([]*models.Trade, error) {
	var trades []*models.Trade
	err := r.db.NewSelect().Model(&trades).
		Where("created_at >= ? AND created_at <= ?", start, end).
		Order("created_at DESC").
		Scan(ctx)
	return trades, err
}

// GetByUserIDAndTimeRange retrieves trades for a user within a time range
func (r *TradeRepositoryImpl) GetByUserIDAndTimeRange(ctx context.Context, userID string, start, end time.Time) ([]*models.Trade, error) {
	var trades []*models.Trade
	err := r.db.NewSelect().Model(&trades).
		Where("user_id = ? AND created_at >= ? AND created_at <= ?", userID, start, end).
		Order("created_at DESC").
		Scan(ctx)
	return trades, err
}

// GetTradesForOrder retrieves all trades for a specific order (alias for GetByOrderID)
func (r *TradeRepositoryImpl) GetTradesForOrder(ctx context.Context, orderID string) ([]*models.Trade, error) {
	return r.GetByOrderID(ctx, orderID)
}

// GetTradesByUserIDAndOrderID retrieves trades for a specific user and order
func (r *TradeRepositoryImpl) GetTradesByUserIDAndOrderID(ctx context.Context, userID, orderID string) ([]*models.Trade, error) {
	var trades []*models.Trade
	err := r.db.NewSelect().Model(&trades).
		Where("user_id = ? AND order_id = ?", userID, orderID).
		Order("created_at DESC").
		Scan(ctx)
	return trades, err
}
