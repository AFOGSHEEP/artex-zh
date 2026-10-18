package notify

import (
	"strings"
	"testing"
)

// hasHan 은 문자열에 CJK 통합 한자가 하나라도 있으면 true 를 반환한다.
// 번역한 라벨이 중국어로 되돌아가면(회귀) 이 판정이 잡아낸다.
func hasHan(s string) bool {
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

// hasHangul 은 문자열에 한글 음절이 하나라도 있으면 true 를 반환한다.
func hasHangul(s string) bool {
	for _, r := range s {
		if r >= 0xAC00 && r <= 0xD7A3 {
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

// TestSeverityLabelLocalized 는 모든 심각도 enum 이 한국어 라벨로 나오는지 검사한다.
// 라벨은 모든 알림 채널(telegram·email·html·markdown·webhook·feishu·dingtalk·wecom)이
// 공유하므로, 하나라도 중국어로 되돌아가면 전 채널 메시지가 혼재된다.
func TestSeverityLabelLocalized(t *testing.T) {
	// 중국어 라벨(严重/高危/中危/低危) → 한국어(심각/높음/중간/낮음), UI status.severity 정합.
	want := map[string]string{
		"critical": "严重",
		"high":     "高危",
		"medium":   "中危",
		"low":      "低危",
	}
	for sev, label := range want {
		got := SeverityLabel(sev)
		assertZh(t, "SeverityLabel("+sev+")", got)
		if !strings.Contains(got, label) {
			t.Errorf("SeverityLabel(%q)=%q，应包含 %q", sev, got, label)
		}
	}
	// 알 수 없는 심각도는 원문을 그대로 돌려준다(없는 값을 지어내지 않는다).
	if got := SeverityLabel("made_up"); got != "made_up" {
		t.Errorf("未知严重度应原样返回，实际 %q", got)
	}
}

// TestStatusLabelLocalized 는 처치 상태 9종이 전부 한국어로 나오는지 검사한다.
// UI status.finding 네임스페이스(B4a)와 동일 표기여야 상태 변경 알림과 화면이 어긋나지 않는다.
func TestStatusLabelLocalized(t *testing.T) {
	want := map[string]string{
		"pending":        "待处理",
		"in_progress":    "处理中",
		"confirmed":      "已确认",
		"resolved":       "已处理",
		"fixed":          "已修复",
		"false_positive": "误报",
		"ignored":        "忽略",
		"duplicate":      "重复",
		"risk_accepted":  "风险接受",
	}
	for status, label := range want {
		got := StatusLabel(status)
		assertZh(t, "StatusLabel("+status+")", got)
		if got != label {
			t.Errorf("StatusLabel(%q)=%q，应为 %q", status, got, label)
		}
	}
	// 알 수 없는 상태는 원문 그대로 회신(없는 라벨을 지어내지 않는다).
	if got := StatusLabel("weird_status"); got != "weird_status" {
		t.Errorf("未知状态应原样返回，实际 %q", got)
	}
}

// TestItemTitlePlaceholderLocalized 는 이름·유형이 모두 빈 항목의 대체 제목이
// 한국어 자리표시자(이름 없는 취약점)인지 검사한다. 절대 빈 제목을 내보내지 않는다.
func TestItemTitlePlaceholderLocalized(t *testing.T) {
	got := Item{}.Title()
	assertZh(t, "Item{}.Title()", got)
	// 이름이 있으면 그 이름을, 유형만 있으면 유형을 우선한다(대체 로직 미회귀 확인).
	if n := (Item{Name: "登录 SQLi"}).Title(); n != "登录 SQLi" {
		t.Errorf("名称优先逻辑失效，实际 %q", n)
	}
	if v := (Item{VulnClass: "XSS"}).Title(); v != "XSS" {
		t.Errorf("类型回退逻辑失效，实际 %q", v)
	}
}

// TestAssetLineLocalized 는 자산 나열 문구의 구분자·총수 표기가 한국어/ASCII 인지 검사한다.
// 중국어 구두점(、)·等·个 가 되돌아오면 알림 본문에 혼재가 생긴다.
func TestAssetLineLocalized(t *testing.T) {
	// 上限以内：全部列出，用顿号（、）分隔。
	if got := assetLine([]string{"a.example.com", "b.example.com"}, 3); got != "a.example.com、b.example.com" {
		t.Errorf("分隔符应为顿号（、），实际 %q", got)
	}
	// 超过上限：前 limit 个 + 中文「等 N 个」（N 为总数）。
	got := assetLine([]string{"a", "b", "c", "d", "e"}, 2)
	if hasHangul(got) {
		t.Errorf("资产列表仍残留韩文：%q", got)
	}
	if !strings.Contains(got, "等 5 个") {
		t.Errorf("总数 5 应标为 '等 5 个'，实际 %q", got)
	}
}
