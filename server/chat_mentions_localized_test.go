package server

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"unicode"
)

// TestChatMentionErrorsLocalized guards F3b chat_mentions.go: every user-facing
// @멘션(인용) error response must be Korean (Hangul present, no Chinese Han). The
// wire token labels (chatMentionPattern·chatMentionKinds), the agent-input
// snapshot header, and the truncation markers fed into that snapshot stay in the
// original language by design and are intentionally not checked here.
func TestChatMentionErrorsLocalized(t *testing.T) {
	// Fixed user-facing literals.
	for label, msg := range map[string]string{
		"badID":       errChatMentionBadID,
		"tooMany":     errChatMentionTooMany,
		"badSearch":   errChatMentionBadSearch,
		"dataUnavail": errChatMentionDataUnavail,
		"tooLarge":    errChatMentionTooLarge,
	} {
		assertZhError(t, label, msg)
	}

	// parseChatMentions — a non-positive id inside a valid wire token.
	if _, err := parseChatMentions("@[漏洞#0]"); err == nil {
		t.Fatal("非法引用 ID 不应通过")
	} else {
		assertZhError(t, "parse.badID", err.Error())
	}

	// parseChatMentions — exceeding the per-message mention cap (11 distinct ids).
	var b strings.Builder
	for i := 1; i <= 11; i++ {
		fmt.Fprintf(&b, "@[漏洞#%d] ", i)
	}
	if _, err := parseChatMentions(b.String()); err == nil {
		t.Fatal("超出引用上限不应通过")
	} else {
		assertZhError(t, "parse.tooMany", err.Error())
	}

	// composeChatMentionMessage — a mention is present but the database is nil.
	if _, err := composeChatMentionMessage(nil, "@[漏洞#1]"); err == nil {
		t.Fatal("无 DB 时引用解析不应通过")
	} else {
		assertZhError(t, "compose.dataUnavail", err.Error())
	}

	// searchChatMentions — invalid kind and over-long query are rejected before
	// the database handle is touched, so &Server{} is enough to reach the branch.
	for _, q := range []string{"kind=unsupported", "q=" + url.QueryEscape(strings.Repeat("字", 201))} {
		w := httptest.NewRecorder()
		(&Server{}).searchChatMentions(w, httptest.NewRequest("GET", "/api/chat/mentions?"+q, nil))
		if w.Code != 400 {
			t.Fatalf("期望校验失败(400)却返回 %d（%s）", w.Code, q)
		}
		assertZhError(t, "search."+q, decodeErrorField(t, w.Body.Bytes()))
	}

	// The display label map mirrors the UI mention kinds, and the not-found
	// template resolves to a fully Korean message for a known kind.
	for _, kind := range []string{"finding", "asset", "company", "endpoint", "ip", "app", "root_domain", "subdomain", "service"} {
		if chatMentionKindLabel[kind] == "" {
			t.Fatalf("种类标签缺失：%s", kind)
		}
	}
	assertZhError(t, "notFound", fmt.Sprintf(errChatMentionNotFoundFmt, chatMentionKindLabel["finding"], 7))
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
