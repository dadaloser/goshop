package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"goshop/app/review/srv/internal/data"
	"goshop/app/review/srv/internal/domain"
)

type verifierFunc func(context.Context, int32, string, int32) error

func (f verifierFunc) VerifyCompleted(c context.Context, u int32, o string, g int32) error {
	return f(c, u, o, g)
}

type memoryRepo struct {
	mu       sync.Mutex
	next     uint64
	reviews  map[string]*domain.Review
	byID     map[uint64]*domain.Review
	appended map[uint64]bool
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{reviews: map[string]*domain.Review{}, byID: map[uint64]*domain.Review{}, appended: map[uint64]bool{}}
}
func (r *memoryRepo) Create(_ context.Context, v *domain.Review) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := fmt.Sprintf("%d:%s:%d", v.UserID, v.OrderSN, v.GoodsID)
	if _, ok := r.reviews[key]; ok {
		return data.ErrConflict
	}
	r.next++
	v.ID = r.next
	v.Status = domain.StatusPending
	clone := *v
	r.reviews[key] = &clone
	r.byID[v.ID] = &clone
	return nil
}
func (r *memoryRepo) Get(_ context.Context, id uint64) (*domain.Review, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v := r.byID[id]
	if v == nil {
		return nil, data.ErrNotFound
	}
	c := *v
	return &c, nil
}
func (r *memoryRepo) Append(_ context.Context, u int32, id uint64, c string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	v := r.byID[id]
	if v == nil || v.UserID != u {
		return data.ErrNotFound
	}
	if r.appended[id] {
		return data.ErrConflict
	}
	r.appended[id] = true
	v.Append = domain.ReviewAppend{ReviewID: id, Content: c}
	return nil
}
func (r *memoryRepo) Moderate(_ context.Context, id uint64, d string, a int32, q, x string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	v := r.byID[id]
	if v == nil {
		return data.ErrNotFound
	}
	if v.Status != domain.StatusPending {
		return data.ErrInvalidState
	}
	v.Status = d
	return nil
}
func (r *memoryRepo) Reply(_ context.Context, id uint64, a int32, c, q string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	v := r.byID[id]
	if v == nil {
		return data.ErrNotFound
	}
	if v.Status != domain.StatusApproved {
		return data.ErrInvalidState
	}
	if v.Reply.ID != 0 {
		return data.ErrConflict
	}
	v.Reply = domain.ReviewReply{ID: 1, ReviewID: id, Content: c}
	return nil
}
func (r *memoryRepo) List(context.Context, int32, int32, string, int, int) ([]domain.Review, int64, error) {
	return nil, 0, nil
}
func (r *memoryRepo) RebuildRating(context.Context, int32, int32, string) (*domain.Rating, error) {
	return &domain.Rating{}, nil
}
func (r *memoryRepo) GetRating(context.Context, int32) (*domain.Rating, error) {
	return &domain.Rating{}, nil
}
func (r *memoryRepo) ProcessOutbox(context.Context, int) error { return nil }

func TestReviewRequiresCompletedPurchase(t *testing.T) {
	svc := New(newMemoryRepo(), verifierFunc(func(context.Context, int32, string, int32) error { return ErrPurchaseRequired }))
	if _, err := svc.Create(context.Background(), 1, "order-1", 2, 5, "great"); err != ErrPurchaseRequired {
		t.Fatalf("Create() error=%v", err)
	}
}
func TestConcurrentFirstReviewIsIdempotent(t *testing.T) {
	repo := newMemoryRepo()
	svc := New(repo, verifierFunc(func(context.Context, int32, string, int32) error { return nil }))
	var success, conflict atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.Create(context.Background(), 1, "order-1", 2, 5, "great")
			if err == nil {
				success.Add(1)
			} else if err == data.ErrConflict {
				conflict.Add(1)
			} else {
				t.Errorf("Create() error=%v", err)
			}
		}()
	}
	wg.Wait()
	if success.Load() != 1 || conflict.Load() != 49 {
		t.Fatalf("success=%d conflict=%d", success.Load(), conflict.Load())
	}
}
func TestAppendModerationReplyAndOutOfOrderSafety(t *testing.T) {
	repo := newMemoryRepo()
	svc := New(repo, verifierFunc(func(context.Context, int32, string, int32) error { return nil }))
	v, err := svc.Create(context.Background(), 1, "order-1", 2, 5, "great")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Reply(context.Background(), v.ID, 9, "thanks", "req-0"); err != data.ErrInvalidState {
		t.Fatalf("early reply error=%v", err)
	}
	if _, err = svc.Append(context.Background(), 1, v.ID, "still great"); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Append(context.Background(), 1, v.ID, "again"); err != data.ErrConflict {
		t.Fatalf("duplicate append error=%v", err)
	}
	if _, err = svc.Moderate(context.Background(), v.ID, "APPROVED", 9, "req-1", ""); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Moderate(context.Background(), v.ID, "REJECTED", 9, "req-2", ""); err != data.ErrInvalidState {
		t.Fatalf("second moderation error=%v", err)
	}
	if _, err = svc.Reply(context.Background(), v.ID, 9, "thanks", "req-3"); err != nil {
		t.Fatal(err)
	}
}
