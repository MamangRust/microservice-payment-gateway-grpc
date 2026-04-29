package repository

import (
	"context"
	"database/sql"
	"errors"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	merchantdocument_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/merchant_document_errors/repository"
)

// merchantDocumentQueryRepository provides methods to query merchant documents from the database.
type merchantDocumentQueryRepository struct {
	db *db.Queries
}

// NewMerchantDocumentQueryRepository creates a new instance of merchantDocumentQueryRepository.
func NewMerchantDocumentQueryRepository(db *db.Queries) MerchantDocumentQueryRepository {
	return &merchantDocumentQueryRepository{
		db: db,
	}
}

func (r *merchantDocumentQueryRepository) FindAllDocuments(ctx context.Context, req *requests.FindAllMerchantDocuments) ([]*db.GetMerchantDocumentsRow, error) {
	offset := (req.Page - 1) * req.PageSize

	params := db.GetMerchantDocumentsParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	docs, err := r.db.GetMerchantDocuments(ctx, params)
	if err != nil {
		return nil, merchantdocument_errors.ErrFindAllMerchantDocumentsFailed.WithInternal(err)
	}

	return docs, nil
}

func (r *merchantDocumentQueryRepository) FindByActiveDocuments(ctx context.Context, req *requests.FindAllMerchantDocuments) ([]*db.GetActiveMerchantDocumentsRow, error) {
	offset := (req.Page - 1) * req.PageSize

	params := db.GetActiveMerchantDocumentsParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	docs, err := r.db.GetActiveMerchantDocuments(ctx, params)
	if err != nil {
		return nil, merchantdocument_errors.ErrFindActiveMerchantDocumentsFailed.WithInternal(err)
	}

	return docs, nil
}

func (r *merchantDocumentQueryRepository) FindByTrashedDocuments(ctx context.Context, req *requests.FindAllMerchantDocuments) ([]*db.GetTrashedMerchantDocumentsRow, error) {
	offset := (req.Page - 1) * req.PageSize

	params := db.GetTrashedMerchantDocumentsParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	docs, err := r.db.GetTrashedMerchantDocuments(ctx, params)
	if err != nil {
		return nil, merchantdocument_errors.ErrFindTrashedMerchantDocumentsFailed.WithInternal(err)
	}

	return docs, nil
}

func (r *merchantDocumentQueryRepository) FindByIdDocument(ctx context.Context, id int) (*db.GetMerchantDocumentRow, error) {
	doc, err := r.db.GetMerchantDocument(ctx, int32(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, merchantdocument_errors.ErrFindMerchantDocumentByIdFailed.WithInternal(err)
		}
		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}
	return doc, nil
}
