package sidequestion

import (
	"errors"
	"fmt"
	"testing"
	"unicode"
)

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

// TestSideQuestionOutputsLocalized guards F21: every user-facing side-question
// (곁질문) error and answer-text literal produced by this package must be Korean.
// These surface through /api/.../side-questions into the chat 곁질문 panel, so a
// revert to Chinese here is a user-visible regression and must fail the build.
func TestSideQuestionOutputsLocalized(t *testing.T) {
	assertZh(t, "ErrContextBudget", ErrContextBudget.Error())
	assertZh(t, "errSideModelInterrupted", errSideModelInterrupted.Error())
	assertZh(t, "errSideNoAnswer", errSideNoAnswer.Error())
	assertZh(t, "msgSideToolUnavailable", msgSideToolUnavailable)
	assertZh(t, "errSideSummaryCallCap", errSideSummaryCallCap.Error())
	assertZh(t, "sideSummaryFailedPrefix", sideSummaryFailedPrefix)
	assertZh(t, "errSideSummaryIncomplete", errSideSummaryIncomplete.Error())
	assertZh(t, "errSideSummaryOverBudget", errSideSummaryOverBudget.Error())
	assertZh(t, "errSideHistoryCursor", errSideHistoryCursor.Error())
	assertZh(t, "errSideCompactionStalled", errSideCompactionStalled.Error())
}

// TestSideQuestionBudgetSentinelPreserved: localizing the message must not break
// callers that branch on the ErrContextBudget sentinel with errors.Is (see
// context_test.go, which relies on it after the recovery path gives up).
func TestSideQuestionBudgetSentinelPreserved(t *testing.T) {
	if !errors.Is(fmt.Errorf("prepare: %w", ErrContextBudget), ErrContextBudget) {
		t.Fatal("ErrContextBudget 센티넬 식별(errors.Is)이 깨졌습니다")
	}
}

// TestSideQuestionBrainPreserved: the agent-brain prompts (the answering
// instruction and the summary instruction) must stay in their benchmarked
// Chinese. BRIEF 현지화 방침은 두뇌 본문을 번역하지 말고 출력 언어만 한국어로
// 강제하라는 것이라, 이 두 프롬프트가 한국어로 바뀌면 벤치마크 동작이 드리프트한다.
func TestSideQuestionBrainPreserved(t *testing.T) {
	for _, c := range []struct{ label, text string }{
		{"instruction", instruction},
		{"summaryInstruction", summaryInstruction},
	} {
		han := false
		for _, r := range c.text {
			if unicode.Is(unicode.Han, r) {
				han = true
				break
			}
		}
		if !han {
			t.Errorf("%s：大脑提示词不再是中文（基准漂移风险）", c.label)
		}
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
