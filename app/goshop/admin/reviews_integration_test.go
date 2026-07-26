package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	rpb "goshop/api/review/v1"
	upbv1 "goshop/api/user/v1"
	"goshop/app/pkg/authz"
	"goshop/gmicro/server/restserver"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestAdminReviewWorkflowEndToEnd(t *testing.T) {
	cfg := newAdminRBACRouteTestConfig()
	userClient := &fakeAdminUserClient{
		authUserResponse: &upbv1.UserAuthResponse{
			User: &upbv1.UserInfoResponse{
				Id:       99,
				Username: "review_admin",
				Status:   string(authz.AccountStatusActive),
			},
			LegacyRole: int32(authz.LegacyUserRoleAdmin),
		},
		staffRolesResponse: &upbv1.StaffRoleListResponse{},
	}
	reviewConn, reviewState := newAdminReviewWorkflowClient(t)
	reviewClient := rpb.NewReviewClient(reviewConn)

	server := restserver.NewServer()
	if err := registerAdminReviewRoutesWithStores(server, cfg, userClient, reviewClient, &fakeAdminRevocationStore{}, &fakeAdminTokenVersionStore{}); err != nil {
		t.Fatalf("registerAdminReviewRoutesWithStores() error = %v", err)
	}

	reviewToken := mustCreateScopedAdminToken(t, cfg.Jwt, 99,
		[]string{string(authz.StaffRoleReview)},
		[]string{string(authz.PermissionReviewModerateAny), string(authz.PermissionReviewReplyAny)},
		string(authz.BusinessDomainReview), "store-a", "")
	adminToken := mustCreateScopedAdminToken(t, cfg.Jwt, 100,
		[]string{string(authz.StaffRoleAdmin)},
		[]string{string(authz.PermissionReviewAggregateRebuild)},
		string(authz.BusinessDomainReview), "store-a", "")

	moderateReq := httptest.NewRequest(http.MethodPost, "/v1/reviews/1/moderate", bytes.NewBufferString(`{"decision":"APPROVED","reason":"looks good"}`))
	moderateReq.Header.Set("Content-Type", "application/json")
	moderateReq.Header.Set("Authorization", "Bearer "+reviewToken)
	moderateReq.Header.Set("X-Resource-Domain", string(authz.BusinessDomainReview))
	moderateReq.Header.Set("X-Store-ID", "store-a")
	moderateRec := httptest.NewRecorder()
	server.ServeHTTP(moderateRec, moderateReq)
	if moderateRec.Code != http.StatusOK {
		t.Fatalf("moderate status = %d, body=%s", moderateRec.Code, moderateRec.Body.String())
	}

	replyReq := httptest.NewRequest(http.MethodPost, "/v1/reviews/1/reply", bytes.NewBufferString(`{"content":"thanks for your feedback"}`))
	replyReq.Header.Set("Content-Type", "application/json")
	replyReq.Header.Set("Authorization", "Bearer "+reviewToken)
	replyReq.Header.Set("X-Resource-Domain", string(authz.BusinessDomainReview))
	replyReq.Header.Set("X-Store-ID", "store-a")
	replyRec := httptest.NewRecorder()
	server.ServeHTTP(replyRec, replyReq)
	if replyRec.Code != http.StatusOK {
		t.Fatalf("reply status = %d, body=%s", replyRec.Code, replyRec.Body.String())
	}

	rebuildReq := httptest.NewRequest(http.MethodPost, "/v1/reviews/ratings/101/rebuild", nil)
	rebuildReq.Header.Set("Authorization", "Bearer "+adminToken)
	rebuildReq.Header.Set("X-Resource-Domain", string(authz.BusinessDomainReview))
	rebuildReq.Header.Set("X-Store-ID", "store-a")
	rebuildReq.Header.Set("X-Admin-Confirm-Token", "confirm-secret")
	rebuildRec := httptest.NewRecorder()
	server.ServeHTTP(rebuildRec, rebuildReq)
	if rebuildRec.Code != http.StatusOK {
		t.Fatalf("rebuild status = %d, body=%s", rebuildRec.Code, rebuildRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/v1/reviews?goods_id=101&status=APPROVED", nil)
	listReq.Header.Set("Authorization", "Bearer "+reviewToken)
	listReq.Header.Set("X-Resource-Domain", string(authz.BusinessDomainReview))
	listReq.Header.Set("X-Store-ID", "store-a")
	listRec := httptest.NewRecorder()
	server.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body=%s", listRec.Code, listRec.Body.String())
	}

	var listBody struct {
		Total int `json:"total"`
		Data  []struct {
			ID            int64  `json:"id"`
			Status        string `json:"status"`
			MerchantReply string `json:"merchantReply"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("json.Unmarshal(list body) error = %v", err)
	}
	if listBody.Total != 1 || len(listBody.Data) != 1 {
		t.Fatalf("list reviews total=%d data=%+v, want one approved review", listBody.Total, listBody.Data)
	}
	if got := listBody.Data[0].Status; got != "APPROVED" {
		t.Fatalf("list review status = %q, want %q", got, "APPROVED")
	}
	if got := listBody.Data[0].MerchantReply; got != "thanks for your feedback" {
		t.Fatalf("list review merchantReply = %q, want %q", got, "thanks for your feedback")
	}

	reviewState.mu.Lock()
	defer reviewState.mu.Unlock()
	if reviewState.rating == nil || reviewState.rating.GetGoodsId() != 101 || reviewState.rating.GetApprovedCount() != 1 || reviewState.rating.GetAverageMilli() != 5000 {
		t.Fatalf("rebuilt rating = %+v, want goods=101 approved=1 averageMilli=5000", reviewState.rating)
	}
	if got := reviewState.reviews[1].GetStatus(); got != "APPROVED" {
		t.Fatalf("review status after workflow = %q, want %q", got, "APPROVED")
	}
	if got := reviewState.reviews[1].GetMerchantReply(); got != "thanks for your feedback" {
		t.Fatalf("review merchant reply after workflow = %q, want %q", got, "thanks for your feedback")
	}
}

type reviewWorkflowState struct {
	mu      sync.Mutex
	reviews map[int64]*rpb.ReviewResponse
	rating  *rpb.RatingResponse
}

type reviewWorkflowGRPCServer struct {
	rpb.UnimplementedReviewServer
	state *reviewWorkflowState
}

func newAdminReviewWorkflowClient(t *testing.T) (*grpc.ClientConn, *reviewWorkflowState) {
	t.Helper()

	state := &reviewWorkflowState{
		reviews: map[int64]*rpb.ReviewResponse{
			1: {
				Id:        1,
				UserId:    11,
				OrderSn:   "review-e2e-order-1",
				GoodsId:   101,
				Rating:    5,
				Content:   "great product",
				Status:    "PENDING",
				CreatedAt: time.Now().Unix(),
			},
		},
	}
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	rpb.RegisterReviewServer(server, &reviewWorkflowGRPCServer{state: state})
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})

	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
	)
	if err != nil {
		t.Fatalf("grpc.DialContext(bufnet) error = %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn, state
}

func (s *reviewWorkflowGRPCServer) ModerateReview(_ context.Context, req *rpb.ModerateReviewRequest) (*rpb.ReviewResponse, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	review := s.state.reviews[req.GetReviewId()]
	if review == nil {
		return nil, grpcInternalError("review not found")
	}
	review.Status = strings.ToUpper(strings.TrimSpace(req.GetDecision()))
	return cloneReviewResponse(review), nil
}

func (s *reviewWorkflowGRPCServer) ReplyReview(_ context.Context, req *rpb.ReplyReviewRequest) (*rpb.ReviewResponse, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	review := s.state.reviews[req.GetReviewId()]
	if review == nil {
		return nil, grpcInternalError("review not found")
	}
	review.MerchantReply = req.GetContent()
	return cloneReviewResponse(review), nil
}

func (s *reviewWorkflowGRPCServer) RebuildRating(_ context.Context, req *rpb.RebuildRatingRequest) (*rpb.RatingResponse, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	var approvedCount int64
	var ratingSum int64
	for _, review := range s.state.reviews {
		if review.GetGoodsId() == req.GetGoodsId() && strings.EqualFold(review.GetStatus(), "APPROVED") {
			approvedCount++
			ratingSum += int64(review.GetRating())
		}
	}
	averageMilli := int32(0)
	if approvedCount > 0 {
		averageMilli = int32((ratingSum * 1000) / approvedCount)
	}
	s.state.rating = &rpb.RatingResponse{
		GoodsId:       req.GetGoodsId(),
		ApprovedCount: approvedCount,
		RatingSum:     ratingSum,
		AverageMilli:  averageMilli,
		RebuiltAt:     time.Now().Unix(),
	}
	return s.state.rating, nil
}

func (s *reviewWorkflowGRPCServer) ListReviews(_ context.Context, req *rpb.ListReviewsRequest) (*rpb.ReviewListResponse, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	resp := &rpb.ReviewListResponse{}
	status := strings.ToUpper(strings.TrimSpace(req.GetStatus()))
	for _, review := range s.state.reviews {
		if req.GetGoodsId() > 0 && review.GetGoodsId() != req.GetGoodsId() {
			continue
		}
		if status != "" && !strings.EqualFold(review.GetStatus(), status) {
			continue
		}
		resp.Data = append(resp.Data, cloneReviewResponse(review))
	}
	resp.Total = int32(len(resp.Data))
	return resp, nil
}

func cloneReviewResponse(in *rpb.ReviewResponse) *rpb.ReviewResponse {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func grpcInternalError(msg string) error {
	return status.Error(codes.Internal, msg)
}
