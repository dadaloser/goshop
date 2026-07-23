package service

import (
	"context"
	stderrors "errors"

	rpb "goshop/api/review/v1"
	"goshop/app/review/srv/internal/data"
	"goshop/app/review/srv/internal/domain"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCServer struct {
	rpb.UnimplementedReviewServer
	svc *Service
}

func NewGRPCServer(svc *Service) *GRPCServer { return &GRPCServer{svc: svc} }
func (s *GRPCServer) CreateReview(ctx context.Context, r *rpb.CreateReviewRequest) (*rpb.ReviewResponse, error) {
	v, e := s.svc.Create(ctx, r.GetUserId(), r.GetOrderSn(), r.GetGoodsId(), r.GetRating(), r.GetContent())
	return reviewResponse(v), grpcError(e)
}
func (s *GRPCServer) AppendReview(ctx context.Context, r *rpb.AppendReviewRequest) (*rpb.ReviewResponse, error) {
	v, e := s.svc.Append(ctx, r.GetUserId(), uint64(r.GetReviewId()), r.GetContent())
	return reviewResponse(v), grpcError(e)
}
func (s *GRPCServer) ModerateReview(ctx context.Context, r *rpb.ModerateReviewRequest) (*rpb.ReviewResponse, error) {
	v, e := s.svc.Moderate(ctx, uint64(r.GetReviewId()), r.GetDecision(), r.GetActorUserId(), r.GetRequestId(), r.GetReason())
	return reviewResponse(v), grpcError(e)
}
func (s *GRPCServer) ReplyReview(ctx context.Context, r *rpb.ReplyReviewRequest) (*rpb.ReviewResponse, error) {
	v, e := s.svc.Reply(ctx, uint64(r.GetReviewId()), r.GetActorUserId(), r.GetContent(), r.GetRequestId())
	return reviewResponse(v), grpcError(e)
}
func (s *GRPCServer) ListReviews(ctx context.Context, r *rpb.ListReviewsRequest) (*rpb.ReviewListResponse, error) {
	values, total, e := s.svc.List(ctx, r.GetGoodsId(), r.GetUserId(), r.GetStatus(), int(r.GetPage()), int(r.GetPageSize()))
	if e != nil {
		return nil, grpcError(e)
	}
	out := &rpb.ReviewListResponse{Total: int32(total)}
	for i := range values {
		out.Data = append(out.Data, reviewResponse(&values[i]))
	}
	return out, nil
}
func (s *GRPCServer) GetRating(ctx context.Context, r *rpb.GetRatingRequest) (*rpb.RatingResponse, error) {
	v, e := s.svc.GetRating(ctx, r.GetGoodsId())
	return ratingResponse(v), grpcError(e)
}
func (s *GRPCServer) RebuildRating(ctx context.Context, r *rpb.RebuildRatingRequest) (*rpb.RatingResponse, error) {
	v, e := s.svc.RebuildRating(ctx, r.GetGoodsId(), r.GetActorUserId(), r.GetRequestId())
	return ratingResponse(v), grpcError(e)
}
func reviewResponse(v *domain.Review) *rpb.ReviewResponse {
	if v == nil {
		return nil
	}
	return &rpb.ReviewResponse{Id: int64(v.ID), UserId: v.UserID, OrderSn: v.OrderSN, GoodsId: v.GoodsID, Rating: v.Rating, Content: v.Content, Status: v.Status, AppendContent: v.Append.Content, MerchantReply: v.Reply.Content, CreatedAt: v.CreatedAt.Unix()}
}
func ratingResponse(v *domain.Rating) *rpb.RatingResponse {
	if v == nil {
		return nil
	}
	return &rpb.RatingResponse{GoodsId: v.GoodsID, ApprovedCount: v.ApprovedCount, RatingSum: v.RatingSum, AverageMilli: v.AverageMilli, RebuiltAt: v.RebuiltAt.Unix()}
}
func grpcError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case stderrors.Is(err, ErrInvalid):
		return status.Error(codes.InvalidArgument, "invalid review request")
	case stderrors.Is(err, ErrPurchaseRequired):
		return status.Error(codes.FailedPrecondition, "completed purchase required")
	case stderrors.Is(err, data.ErrConflict):
		return status.Error(codes.AlreadyExists, "review operation already exists")
	case stderrors.Is(err, data.ErrNotFound):
		return status.Error(codes.NotFound, "review not found")
	case stderrors.Is(err, data.ErrInvalidState):
		return status.Error(codes.FailedPrecondition, "invalid review state")
	default:
		return status.Error(codes.Internal, "review operation failed")
	}
}
