package intercept

import (
	"fmt"
	"strings"
	"testing"
	"unicode"
)

// hasHangul reports whether s contains any Hangul syllable.
func hasHangul(s string) bool {
	for _, r := range s {
		if r >= 0xAC00 && r <= 0xD7A3 {
			return true
		}
	}
	return false
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

// assertZh fails if s lacks Han ideographs or still carries Hangul (a
// leftover from the ko edition). Reverting any localized message back to
// Korean trips the Hangul check, so the test is not vacuous.
func assertZh(t *testing.T, label, s string) {
	t.Helper()
	if !hasHan(s) {
		t.Errorf("%s: 没有中文: %q", label, s)
	}
	if hasHangul(s) {
		t.Errorf("%s: 残留韩文: %q", label, s)
	}
}

// TestJudgeActionLabelLocalized checks the three judge verdict labels are Korean
// (허용 / 차단 / 확인 요청, per the glossary) and that an unknown action still
// passes through untranslated.
func TestJudgeActionLabelLocalized(t *testing.T) {
	for _, action := range []string{"allow", "deny", "ask"} {
		assertZh(t, "judgeActionLabel("+action+")", judgeActionLabel(action))
	}
	if got := judgeActionLabel("weird"); got != "weird" {
		t.Errorf("judgeActionLabel(weird) = %q, want passthrough", got)
	}
}

// TestDefaultMessageLocalized checks the rule-derived deny/ask messages are Korean
// and still embed the rule name, while allow stays empty.
func TestDefaultMessageLocalized(t *testing.T) {
	for _, action := range []string{"deny", "ask"} {
		msg := defaultMessage(action, "R1")
		assertZh(t, "defaultMessage("+action+")", msg)
		if !strings.Contains(msg, "R1") {
			t.Errorf("defaultMessage(%s) dropped rule name: %q", action, msg)
		}
	}
	if got := defaultMessage("allow", "R1"); got != "" {
		t.Errorf("defaultMessage(allow) = %q, want empty", got)
	}
}

// TestToolApprovalSummaryLocalized checks the activity summary is Korean and stays
// parseable by transcript.tsx, which extracts the pending id via /\(#(\d+)\)/ and
// the tool name via /工具\s+(\S+)\s+审批/.
func TestToolApprovalSummaryLocalized(t *testing.T) {
	s := fmt.Sprintf(msgToolApprovalRequestFmt, "Bash", 42)
	assertZh(t, "msgToolApprovalRequestFmt", s)
	if !strings.Contains(s, "(#42)") {
		t.Errorf("summary lost the (#N) marker (transcript.tsx pending_id regex): %q", s)
	}
	if !strings.Contains(s, "工具 Bash 审批") {
		t.Errorf("summary lost the '工具 X 审批' shape (transcript.tsx toolName regex): %q", s)
	}
}
