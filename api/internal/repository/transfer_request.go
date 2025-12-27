package repository

import (
	"database/sql"
	"fmt"
	"time"

	"bitgo-wallets-api/internal/models"

	"github.com/google/uuid"
)

type TransferRequestRepository interface {
	Create(request *models.TransferRequest) error
	GetByID(id uuid.UUID) (*models.TransferRequest, error)
	List(walletID uuid.UUID, limit, offset int) ([]*models.TransferRequest, error)
	ListByStatus(status models.TransferStatus, limit, offset int) ([]*models.TransferRequest, error)
	Update(request *models.TransferRequest) error
	UpdateStatus(id uuid.UUID, status models.TransferStatus) error
}

type transferRequestRepository struct {
	db *sql.DB
}

func NewTransferRequestRepository(db *sql.DB) TransferRequestRepository {
	return &transferRequestRepository{db: db}
}

func (r *transferRequestRepository) Create(request *models.TransferRequest) error {
	query := `
		INSERT INTO transfer_requests (
			id, wallet_id, requested_by_user_id, recipient_address, amount_string,
			coin, transfer_type, status, required_approvals, memo
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at, updated_at
	`

	request.ID = uuid.New()
	err := r.db.QueryRow(
		query,
		request.ID, request.WalletID, request.RequestedByUserID,
		request.RecipientAddress, request.AmountString, request.Coin,
		request.TransferType, request.Status, request.RequiredApprovals,
		request.Memo,
	).Scan(&request.CreatedAt, &request.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create transfer request: %w", err)
	}

	return nil
}

func (r *transferRequestRepository) GetByID(id uuid.UUID) (*models.TransferRequest, error) {
	query := `
		SELECT id, wallet_id, requested_by_user_id, recipient_address, amount_string,
		       coin, transfer_type, status, bitgo_transfer_id, transaction_hash,
		       required_approvals, received_approvals, memo, fee_string,
		       estimated_fee_string, submitted_at, approved_at, completed_at,
		       failed_at, created_at, updated_at
		FROM transfer_requests
		WHERE id = $1
	`

	request := &models.TransferRequest{}
	err := r.db.QueryRow(query, id).Scan(
		&request.ID, &request.WalletID, &request.RequestedByUserID,
		&request.RecipientAddress, &request.AmountString, &request.Coin,
		&request.TransferType, &request.Status, &request.BitgoTransferID,
		&request.TransactionHash, &request.RequiredApprovals,
		&request.ReceivedApprovals, &request.Memo, &request.FeeString,
		&request.EstimatedFeeString, &request.SubmittedAt, &request.ApprovedAt,
		&request.CompletedAt, &request.FailedAt, &request.CreatedAt,
		&request.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get transfer request by ID: %w", err)
	}

	return request, nil
}

func (r *transferRequestRepository) List(walletID uuid.UUID, limit, offset int) ([]*models.TransferRequest, error) {
	query := `
		SELECT id, wallet_id, requested_by_user_id, recipient_address, amount_string,
		       coin, transfer_type, status, bitgo_transfer_id, transaction_hash,
		       required_approvals, received_approvals, memo, fee_string,
		       estimated_fee_string, submitted_at, approved_at, completed_at,
		       failed_at, created_at, updated_at
		FROM transfer_requests
		WHERE wallet_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, walletID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list transfer requests: %w", err)
	}
	defer rows.Close()

	var requests []*models.TransferRequest
	for rows.Next() {
		request := &models.TransferRequest{}
		err := rows.Scan(
			&request.ID, &request.WalletID, &request.RequestedByUserID,
			&request.RecipientAddress, &request.AmountString, &request.Coin,
			&request.TransferType, &request.Status, &request.BitgoTransferID,
			&request.TransactionHash, &request.RequiredApprovals,
			&request.ReceivedApprovals, &request.Memo, &request.FeeString,
			&request.EstimatedFeeString, &request.SubmittedAt, &request.ApprovedAt,
			&request.CompletedAt, &request.FailedAt, &request.CreatedAt,
			&request.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transfer request: %w", err)
		}
		requests = append(requests, request)
	}

	return requests, nil
}

func (r *transferRequestRepository) ListByStatus(status models.TransferStatus, limit, offset int) ([]*models.TransferRequest, error) {
	query := `
		SELECT id, wallet_id, requested_by_user_id, recipient_address, amount_string,
		       coin, transfer_type, status, bitgo_transfer_id, transaction_hash,
		       required_approvals, received_approvals, memo, fee_string,
		       estimated_fee_string, submitted_at, approved_at, completed_at,
		       failed_at, created_at, updated_at
		FROM transfer_requests
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list transfer requests by status: %w", err)
	}
	defer rows.Close()

	var requests []*models.TransferRequest
	for rows.Next() {
		request := &models.TransferRequest{}
		err := rows.Scan(
			&request.ID, &request.WalletID, &request.RequestedByUserID,
			&request.RecipientAddress, &request.AmountString, &request.Coin,
			&request.TransferType, &request.Status, &request.BitgoTransferID,
			&request.TransactionHash, &request.RequiredApprovals,
			&request.ReceivedApprovals, &request.Memo, &request.FeeString,
			&request.EstimatedFeeString, &request.SubmittedAt, &request.ApprovedAt,
			&request.CompletedAt, &request.FailedAt, &request.CreatedAt,
			&request.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transfer request: %w", err)
		}
		requests = append(requests, request)
	}

	return requests, nil
}

func (r *transferRequestRepository) Update(request *models.TransferRequest) error {
	query := `
		UPDATE transfer_requests
		SET status = $1, bitgo_transfer_id = $2, transaction_hash = $3,
		    received_approvals = $4, fee_string = $5, estimated_fee_string = $6,
		    submitted_at = $7, approved_at = $8, completed_at = $9, failed_at = $10,
		    updated_at = NOW()
		WHERE id = $11
		RETURNING updated_at
	`

	err := r.db.QueryRow(
		query,
		request.Status, request.BitgoTransferID, request.TransactionHash,
		request.ReceivedApprovals, request.FeeString, request.EstimatedFeeString,
		request.SubmittedAt, request.ApprovedAt, request.CompletedAt,
		request.FailedAt, request.ID,
	).Scan(&request.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to update transfer request: %w", err)
	}

	return nil
}

func (r *transferRequestRepository) UpdateStatus(id uuid.UUID, status models.TransferStatus) error {
	var query string
	var args []interface{}

	switch status {
	case models.TransferStatusSubmitted:
		query = `UPDATE transfer_requests SET status = $1, submitted_at = $2, updated_at = NOW() WHERE id = $3`
		args = []interface{}{status, time.Now(), id}
	case models.TransferStatusApproved:
		query = `UPDATE transfer_requests SET status = $1, approved_at = $2, updated_at = NOW() WHERE id = $3`
		args = []interface{}{status, time.Now(), id}
	case models.TransferStatusCompleted:
		query = `UPDATE transfer_requests SET status = $1, completed_at = $2, updated_at = NOW() WHERE id = $3`
		args = []interface{}{status, time.Now(), id}
	case models.TransferStatusFailed:
		query = `UPDATE transfer_requests SET status = $1, failed_at = $2, updated_at = NOW() WHERE id = $3`
		args = []interface{}{status, time.Now(), id}
	default:
		query = `UPDATE transfer_requests SET status = $1, updated_at = NOW() WHERE id = $2`
		args = []interface{}{status, id}
	}

	_, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update transfer request status: %w", err)
	}

	return nil
}
