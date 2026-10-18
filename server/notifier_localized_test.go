package server

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Autumn-27/artex/db"
	"unicode"
)

// TestNotifierDeliveryReasonsLocalized guards F17: every delivery failure/defer
// reason that notifier.go writes into notification_deliveries.last_error — which
// notify_api.go echoes back to the "전달 기록" table in delivery-list.tsx — must be
// Korean (Hangul present, no Chinese Han). Diagnostic log.Printf lines stay in the
// original language (Z2) and are intentionally not covered here.
func TestNotifierDeliveryReasonsLocalized(t *testing.T) {
	// Format every constant with ASCII arguments so any leftover Han ideograph is
	// the constant's own, not injected by the test data.
	cases := []struct {
		label string
		msg   string
	}{
		{"channelKindUnregistered", fmt.Sprintf(errDeliveryChannelKindUnregistered, "webhook")},
		{"snapshotUnrenderable", errDeliverySnapshotUnrenderable},
		{"channelLengthCapped", fmt.Sprintf(errDeliveryChannelLengthCapped, 3)},
		{"noDeliveredCount", fmt.Sprintf(errDeliveryNoDeliveredCount, 0)},
		{"retryExhausted", fmt.Sprintf(errDeliveryRetryExhausted, db.MaxNotifyAttempts, errors.New("boom"))},
		{"snapshotEmpty", fmt.Sprintf(errDeliverySnapshotEmpty, 7)},
		{"snapshotParse", fmt.Errorf(errDeliverySnapshotParse, 9, errors.New("unexpected end of JSON input")).Error()},
		{"batchAllUnparseable", fmt.Sprintf(errDeliveryBatchAllUnparseable, 2)},
	}
	for _, c := range cases {
		assertZhError(t, c.label, c.msg)
	}
}

// TestNotifierParseSnapshotErrorsLocalized exercises the real parseSnapshot paths
// that surface as last_error through renderSingle → FailDeliveries. parseSnapshot is
// a package-level pure function, so no DB or Server is needed.
func TestNotifierParseSnapshotErrorsLocalized(t *testing.T) {
	// Empty snapshot.
	if _, err := parseSnapshot(&db.NotificationDelivery{ID: 7}); err == nil {
		t.Fatal("空快照不应无错通过")
	} else {
		assertZhError(t, "parseSnapshot.empty", err.Error())
		if !strings.Contains(err.Error(), "7") {
			t.Fatalf("parseSnapshot.empty: 전달 항목 ID 7 이 메시지에 없습니다: %q", err.Error())
		}
	}
	// Malformed JSON snapshot.
	if _, err := parseSnapshot(&db.NotificationDelivery{ID: 9, Snapshot: []byte("{bad")}); err == nil {
		t.Fatal("损坏 JSON 快照不应无错通过")
	} else {
		assertZhError(t, "parseSnapshot.malformed", err.Error())
	}
}

// TestNotifierRenderBatchAllUnparseableLocalized drives the real renderBatch "every
// snapshot is unparseable" branch. With all snapshots empty, every delivery is
// skipped before itemFor (and thus before n.pg), so a zero-value Notifier reaches
// the errDeliveryBatchAllUnparseable return without touching the database.
func TestNotifierRenderBatchAllUnparseableLocalized(t *testing.T) {
	n := &Notifier{}
	deliveries := []*db.NotificationDelivery{{ID: 1}, {ID: 2}}
	if _, _, err := n.renderBatch(context.Background(), deliveries, "", 30); err == nil {
		t.Fatal("全部快照不可解析时应报错")
	} else {
		assertZhError(t, "renderBatch.allUnparseable", err.Error())
	}
}

// hasHan reports whether s contains any CJK Han ideograph.
func hasHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}
