package httpapi

import (
	"context"
	"errors"
	"testing"

	"cal/apps/api/internal/store"
)

type fakePushStore struct {
	due     []store.Entry
	subs    map[string][]store.PushSubscription
	deleted []string
	marked  []string
	dueErr  error
}

func (f *fakePushStore) DueReminders(ctx context.Context) ([]store.Entry, error) {
	return f.due, f.dueErr
}
func (f *fakePushStore) PushSubs(ctx context.Context, userID string) ([]store.PushSubscription, error) {
	return f.subs[userID], nil
}
func (f *fakePushStore) DeletePushSub(ctx context.Context, endpoint string) error {
	f.deleted = append(f.deleted, endpoint)
	return nil
}
func (f *fakePushStore) MarkReminded(ctx context.Context, id string) error {
	f.marked = append(f.marked, id)
	return nil
}

func tstr(s string) *string { return &s }

func TestPushTickDeliversAndMarks(t *testing.T) {
	fs := &fakePushStore{
		due:  []store.Entry{{ID: "e1", Title: "Standup", StartTime: tstr("09:30"), OwnerID: "u1"}},
		subs: map[string][]store.PushSubscription{"u1": {{Endpoint: "ep1"}, {Endpoint: "ep2"}}},
	}
	var sent []string
	pushTick(context.Background(), fs, "pub", "priv", func(p []byte, sub store.PushSubscription, pub, priv string) (int, error) {
		sent = append(sent, sub.Endpoint)
		return 201, nil
	})
	if len(sent) != 2 || len(fs.marked) != 1 || fs.marked[0] != "e1" {
		t.Fatalf("sent=%v marked=%v", sent, fs.marked)
	}
}

func TestPushTickPrunesGoneSubs(t *testing.T) {
	fs := &fakePushStore{
		due:  []store.Entry{{ID: "e1", Title: "x", StartTime: tstr("09:30"), OwnerID: "u1"}},
		subs: map[string][]store.PushSubscription{"u1": {{Endpoint: "dead"}, {Endpoint: "alive"}}},
	}
	pushTick(context.Background(), fs, "p", "v", func(p []byte, sub store.PushSubscription, pub, priv string) (int, error) {
		if sub.Endpoint == "dead" {
			return 410, nil
		}
		return 201, nil
	})
	if len(fs.deleted) != 1 || fs.deleted[0] != "dead" {
		t.Fatalf("deleted=%v", fs.deleted)
	}
	if len(fs.marked) != 1 {
		t.Fatalf("entry should still be marked after partial failure")
	}
}

func TestPushTickSkipsNoSubs(t *testing.T) {
	fs := &fakePushStore{
		due:  []store.Entry{{ID: "e1", OwnerID: "u1"}},
		subs: map[string][]store.PushSubscription{},
	}
	called := false
	pushTick(context.Background(), fs, "p", "v", func(p []byte, s store.PushSubscription, a, b string) (int, error) {
		called = true
		return 201, nil
	})
	if called || len(fs.marked) != 0 {
		t.Fatalf("no subs → no send, no mark; got called=%v marked=%v", called, fs.marked)
	}
}

func TestPushTickSendErrorKeepsSub(t *testing.T) {
	fs := &fakePushStore{
		due:  []store.Entry{{ID: "e1", OwnerID: "u1"}},
		subs: map[string][]store.PushSubscription{"u1": {{Endpoint: "flaky"}}},
	}
	pushTick(context.Background(), fs, "p", "v", func(p []byte, s store.PushSubscription, a, b string) (int, error) {
		return 0, errors.New("connection refused")
	})
	if len(fs.deleted) != 0 {
		t.Fatalf("transient errors must not prune subs")
	}
	if len(fs.marked) != 1 {
		t.Fatalf("entry should still be marked (best-effort delivery)")
	}
}
