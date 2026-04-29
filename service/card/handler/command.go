package handler

import (
	"context"
	"time"

	"github.com/MamangRust/microservice-payment-gateway-grpc/service/card/service"
	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/card"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	card_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/card_errors/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type cardCommandService struct {
	pb.UnimplementedCardCommandServiceServer

	cardCommand service.CardCommandService
}

func NewCardCommandHandleGrpc(cardCommand service.CardCommandService) CardCommandService {
	return &cardCommandService{
		cardCommand: cardCommand,
	}
}

func (s *cardCommandService) CreateCard(ctx context.Context, req *pb.CreateCardRequest) (*pb.ApiResponseCard, error) {
	request := requests.CreateCardRequest{
		UserID:       int(req.UserId),
		CardType:     req.CardType,
		ExpireDate:   req.ExpireDate.AsTime(),
		CVV:          req.Cvv,
		CardProvider: req.CardProvider,
	}

	if err := request.Validate(); err != nil {
		return nil, card_errors.ErrGrpcValidateCreateCardRequest
	}

	res, err := s.cardCommand.CreateCard(ctx, &request)
	if err != nil {
		return nil, errors.ToGrpcError(err)
	}

	protoCard := &pb.CardResponse{
		Id:         int32(res.CardID),
		UserId:     int32(res.UserID),
		CardNumber: res.CardNumber,
		CardType:   res.CardType,
		Cvv:        res.Cvv,
		ExpireDate: res.ExpireDate.Time.Format(time.RFC3339),
		CreatedAt:  res.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:  res.UpdatedAt.Time.Format(time.RFC3339),
	}

	return &pb.ApiResponseCard{
		Status:  "success",
		Message: "Successfully created card",
		Data:    protoCard,
	}, nil
}

func (s *cardCommandService) UpdateCard(ctx context.Context, req *pb.UpdateCardRequest) (*pb.ApiResponseCard, error) {
	request := requests.UpdateCardRequest{
		CardID:       int(req.CardId),
		UserID:       int(req.UserId),
		CardType:     req.CardType,
		ExpireDate:   req.ExpireDate.AsTime(),
		CVV:          req.Cvv,
		CardProvider: req.CardProvider,
	}

	if err := request.Validate(); err != nil {
		return nil, card_errors.ErrGrpcValidateUpdateCardRequest
	}

	res, err := s.cardCommand.UpdateCard(ctx, &request)
	if err != nil {
		return nil, errors.ToGrpcError(err)
	}

	protoCard := &pb.CardResponse{
		Id:         int32(res.CardID),
		UserId:     int32(res.UserID),
		CardNumber: res.CardNumber,
		CardType:   res.CardType,
		Cvv:        res.Cvv,
		ExpireDate: res.ExpireDate.Time.Format(time.RFC3339),
		CreatedAt:  res.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:  res.UpdatedAt.Time.Format(time.RFC3339),
	}

	return &pb.ApiResponseCard{
		Status:  "success",
		Message: "Successfully updated card",
		Data:    protoCard,
	}, nil
}

func (s *cardCommandService) TrashedCard(ctx context.Context, req *pb.FindByIdCardRequest) (*pb.ApiResponseCardDeleteAt, error) {
	id := int(req.GetCardId())
	if id == 0 {
		return nil, card_errors.ErrGrpcInvalidCardID
	}

	res, err := s.cardCommand.TrashedCard(ctx, id)
	if err != nil {
		return nil, errors.ToGrpcError(err)
	}

	protoCard := &pb.CardResponseDeleteAt{
		Id:         int32(res.CardID),
		UserId:     int32(res.UserID),
		CardNumber: res.CardNumber,
		CardType:   res.CardType,
		Cvv:        res.Cvv,
		ExpireDate: res.ExpireDate.Time.Format(time.RFC3339),
		CreatedAt:  res.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:  res.UpdatedAt.Time.Format(time.RFC3339),
		DeletedAt:  wrapperspb.String(res.DeletedAt.Time.Format(time.RFC3339)),
	}

	return &pb.ApiResponseCardDeleteAt{
		Status:  "success",
		Message: "Successfully trashed card",
		Data:    protoCard,
	}, nil
}

func (s *cardCommandService) RestoreCard(ctx context.Context, req *pb.FindByIdCardRequest) (*pb.ApiResponseCardDeleteAt, error) {
	id := int(req.GetCardId())
	if id == 0 {
		return nil, card_errors.ErrGrpcInvalidCardID
	}

	res, err := s.cardCommand.RestoreCard(ctx, id)
	if err != nil {
		return nil, errors.ToGrpcError(err)
	}

	protoCard := &pb.CardResponseDeleteAt{
		Id:         int32(res.CardID),
		UserId:     int32(res.UserID),
		CardNumber: res.CardNumber,
		CardType:   res.CardType,
		Cvv:        res.Cvv,
		ExpireDate: res.ExpireDate.Time.Format(time.RFC3339),
		CreatedAt:  res.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:  res.UpdatedAt.Time.Format(time.RFC3339),
		DeletedAt:  wrapperspb.String(res.DeletedAt.Time.Format(time.RFC3339)),
	}

	return &pb.ApiResponseCardDeleteAt{
		Status:  "success",
		Message: "Successfully restored card",
		Data:    protoCard,
	}, nil
}

func (s *cardCommandService) DeleteCardPermanent(ctx context.Context, req *pb.FindByIdCardRequest) (*pb.ApiResponseCardDelete, error) {
	id := int(req.GetCardId())
	if id == 0 {
		return nil, card_errors.ErrGrpcInvalidCardID
	}

	_, err := s.cardCommand.DeleteCardPermanent(ctx, id)
	if err != nil {
		return nil, errors.ToGrpcError(err)
	}

	return &pb.ApiResponseCardDelete{
		Status:  "success",
		Message: "Successfully deleted card",
	}, nil
}

func (s *cardCommandService) RestoreAllCard(ctx context.Context, _ *emptypb.Empty) (*pb.ApiResponseCardAll, error) {
	_, err := s.cardCommand.RestoreAllCard(ctx)
	if err != nil {
		return nil, errors.ToGrpcError(err)
	}

	return &pb.ApiResponseCardAll{
		Status:  "success",
		Message: "Successfully restore card",
	}, nil
}

func (s *cardCommandService) DeleteAllCardPermanent(ctx context.Context, _ *emptypb.Empty) (*pb.ApiResponseCardAll, error) {
	_, err := s.cardCommand.DeleteAllCardPermanent(ctx)
	if err != nil {
		return nil, errors.ToGrpcError(err)
	}

	return &pb.ApiResponseCardAll{
		Status:  "success",
		Message: "Successfully delete card permanent",
	}, nil
}
