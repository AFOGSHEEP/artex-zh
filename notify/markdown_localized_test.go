package notify

import (
	"strings"
	"testing"
	"unicode"
)

// hanFreeItems 는 CJK 한자가 없는 항목(ASCII 제목·유형·요약)만 만든다.
// 라벨이 중국어로 되돌아가면 렌더 결과 전체에 한자가 생기므로, 데이터가
// 한자 0 일 때만 "출력 전체에 한자 0" 단언이 라벨 회귀를 정확히 잡아낸다.
func hanFreeItems(n int) []Item {
	items := make([]Item, 0, n)
	for i := 0; i < n; i++ {
		items = append(items, Item{
			FindingID: int64(i + 1),
			Name:      "login-flaw",
			VulnClass: "SQLi",
			Severity:  "high",
			Summary:   "SQL injection via q param",
			Assets:    []string{"a.example.com"},
		})
	}
	return items
}

// TestMarkdownTitleLocalized 는 markdown 계열 채널(dingtalk·wecom)과 제목을
// 공유하는 feishu·html·telegram·webhook 이 함께 쓰는 markdownTitle 이 한국어인지
// 검사한다. 하나라도 중국어로 되돌아가면 이 다섯 채널의 제목이 전부 혼재된다.
func TestMarkdownTitleLocalized(t *testing.T) {
	// 다건(汇总): "취약점 요약 · 총 N건"
	got := markdownTitle(Message{Batch: true, Items: hanFreeItems(3)})
	assertZh(t, "markdownTitle(batch)", got)
	if !strings.Contains(got, "漏洞汇总") || !strings.Contains(got, "共 3 条") {
		t.Errorf("多漏洞标题应为 '漏洞汇总 · 共 3 条' 形态，实际 %q", got)
	}
	// 항목 없음: "漏洞通知"
	empty := markdownTitle(Message{})
	assertZh(t, "markdownTitle(empty)", empty)
	if empty != "漏洞通知" {
		t.Errorf("空消息标题应为 '漏洞通知'，实际 %q", empty)
	}
}

// TestMarkdownBatchIntroLocalized 는 다이제스트 머리말(시간창·건수·초과 안내)이
// 한국어인지 검사한다. 항목 데이터가 한자 0 이므로 출력에 한자가 보이면 머리말
// 문구가 중국어로 회귀한 것이다.
func TestMarkdownBatchIntroLocalized(t *testing.T) {
	items := hanFreeItems(3)

	// 시간창 있음: "최근 N분간 신규 취약점 N건"
	win := markdownBatchIntro(Message{Batch: true, WindowMinutes: 30}, items, 3)
	if hasHan(win) {
		t.Errorf("时间窗头部仍残留韩文：%q", win)
	}
	if !strings.Contains(win, "近 30 分钟") || !strings.Contains(win, "新增 3 个漏洞") {
		t.Errorf("时间窗头部应为 '近 30 分钟新增 3 个漏洞' 形态，实际 %q", win)
	}

	// 시간창 없음: "신규 취약점 N건"(분간 표기 없음)
	noWin := markdownBatchIntro(Message{Batch: true}, items, 3)
	if hasHan(noWin) {
		t.Errorf("头部仍残留韩文：%q", noWin)
	}
	if !strings.Contains(noWin, "新增 3 个漏洞") || strings.Contains(noWin, "分钟") {
		t.Errorf("无时间窗头部应为 '新增 3 个漏洞'（不含分钟字样），实际 %q", noWin)
	}

	// 일부만 담겼을 때: "(이 메시지에는 앞 N건만 … 나머지 N건은 다음 메시지에서 …)"
	trunc := markdownBatchIntro(Message{Batch: true, WindowMinutes: 30}, items, 5)
	if hasHan(trunc) {
		t.Errorf("溢出提示仍残留韩文：%q", trunc)
	}
	for _, want := range []string{"前 3 条", "其余 2 条", "下一条消息"} {
		if !strings.Contains(trunc, want) {
			t.Errorf("溢出提示应包含 %q，实际 %q", want, trunc)
		}
	}
}

// TestWriteItemLabelsLocalized 는 단건 상세 렌더(markdown 3채널 공유 경로)의
// 필드 라벨(상태 변경·유형·자산·개요)과 상세 링크가 한국어인지 검사한다.
func TestWriteItemLabelsLocalized(t *testing.T) {
	it := Item{
		Name:       "login-flaw",
		VulnClass:  "SQLi",
		Severity:   "high",
		Summary:    "SQL injection via q param",
		Assets:     []string{"a.example.com"},
		DetailURL:  "https://platform.example/finding/1",
		FromStatus: "pending",
		ToStatus:   "fixed",
	}
	var b strings.Builder
	writeItem(&b, it, "", true)
	got := b.String()

	assertZh(t, "writeItem(single)", got)
	for _, want := range []string{
		"**状态变更**：待处理 → 已修复",
		"**类型**：SQLi",
		"**资产**：a.example.com",
		"**摘要**：SQL injection via q param",
		"[查看详情](https://platform.example/finding/1)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("单漏洞渲染应包含 %q：\n%s", want, got)
		}
	}
}

// TestMarkdownBodyFooterLocalized 는 다건 본문 끝의 플랫폼 입구 링크가
// 한국어인지 검사한다(HomeURL 이 있을 때만 붙는다).
func TestMarkdownBodyFooterLocalized(t *testing.T) {
	m := Message{Batch: true, HomeURL: "https://platform.example", Items: hanFreeItems(2)}
	body, kept := markdownBody(m, 0)
	if kept != 2 {
		t.Fatalf("限额为 0 时两条都应包含，实际 %d", kept)
	}
	if !strings.Contains(body, "[在平台中查看全部](https://platform.example)") {
		t.Errorf("正文末尾应有 '在平台中查看全部' 链接：\n%s", body)
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
