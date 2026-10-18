package agent

import (
	"strings"
	"testing"
	"unicode"
)

// A2: wrap-up / settlement 프롬프트 한국어화.
//
// 이 상수들은 run 또는 task 가 단계/시간 예산에 걸려 종료될 때 settlement 단계에서
// 주입되어, 사용자에게 그대로 노출되는 최종 요약을 직접 지시한다. 따라서 (1) 한국어로
// 작성되어야 하고, (2) 중국어(CJK 한자) 잔재가 없어야 하며, (3) 도구 이름과
// "한 문장 순수 텍스트" 같은 지시 의미가 보존되어야 한다.
//
// DB 시드는 wrapup_prompt / task_timeout_wrapup_prompt 를 빈 문자열로 두고(db/db.go 의
// builtin 에이전트 INSERT 는 이 컬럼을 채우지 않는다), 비어 있으면 이 상수로 떨어진다.
// 즉 이 상수들이 wrap-up 문구의 유일한 원천이다.
func TestWrapupPromptsLocalizedToKorean(t *testing.T) {
	all := map[string]string{
		"settleWrapUpPrompt":        settleWrapUpPrompt,
		"plannerWrapUpDefault":      plannerWrapUpDefault,
		"mainAgentWrapUpDefault":    mainAgentWrapUpDefault,
		"genericWrapUpDefault":      genericWrapUpDefault,
		"workerTaskTimeoutDefault":  workerTaskTimeoutDefault,
		"plannerTaskTimeoutDefault": plannerTaskTimeoutDefault,
	}

	hasScript := func(s string, table *unicode.RangeTable) bool {
		for _, r := range s {
			if unicode.Is(table, r) {
				return true
			}
		}
		return false
	}

	for name, p := range all {
		if !hasScript(p, unicode.Han) {
			t.Errorf("%s: 没有任何中文，未完成汉化", name)
		}
		// 工具名是 ASCII；若仍残留 Hangul 说明 ko 版文案没有清理干净。
		if hasScript(p, unicode.Hangul) {
			t.Errorf("%s: 残留韩文，汉化未完成：%q", name, p)
		}
	}

	// 도구 이름은 식별자이므로 번역하지 않고 그대로 보존되어야 한다.
	// worker 계열(per-run·task-timeout)은 record_fact 로 결론을, report_finding 으로
	// 취약점을 쓰고, 마지막에 한 문장 순수 텍스트로 요약하라는 지시를 유지한다.
	mustContain := func(name, p string, subs ...string) {
		for _, s := range subs {
			if !strings.Contains(p, s) {
				t.Errorf("%s：指令语义 %q 应被保留但缺失", name, s)
			}
		}
	}
	mustContain("settleWrapUpPrompt", settleWrapUpPrompt,
		"insert_assets", "record_fact", "report_finding", "一句话", "纯文本")
	mustContain("workerTaskTimeoutDefault", workerTaskTimeoutDefault,
		"insert_assets", "record_fact", "report_finding", "一句话", "纯文本")
	mustContain("genericWrapUpDefault", genericWrapUpDefault, "一句话", "纯文本")
	mustContain("mainAgentWrapUpDefault", mainAgentWrapUpDefault, "一句话", "纯文本")
	// planner 는 요약 문장을 내지 않고(판정만 하고 종료) 의도·목표·할일 도구를 유지한다.
	mustContain("plannerWrapUpDefault", plannerWrapUpDefault, "add_intent", "prove_goal", "TodoWrite")
	mustContain("plannerTaskTimeoutDefault", plannerTaskTimeoutDefault, "prove_goal")
}

// per-run 과 task-timeout 은 의미가 달라야 한다(특히 planner): per-run 은 "이번 라운드만
// 끝난다"이고 task-timeout 은 "작업 전체가 끝난다"이다. 상수 매핑이 바뀌어 섞이면 안 된다.
func TestWrapupDefaultsRouting(t *testing.T) {
	if WrapupDefault("worker") != settleWrapUpPrompt {
		t.Error("worker per-run 默认值不是 settleWrapUpPrompt")
	}
	if WrapupDefault("planner") != plannerWrapUpDefault {
		t.Error("planner per-run 默认值不是 plannerWrapUpDefault")
	}
	if WrapupDefault("mainagent") != mainAgentWrapUpDefault {
		t.Error("mainagent per-run 默认值不是 mainAgentWrapUpDefault")
	}
	// 미등록 키(커스텀 에이전트)는 generic 으로 떨어진다.
	if WrapupDefault("unknown-agent") != genericWrapUpDefault {
		t.Error("未注册 key 没有回落到 genericWrapUpDefault")
	}
	// task-timeout 은 worker/planner 에만 있고, 그 외는 빈 문자열(호출부가 per-run 으로 회귀).
	if TaskTimeoutWrapupDefault("worker") != workerTaskTimeoutDefault {
		t.Error("worker task-timeout 默认值不是 workerTaskTimeoutDefault")
	}
	if TaskTimeoutWrapupDefault("planner") != plannerTaskTimeoutDefault {
		t.Error("planner task-timeout 默认值不是 plannerTaskTimeoutDefault")
	}
	if TaskTimeoutWrapupDefault("mainagent") != "" {
		t.Error("mainagent 不应有 task-timeout 文案（空字符串）")
	}
	// per-run 과 task-timeout 문구가 동일하면 의미 구분이 사라진 것이다.
	if workerTaskTimeoutDefault == settleWrapUpPrompt {
		t.Error("worker 的 per-run 与 task-timeout 文案相同")
	}
	if plannerTaskTimeoutDefault == plannerWrapUpDefault {
		t.Error("planner 的 per-run 与 task-timeout 文案相同")
	}
}
