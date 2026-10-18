package report

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/Autumn-27/artex/db"
)

// containsHangul 返回字符串中残留的韩文字符（ko 版遗留），中文/英文/数字/符号不会命中。
func containsHangul(s string) (rune, bool) {
	for _, r := range s {
		if unicode.Is(unicode.Hangul, r) {
			return r, true
		}
	}
	return 0, false
}

func sampleFindingNode(t *testing.T) *db.Node {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"vulnclass": "SQL Injection",
		"name":      "登录表单 SQL 注入",
		"severity":  "high",
		"summary":   "在登录参数中确认了基于报错的 SQL 注入。",
		"evidence":  map[string]string{"poc": "' OR '1'='1"},
	})
	if err != nil {
		t.Fatalf("payload 序列化失败：%v", err)
	}
	return &db.Node{ID: 1, Kind: "finding", Payload: payload}
}

func sampleDBFinding() *db.DBFinding {
	return &db.DBFinding{
		ID:              123,
		VulnClass:       "SQL Injection",
		Name:            "登录表单 SQL 注入",
		Severity:        "high",
		Summary:         "在登录参数中确认了基于报错的 SQL 注入。",
		Evidence:        "' OR '1'='1",
		Status:          "confirmed",
		Report:          "详细分析正文。",
		CreatedAt:       time.Date(2026, 10, 4, 2, 30, 0, 0, time.UTC),
		EvidenceVersion: 2,
		TaskDescription: "演示目标渗透测试",
		TrafficBindings: []db.FindingTrafficBinding{
			{
				ID:   9,
				Role: "request",
				Note: "注入 payload 发送点",
				Snapshot: db.TrafficEvidenceSnapshot{
					Method: "POST",
					URL:    "https://sandbox.local/login",
					Status: 200,
				},
			},
		},
	}
}

// TestReportMarkdownLocalized 는 탐색 그래프 보고서(report.Markdown)의 골격이 한국어이고
// 중국어 한자가 전혀 없음을 검증한다.
func TestReportMarkdownLocalized(t *testing.T) {
	out := Markdown(Input{
		Title:       "演示任务",
		Goal:        "沙箱全量巡检",
		GeneratedAt: time.Date(2026, 10, 4, 2, 30, 0, 0, time.UTC),
		AssetCounts: map[string]int{"host": 2, "url": 5},
		Findings:    []*db.Node{sampleFindingNode(t)},
	})
	if r, ok := containsHangul(out); ok {
		t.Fatalf("报告骨架仍残留韩文 %q：\n%s", string(r), out)
	}
	for _, want := range []string{
		"# 渗透测试报告 — 演示任务",
		"- **任务目标**：沙箱全量巡检",
		"- **生成时间**：",
		"## 摘要",
		"- 确认发现：**1** 个",
		"## 发现",
		"**PoC / 证据：**",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("报告骨架缺少 %q", want)
		}
	}
}

// TestReportMarkdownEmptyLocalized 는 취약점이 없을 때의 안내 문구가 한국어임을 검증한다.
func TestReportMarkdownEmptyLocalized(t *testing.T) {
	out := Markdown(Input{Title: "空任务", GeneratedAt: time.Now()})
	if r, ok := containsHangul(out); ok {
		t.Fatalf("空报告仍残留韩文 %q：\n%s", string(r), out)
	}
	if !strings.Contains(out, "_本次未确认漏洞。_") {
		t.Errorf("空报告提示语不是中文：\n%s", out)
	}
}

// TestFindingsMarkdownLocalized 는 취약점 요약/상세 Markdown 익스포트 골격과 심각도 라벨이
// 한국어이고 중국어 한자가 없음을 검증한다.
func TestFindingsMarkdownLocalized(t *testing.T) {
	f := sampleDBFinding()
	out := FindingsMarkdown([]*db.DBFinding{f}, time.Date(2026, 10, 4, 2, 30, 0, 0, time.UTC))
	if r, ok := containsHangul(out); ok {
		t.Fatalf("漏洞汇总报告仍残留韩文 %q：\n%s", string(r), out)
	}
	for _, want := range []string{
		"# 漏洞发现汇总报告",
		"- **生成时间**：",
		"- **发现总数**：1 个",
		"## 摘要",
		"| 严重等级 | 数量 |",
		"| 严重 |", "| 高危 |", "| 中危 |", "| 低危 |", // 심각도 라벨(UI status.severity 와 동일)
		"## 漏洞明细",
		"- **类别**：SQL Injection",
		"- **状态**：confirmed",
		"- **所属任务**：演示目标渗透测试",
		"- **发现时间**：",
		"**证据：**",
		"**详细报告：**",
		"## 关联流量证据",
		"证据版本：2；绑定数量：1。",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("汇总报告缺少 %q", want)
		}
	}

	single := SingleFindingMarkdown(f, time.Date(2026, 10, 4, 2, 30, 0, 0, time.UTC))
	if r, ok := containsHangul(single); ok {
		t.Fatalf("单漏洞报告仍残留韩文 %q：\n%s", string(r), single)
	}
	for _, want := range []string{
		"- **严重等级**：high",
		"## 概述",
		"## 证据",
		"## 详细报告",
		"[请求报文](evidence/123/9/request.http)",
		"[响应报文](evidence/123/9/response.http)",
	} {
		if !strings.Contains(single, want) {
			t.Errorf("单漏洞报告缺少 %q", want)
		}
	}
}

// TestFindingsCSVLocalized 는 CSV 익스포트 헤더가 한국어이고 중국어 한자가 없음을 검증한다.
// (BOM 뒤 헤더 행만 한국어면 된다. 데이터 값은 원본 그대로 둔다.)
func TestFindingsCSVLocalized(t *testing.T) {
	raw := FindingsCSV([]*db.DBFinding{sampleDBFinding()})
	out := strings.TrimPrefix(string(raw), "\xEF\xBB\xBF")
	header := strings.SplitN(out, "\n", 2)[0]
	if r, ok := containsHangul(header); ok {
		t.Fatalf("CSV 表头仍残留韩文 %q：%s", string(r), header)
	}
	for _, col := range []string{"ID", "名称", "类别", "严重等级", "状态", "所属任务", "发现时间", "概述", "流量证据数量", "流量证据ID"} {
		if !strings.Contains(header, col) {
			t.Errorf("CSV 表头缺少列 %q：%s", col, header)
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
