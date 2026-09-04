package repository

import (
	"context"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
)

type merchantDocumentCommandRepository struct {
	db *db.Queries
}

func NewMerchantDocumentCommandRepository(db *db.Queries) MerchantDocumentCommandRepository {
	return &merchantDocumentCommandRepository{
		db: db,
	}
}

func (r *merchantDocumentCommandRepository) CreateMerchantDocument(ctx context.Context, request *requests.CreateMerchantDocumentRequest) (*db.CreateMerchantDocumentRow, error) {
	note := ""

	req := db.CreateMerchantDocumentParams{
		MerchantID:   int32(request.MerchantID),
		DocumentType: request.DocumentType,
		DocumentUrl:  request.DocumentUrl,
		Status:       "pending",
		Note:         &note,
	}

	res, err := r.db.CreateMerchantDocument(ctx, req)
	if err != nil {
		return nil, sharedErrors.ErrFailed("create merchant document").WithInternal(err)
	}

	return res, nil
}

func (r *merchantDocumentCommandRepository) UpdateMerchantDocument(ctx context.Context, request *requests.UpdateMerchantDocumentRequest) (*db.UpdateMerchantDocumentRow, error) {
	if request.DocumentID == nil || *request.DocumentID <= 0 {
		return nil, sharedErrors.NewBadRequestError("merchant document ID is required")
	}

	note := ""

	req := db.UpdateMerchantDocumentParams{
		DocumentID:   int32(*request.DocumentID),
		DocumentType: request.DocumentType,
		DocumentUrl:  request.DocumentUrl,
		Status:       request.Status,
		Note:         &note,
	}

	res, err := r.db.UpdateMerchantDocument(ctx, req)
	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "merchant document", "update merchant document")
	}

	return res, nil
}

func (r *merchantDocumentCommandRepository) UpdateMerchantDocumentStatus(ctx context.Context, request *requests.UpdateMerchantDocumentStatusRequest) (*db.UpdateMerchantDocumentStatusRow, error) {
	if request.DocumentID == nil || *request.DocumentID <= 0 {
		return nil, sharedErrors.NewBadRequestError("merchant document ID is required")
	}

	note := ""

	req := db.UpdateMerchantDocumentStatusParams{
		DocumentID: int32(*request.DocumentID),
		Status:     request.Status,
		Note:       &note,
	}

	res, err := r.db.UpdateMerchantDocumentStatus(ctx, req)
	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "merchant document", "update merchant document status")
	}

	return res, nil
}

func (r *merchantDocumentCommandRepository) TrashedMerchantDocument(ctx context.Context, documentID int) (*db.MerchantDocument, error) {
	res, err := r.db.TrashMerchantDocument(ctx, int32(documentID))
	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "merchant document", "trash merchant document")
	}

	return res, nil
}

func (r *merchantDocumentCommandRepository) RestoreMerchantDocument(ctx context.Context, documentID int) (*db.MerchantDocument, error) {
	res, err := r.db.RestoreMerchantDocument(ctx, int32(documentID))
	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "merchant document", "restore merchant document")
	}

	return res, nil
}

func (r *merchantDocumentCommandRepository) DeleteMerchantDocumentPermanent(ctx context.Context, documentID int) (bool, error) {
	err := r.db.DeleteMerchantDocumentPermanently(ctx, int32(documentID))
	if err != nil {
		return false, sharedErrors.ErrNoRowsOrFailed(err, "merchant document", "delete merchant document permanently")
	}

	return true, nil
}

func (r *merchantDocumentCommandRepository) RestoreAllMerchantDocument(ctx context.Context) (bool, error) {
	err := r.db.RestoreAllMerchantDocuments(ctx)
	if err != nil {
		return false, sharedErrors.ErrFailed("restore all merchant documents").WithInternal(err)
	}

	return true, nil
}

func (r *merchantDocumentCommandRepository) DeleteAllMerchantDocumentPermanent(ctx context.Context) (bool, error) {
	err := r.db.DeleteAllPermanentMerchantDocuments(ctx)
	if err != nil {
		return false, sharedErrors.ErrFailed("delete all merchant documents permanently").WithInternal(err)
	}

	return true, nil
}
