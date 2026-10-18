package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode"
)

// findings_groups.go 의 사용자 노출 문구를 한국어로 유지하는 회귀 방어 테스트다.
// 한국어 판정은 F3a 가 만든 assertZhError(한글 포함·중국어 한자 0)를 재사용한다.

// TestDeepenFindingBodyTooLargeLocalized 는 본문 초과(413) 응답이 한국어임을 실제 HTTP 로
// 확인한다. 이 경로는 MaxBytesReader 디코드 단계에서 반환되어 s.m.pg(DB) 에 닿기 전에 끝나므로
// 빈 Server 로도 끝까지 돈다.
func TestDeepenFindingBodyTooLargeLocalized(t *testing.T) {
	s := &Server{}
	body := `{"description":"` + strings.Repeat("a", 33<<10) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/exploration/findings/1/deepen", strings.NewReader(body))
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()
	s.deepenFinding(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("状态码 = %d，期望 = %d（正文 %q）", rec.Code, http.StatusRequestEntityTooLarge, rec.Body.String())
	}
	var resp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应 JSON 失败：%v（正文 %q）", err, rec.Body.String())
	}
	const want = "请求体过大"
	if resp.Error != want {
		t.Fatalf("响应文案 = %q，期望 = %q", resp.Error, want)
	}
	assertZhError(t, "body_too_large", resp.Error)
}

// TestFindingFollowUpAuditSummaryLocalized 는 후속 의도 활동 요약 상수가 한국어임을 단언한다.
// 이 요약을 저장하는 AddFindingFollowUpIntent 는 DB 트랜잭션이 필요해 DB 없는 이 호스트에서
// 끝까지 못 도므로 상수 자체를 단언한다. 이 문구는 에이전트가 읽는 의도 payload(사용자가
// 입력한 description)와 분리된, 활동 타임라인 표시 전용 요약이다.
func TestFindingFollowUpAuditSummaryLocalized(t *testing.T) {
	assertZhError(t, "follow_up_summary", auditFindingFollowUpSummary)
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
