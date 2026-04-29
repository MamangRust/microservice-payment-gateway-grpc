package repository

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	merchantdocument_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/merchant_document_errors/repository"
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
		return nil, merchantdocument_errors.ErrCreateMerchantDocumentFailed.WithInternal(err)
	}

	return res, nil
}

func (r *merchantDocumentCommandRepository) UpdateMerchantDocument(ctx context.Context, request *requests.UpdateMerchantDocumentRequest) (*db.UpdateMerchantDocumentRow, error) {
	note := ""

	req := db.UpdateMerchantDocumentParams{
		DocumentID:   int32(request.MerchantID),
		DocumentType: request.DocumentType,
		DocumentUrl:  request.DocumentUrl,
		Status:       request.Status,
		Note:         &note,
	}

	res, err := r.db.UpdateMerchantDocument(ctx, req)
	if err != nil {
		return nil, merchantdocument_errors.ErrUpdateMerchantDocumentFailed.WithInternal(err)
	}

	return res, nil
}

func (r *merchantDocumentCommandRepository) UpdateMerchantDocumentStatus(ctx context.Context, request *requests.UpdateMerchantDocumentStatusRequest) (*db.UpdateMerchantDocumentStatusRow, error) {
	note := ""

	req := db.UpdateMerchantDocumentStatusParams{
		DocumentID: int32(request.MerchantID),
		Status:     request.Status,
		Note:       &note,
	}

	res, err := r.db.UpdateMerchantDocumentStatus(ctx, req)
	if err != nil {
		return nil, merchantdocument_errors.ErrUpdateMerchantDocumentStatusFailed.WithInternal(err)
	}

	return res, nil
}

func (r *merchantDocumentCommandRepository) TrashedMerchantDocument(ctx context.Context, documentID int) (*db.MerchantDocument, error) {
	res, err := r.db.TrashMerchantDocument(ctx, int32(documentID))
	if err != nil {
		return nil, merchantdocument_errors.ErrTrashedMerchantDocumentFailed.WithInternal(err)
	}

	return res, nil
}

func (r *merchantDocumentCommandRepository) RestoreMerchantDocument(ctx context.Context, documentID int) (*db.MerchantDocument, error) {
	res, err := r.db.RestoreMerchantDocument(ctx, int32(documentID))
	if err != nil {
		return nil, merchantdocument_errors.ErrRestoreMerchantDocumentFailed.WithInternal(err)
	}

	return res, nil
}

func (r *merchantDocumentCommandRepository) DeleteMerchantDocumentPermanent(ctx context.Context, documentID int) (bool, error) {
	err := r.db.DeleteMerchantDocumentPermanently(ctx, int32(documentID))
	if err != nil {
		return false, merchantdocument_errors.ErrDeleteMerchantDocumentPermanentFailed.WithInternal(err)
	}

	return true, nil
}

func (r *merchantDocumentCommandRepository) RestoreAllMerchantDocument(ctx context.Context) (bool, error) {
	err := r.db.RestoreAllMerchantDocuments(ctx)
	if err != nil {
		return false, merchantdocument_errors.ErrRestoreAllMerchantDocumentsFailed.WithInternal(err)
	}

	return true, nil
}

func (r *merchantDocumentCommandRepository) DeleteAllMerchantDocumentPermanent(ctx context.Context) (bool, error) {
	err := r.db.DeleteAllPermanentMerchantDocuments(ctx)
	if err != nil {
		return false, merchantdocument_errors.ErrDeleteAllMerchantDocumentsPermanentFailed.WithInternal(err)
	}

	return true, nil
}
