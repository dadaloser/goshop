package v1

import (
	"context"
	stderrors "errors"
	"sync/atomic"
	"testing"
	"time"

	"goshop/app/goshop/api/internal/data"
	"goshop/app/goshop/api/internal/smscode"
	"goshop/app/pkg/code"
	"goshop/app/pkg/options"
	"goshop/pkg/errors"
	"goshop/pkg/storage"
)

func TestSmsLoginRejectsLockedMobileBeforeCodeLookup(t *testing.T) {
	codes := &fakeSmsCodeStore{}
	attempts := &fakeSmsAttempts{locked: true}
	svc := newSmsTestService(&fakeUserData{}, codes, attempts)

	_, err := svc.SmsLogin(context.Background(), "13800138000", "123456")

	if !errors.IsCode(err, code.ErrSmsVerifyLocked) {
		t.Fatalf("SmsLogin() error = %v, want ErrSmsVerifyLocked", err)
	}
	if codes.getCalled {
		t.Fatal("SmsLogin() read sms code for locked mobile")
	}
}

func TestSmsLoginRecordsFailureForIncorrectCode(t *testing.T) {
	codes := &fakeSmsCodeStore{value: "123456"}
	attempts := &fakeSmsAttempts{}
	svc := newSmsTestService(&fakeUserData{}, codes, attempts)

	_, err := svc.SmsLogin(context.Background(), "13800138000", "654321")

	if !errors.IsCode(err, code.ErrCodeInCorrect) {
		t.Fatalf("SmsLogin() error = %v, want ErrCodeInCorrect", err)
	}
	if attempts.recordMobile != "13800138000" {
		t.Fatalf("record mobile = %q, want 13800138000", attempts.recordMobile)
	}
	if attempts.recordType != smscode.TypeLogin {
		t.Fatalf("record type = %d, want %d", attempts.recordType, smscode.TypeLogin)
	}
}

func TestSmsLoginReturnsLockedWhenFailureReachesThreshold(t *testing.T) {
	codes := &fakeSmsCodeStore{value: "123456"}
	attempts := &fakeSmsAttempts{recordLocked: true}
	svc := newSmsTestService(&fakeUserData{}, codes, attempts)

	_, err := svc.SmsLogin(context.Background(), "13800138000", "654321")

	if !errors.IsCode(err, code.ErrSmsVerifyLocked) {
		t.Fatalf("SmsLogin() error = %v, want ErrSmsVerifyLocked", err)
	}
}

func TestRegisterResetsSmsFailuresOnSuccess(t *testing.T) {
	codes := &fakeSmsCodeStore{value: "123456"}
	attempts := &fakeSmsAttempts{}
	users := &fakeUserData{}
	svc := newSmsTestService(users, codes, attempts)

	got, err := svc.Register(context.Background(), "13800138000", "user@example.com", "user_001", "Strong1!", "tester", "123456")

	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if got.Token == "" {
		t.Fatal("Register() token is empty")
	}
	if users.created.Username != "user_001" {
		t.Fatalf("Register() username = %q, want user_001", users.created.Username)
	}
	if attempts.resetMobile != "13800138000" {
		t.Fatalf("reset mobile = %q, want 13800138000", attempts.resetMobile)
	}
	if attempts.resetType != smscode.TypeRegister {
		t.Fatalf("reset type = %d, want %d", attempts.resetType, smscode.TypeRegister)
	}
	if !codes.deleteCalled {
		t.Fatal("Register() did not delete used sms code")
	}
}

func TestRegisterReturnsContextDeadlineWhenSmsCodeLookupTimesOut(t *testing.T) {
	var active int32
	done := make(chan struct{})
	codes := &fakeSmsCodeStore{
		get: func(ctx context.Context, _ string) (string, error) {
			atomic.AddInt32(&active, 1)
			defer func() {
				atomic.AddInt32(&active, -1)
				close(done)
			}()
			<-ctx.Done()
			return "", ctx.Err()
		},
	}
	attempts := &fakeSmsAttempts{}
	svc := newSmsTestService(&fakeUserData{}, codes, attempts)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := svc.Register(ctx, "13800138000", "user@example.com", "user_001", "Strong1!", "tester", "123456")

	if !stderrors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Register() error = %v, want context deadline exceeded", err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("sms code lookup did not stop after context timeout")
	}
	if got := atomic.LoadInt32(&active); got != 0 {
		t.Fatalf("active sms code lookups = %d, want 0", got)
	}
	if attempts.recordMobile != "" {
		t.Fatalf("record mobile = %q, want empty when backend timed out", attempts.recordMobile)
	}
}

func TestSmsLoginReturnsCodeNotExistForMissingSmsCode(t *testing.T) {
	codes := &fakeSmsCodeStore{getErr: storage.ErrKeyNotFound}
	attempts := &fakeSmsAttempts{}
	svc := newSmsTestService(&fakeUserData{}, codes, attempts)

	_, err := svc.SmsLogin(context.Background(), "13800138000", "123456")

	if !errors.IsCode(err, code.ErrCodeNotExist) {
		t.Fatalf("SmsLogin() error = %v, want ErrCodeNotExist", err)
	}
	if attempts.recordMobile != "13800138000" {
		t.Fatalf("record mobile = %q, want 13800138000", attempts.recordMobile)
	}
}

func newSmsTestService(users *fakeUserData, codes *fakeSmsCodeStore, attempts *fakeSmsAttempts) UserSrv {
	return NewUserService(
		&fakeDataFactory{users: users},
		&options.JwtOptions{
			Realm:      "test",
			Key:        "01234567890123456789012345678901",
			Timeout:    time.Hour,
			MaxRefresh: time.Hour,
		},
		codes,
		nil,
		attempts,
		nil,
	)
}

type fakeSmsCodeStore struct {
	value        string
	getErr       error
	getCalled    bool
	deleteCalled bool
	get          func(context.Context, string) (string, error)
}

func (f *fakeSmsCodeStore) Get(ctx context.Context, key string) (string, error) {
	f.getCalled = true
	if f.get != nil {
		return f.get(ctx, key)
	}
	if f.getErr != nil {
		return "", f.getErr
	}
	return f.value, nil
}

func (f *fakeSmsCodeStore) Set(context.Context, string, string, time.Duration) error {
	return nil
}

func (f *fakeSmsCodeStore) Delete(context.Context, string) bool {
	f.deleteCalled = true
	return true
}

type fakeSmsAttempts struct {
	locked       bool
	recordLocked bool
	recordMobile string
	recordType   uint
	resetMobile  string
	resetType    uint
}

func (f *fakeSmsAttempts) IsLocked(context.Context, string, uint) (bool, error) {
	return f.locked, nil
}

func (f *fakeSmsAttempts) RecordFailure(_ context.Context, mobile string, codeType uint) (bool, error) {
	f.recordMobile = mobile
	f.recordType = codeType
	return f.recordLocked, nil
}

func (f *fakeSmsAttempts) Reset(_ context.Context, mobile string, codeType uint) error {
	f.resetMobile = mobile
	f.resetType = codeType
	return nil
}

var _ smscode.Store = &fakeSmsCodeStore{}
var _ data.UserData = &fakeUserData{}
