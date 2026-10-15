package auth

import (
	"context"
	"errors"
	"testing"
)

func TestPasswordConcurrencyBudget(t *testing.T) {
	ctx := context.Background()
	first, err := passwordSlot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer first()
	second, err := passwordSlot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	secondReleased := false
	defer func() {
		if !secondReleased {
			second()
		}
	}()
	service := &Service{}
	if err := service.Register(ctx, "test@example.com", "long password"); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	if _, err := service.Login(ctx, "test@example.com", "long password"); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	if err := service.Reset(ctx, RandomToken(), "long password"); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	second()
	secondReleased = true
	release, err := passwordSlot(ctx)
	if err != nil {
		t.Fatal("slot was not released", err)
	}
	release()
}

func TestResetRejectsMalformedTokenBeforeRepository(t *testing.T) {
	service := &Service{}
	for _, token := range []string{"", "invalid", RandomToken() + "=", "a"} {
		if err := service.Reset(context.Background(), token, "long password"); !errors.Is(err, ErrInvalid) {
			t.Fatal(token, err)
		}
	}
}
