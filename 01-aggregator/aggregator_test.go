package aggregator

import (
	"context"
	"errors"
	"testing"
	"time"
)

var errProfile = errors.New("profile is down")

func TestAggregate_HappyPath(t *testing.T) {
	profile := func(ctx context.Context, id int) (string, error) {
		return "Name: Alice", nil
	}

	order := func(ctx context.Context, id int) (string, error) {
		return "Orders: 5", nil
	}

	agg := New(profile, order)
	got, err := agg.Aggregate(context.Background(), 5)

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	want := "User: Alice | Orders: 5"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAggregate_DominoEffect(t *testing.T) {
	profile := func(ctx context.Context, id int) (string, error) {
		return "", errProfile
	}

	order := func(ctx context.Context, id int) (string, error) {
		select {
		case <-time.After(10 * time.Second):
			return "Orders: 5", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	agg := New(profile, order)
	start := time.Now()
	_, err := agg.Aggregate(context.Background(), 5)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errProfile) {
		t.Fatalf("got error %v, want %v", err, errProfile)
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("took %s, want under 100ms", elapsed)
	}

	t.Logf("domino effect took %s", elapsed)
}

func TestAggregate_SlowPoke(t *testing.T) {
	profile := func(ctx context.Context, id int) (string, error) {
		return "Name: Alice", nil
	}

	order := func(ctx context.Context, id int) (string, error) {
		select {
		case <-time.After(2 * time.Second):
			return "Orders: 5", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	agg := New(profile, order, WithTimeout(1*time.Second))
	start := time.Now()
	_, err := agg.Aggregate(context.Background(), 5)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
	if elapsed > 1500*time.Millisecond {
		t.Errorf("took %s, want about 1s", elapsed)
	}

	t.Logf("slow poke took %s", elapsed)
}
