package repositories

import (
	"context"
	"time"

	"github.com/EggysOnCode/anomi/storage/models"
	"github.com/uptrace/bun"
)

// ReceiptRepository defines the interface for receipt-related database operations
type ReceiptRepository interface {
	Repository[*models.Receipt]
	
	// Domain-specific methods
	GetByUserID(ctx context.Context, userID string) ([]*models.Receipt, error)
	GetByOrderID(ctx context.Context, orderID string) ([]*models.Receipt, error)
	GetByTradeID(ctx context.Context, tradeID string) (*models.Receipt, error)
	GetByUserIDAndOrderID(ctx context.Context, userID, orderID string) ([]*models.Receipt, error)
	GetByTimeRange(ctx context.Context, start, end time.Time) ([]*models.Receipt, error)
	GetByUserIDAndTimeRange(ctx context.Context, userID string, start, end time.Time) ([]*models.Receipt, error)
	GetReceiptsForOrder(ctx context.Context, orderID string) ([]*models.Receipt, error)
}

// ReceiptRepositoryImpl implements ReceiptRepository interface
type ReceiptRepositoryImpl struct {
	*BaseRepository[*models.Receipt]
}

// NewReceiptRepository creates a new receipt repository
func NewReceiptRepository(db *bun.DB) ReceiptRepository {
	return &ReceiptRepositoryImpl{
		BaseRepository: NewBaseRepository[*models.Receipt](db, "receipts"),
	}
}

// GetByUserID retrieves all receipts for a specific user
func (r *ReceiptRepositoryImpl) GetByUserID(ctx context.Context, userID string) ([]*models.Receipt, error) {
	var receipts []*models.Receipt
	err := r.db.NewSelect().Model(&receipts).Where("user_id = ?", userID).Scan(ctx)
	return receipts, err
}

// GetByOrderID retrieves all receipts for a specific order
func (r *ReceiptRepositoryImpl) GetByOrderID(ctx context.Context, orderID string) ([]*models.Receipt, error) {
	var receipts []*models.Receipt
	err := r.db.NewSelect().Model(&receipts).Where("order_id = ?", orderID).Scan(ctx)
	return receipts, err
}

// GetByTradeID retrieves a receipt by trade ID (should be unique)
func (r *ReceiptRepositoryImpl) GetByTradeID(ctx context.Context, tradeID string) (*models.Receipt, error) {
	receipt := new(models.Receipt)
	err := r.db.NewSelect().Model(receipt).Where("trade_id = ?", tradeID).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

// GetByUserIDAndOrderID retrieves receipts for a specific user and order
func (r *ReceiptRepositoryImpl) GetByUserIDAndOrderID(ctx context.Context, userID, orderID string) ([]*models.Receipt, error) {
	var receipts []*models.Receipt
	err := r.db.NewSelect().Model(&receipts).
		Where("user_id = ? AND order_id = ?", userID, orderID).
		Order("created_at DESC").
		Scan(ctx)
	return receipts, err
}

// GetByTimeRange retrieves receipts within a time range
func (r *ReceiptRepositoryImpl) GetByTimeRange(ctx context.Context, start, end time.Time) ([]*models.Receipt, error) {
	var receipts []*models.Receipt
	err := r.db.NewSelect().Model(&receipts).
		Where("created_at >= ? AND created_at <= ?", start, end).
		Order("created_at DESC").
		Scan(ctx)
	return receipts, err
}

// GetByUserIDAndTimeRange retrieves receipts for a user within a time range
func (r *ReceiptRepositoryImpl) GetByUserIDAndTimeRange(ctx context.Context, userID string, start, end time.Time) ([]*models.Receipt, error) {
	var receipts []*models.Receipt
	err := r.db.NewSelect().Model(&receipts).
		Where("user_id = ? AND created_at >= ? AND created_at <= ?", userID, start, end).
		Order("created_at DESC").
		Scan(ctx)
	return receipts, err
}

// GetReceiptsForOrder retrieves all receipts for a specific order (alias for GetByOrderID)
func (r *ReceiptRepositoryImpl) GetReceiptsForOrder(ctx context.Context, orderID string) ([]*models.Receipt, error) {
	return r.GetByOrderID(ctx, orderID)
}
