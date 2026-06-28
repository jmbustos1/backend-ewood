package models

import (
	"time"
)

// WalletTransactionType representa los tipos de transaccion en la wallet.
type WalletTransactionType string

const (
	WalletTxRecharge   WalletTransactionType = "recharge"
	WalletTxTripCharge WalletTransactionType = "trip_charge"
	WalletTxRefund     WalletTransactionType = "refund"
	WalletTxAdjustment WalletTransactionType = "adjustment"
)

// Wallet representa el saldo prepago de un usuario.
// Balance puede ser negativo si un viaje supero el saldo disponible —
// el usuario debe regularizar antes de poder iniciar otro arriendo.
type Wallet struct {
	UserID    uint      `gorm:"primaryKey;column:user_id" json:"user_id"`
	Balance   float64   `gorm:"type:decimal(10,2);not null;default:0" json:"balance"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Wallet) TableName() string {
	return "wallets"
}

// CanRent retorna true si el usuario puede iniciar un arriendo
// (saldo no negativo).
func (w *Wallet) CanRent() bool {
	return w.Balance >= 0
}

// WalletTransaction representa un movimiento en la wallet.
// amount es positivo en recargas/refunds, negativo en cobros.
type WalletTransaction struct {
	ID            uint                  `gorm:"primaryKey" json:"id"`
	UserID        uint                  `gorm:"not null;index" json:"user_id"`
	Type          WalletTransactionType `gorm:"type:varchar(20);not null" json:"type"`
	Amount        float64               `gorm:"type:decimal(10,2);not null" json:"amount"`
	BalanceAfter  float64               `gorm:"type:decimal(10,2);not null;column:balance_after" json:"balance_after"`
	TripID        *uint                 `gorm:"index;column:trip_id" json:"trip_id,omitempty"`
	MPPaymentID   *string               `gorm:"size:100;column:mp_payment_id" json:"mp_payment_id,omitempty"`
	Description   string                `gorm:"type:text" json:"description,omitempty"`
	CreatedAt     time.Time             `gorm:"index" json:"created_at"`
}

func (WalletTransaction) TableName() string {
	return "wallet_transactions"
}
