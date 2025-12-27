package repository

import (
	"database/sql"
	"fmt"

	"bitgo-wallets-api/internal/models"

	"github.com/google/uuid"
)

type WalletRepository interface {
	Create(wallet *models.Wallet) error
	GetByID(id uuid.UUID) (*models.Wallet, error)
	GetByBitgoID(bitgoWalletID string) (*models.Wallet, error)
	List(organizationID uuid.UUID, limit, offset int) ([]*models.Wallet, error)
	Update(wallet *models.Wallet) error
	Delete(id uuid.UUID) error
}

type walletRepository struct {
	db *sql.DB
}

func NewWalletRepository(db *sql.DB) WalletRepository {
	return &walletRepository{db: db}
}

func (r *walletRepository) Create(wallet *models.Wallet) error {
	query := `
		INSERT INTO wallets (
			id, organization_id, bitgo_wallet_id, label, coin, wallet_type,
			balance_string, confirmed_balance_string, spendable_balance_string,
			is_active, frozen, multisig_type, threshold, tags, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING created_at, updated_at
	`

	wallet.ID = uuid.New()
	err := r.db.QueryRow(
		query,
		wallet.ID, wallet.OrganizationID, wallet.BitgoWalletID, wallet.Label,
		wallet.Coin, wallet.WalletType, wallet.BalanceString,
		wallet.ConfirmedBalanceString, wallet.SpendableBalanceString,
		wallet.IsActive, wallet.Frozen, wallet.MultisigType, wallet.Threshold,
		wallet.Tags, wallet.Metadata,
	).Scan(&wallet.CreatedAt, &wallet.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create wallet: %w", err)
	}

	return nil
}

func (r *walletRepository) GetByID(id uuid.UUID) (*models.Wallet, error) {
	query := `
		SELECT id, organization_id, bitgo_wallet_id, label, coin, wallet_type,
		       balance_string, confirmed_balance_string, spendable_balance_string,
		       is_active, frozen, multisig_type, threshold, tags, metadata,
		       created_at, updated_at
		FROM wallets
		WHERE id = $1 AND is_active = true
	`

	wallet := &models.Wallet{}
	err := r.db.QueryRow(query, id).Scan(
		&wallet.ID, &wallet.OrganizationID, &wallet.BitgoWalletID, &wallet.Label,
		&wallet.Coin, &wallet.WalletType, &wallet.BalanceString,
		&wallet.ConfirmedBalanceString, &wallet.SpendableBalanceString,
		&wallet.IsActive, &wallet.Frozen, &wallet.MultisigType, &wallet.Threshold,
		&wallet.Tags, &wallet.Metadata, &wallet.CreatedAt, &wallet.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet by ID: %w", err)
	}

	return wallet, nil
}

func (r *walletRepository) GetByBitgoID(bitgoWalletID string) (*models.Wallet, error) {
	query := `
		SELECT id, organization_id, bitgo_wallet_id, label, coin, wallet_type,
		       balance_string, confirmed_balance_string, spendable_balance_string,
		       is_active, frozen, multisig_type, threshold, tags, metadata,
		       created_at, updated_at
		FROM wallets
		WHERE bitgo_wallet_id = $1 AND is_active = true
	`

	wallet := &models.Wallet{}
	err := r.db.QueryRow(query, bitgoWalletID).Scan(
		&wallet.ID, &wallet.OrganizationID, &wallet.BitgoWalletID, &wallet.Label,
		&wallet.Coin, &wallet.WalletType, &wallet.BalanceString,
		&wallet.ConfirmedBalanceString, &wallet.SpendableBalanceString,
		&wallet.IsActive, &wallet.Frozen, &wallet.MultisigType, &wallet.Threshold,
		&wallet.Tags, &wallet.Metadata, &wallet.CreatedAt, &wallet.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet by BitGo ID: %w", err)
	}

	return wallet, nil
}

func (r *walletRepository) List(organizationID uuid.UUID, limit, offset int) ([]*models.Wallet, error) {
	query := `
		SELECT id, organization_id, bitgo_wallet_id, label, coin, wallet_type,
		       balance_string, confirmed_balance_string, spendable_balance_string,
		       is_active, frozen, multisig_type, threshold, tags, metadata,
		       created_at, updated_at
		FROM wallets
		WHERE organization_id = $1 AND is_active = true
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, organizationID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list wallets: %w", err)
	}
	defer rows.Close()

	var wallets []*models.Wallet
	for rows.Next() {
		wallet := &models.Wallet{}
		err := rows.Scan(
			&wallet.ID, &wallet.OrganizationID, &wallet.BitgoWalletID, &wallet.Label,
			&wallet.Coin, &wallet.WalletType, &wallet.BalanceString,
			&wallet.ConfirmedBalanceString, &wallet.SpendableBalanceString,
			&wallet.IsActive, &wallet.Frozen, &wallet.MultisigType, &wallet.Threshold,
			&wallet.Tags, &wallet.Metadata, &wallet.CreatedAt, &wallet.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan wallet: %w", err)
		}
		wallets = append(wallets, wallet)
	}

	return wallets, nil
}

func (r *walletRepository) Update(wallet *models.Wallet) error {
	query := `
		UPDATE wallets
		SET label = $1, balance_string = $2, confirmed_balance_string = $3,
		    spendable_balance_string = $4, frozen = $5, tags = $6, metadata = $7,
		    updated_at = NOW()
		WHERE id = $8
		RETURNING updated_at
	`

	err := r.db.QueryRow(
		query,
		wallet.Label, wallet.BalanceString, wallet.ConfirmedBalanceString,
		wallet.SpendableBalanceString, wallet.Frozen, wallet.Tags,
		wallet.Metadata, wallet.ID,
	).Scan(&wallet.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to update wallet: %w", err)
	}

	return nil
}

func (r *walletRepository) Delete(id uuid.UUID) error {
	query := `UPDATE wallets SET is_active = false, updated_at = NOW() WHERE id = $1`

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete wallet: %w", err)
	}

	return nil
}
