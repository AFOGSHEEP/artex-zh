package server

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"unicode"
)

// chatupload.go 의 채팅 첨부 업로드 API 에러 응답을 한국어로 유지하는 회귀 방어 테스트다.
// 한국어 판정은 F3a 의 assertZhError(한글 포함·중국어 한자 0)를, 응답 본문 추출은
// task_categories 테스트의 decodeErrorField 를 재사용한다(같은 package server).

// TestChatUploadErrorConstantsLocalized 는 응답 상수 7종이 전부 한국어임을 단언한다.
// ...못했습니다: 세 종류는 뒤에 err.Error() 를 이어 붙이는 접두 상수라 끝에 ": " 가 붙지만,
// 한글이 들어 있고 중국어 한자가 없으면 판정을 통과한다.
func TestChatUploadErrorConstantsLocalized(t *testing.T) {
	cases := map[string]string{
		"scope_invalid": errChatUploadScopeInvalid,
		"bad_id":        errChatUploadBadID,
		"task_deleting": errChatUploadTaskDeleting,
		"mkdir":         errChatUploadMkdir,
		"parse":         errChatUploadParse,
		"no_file":       errChatUploadNoFile,
		"save_failed":   errChatUploadSaveFailed,
	}
	for label, msg := range cases {
		assertZhError(t, label, msg)
	}
}

// TestChatUploadHandlerResponsesLocalized 는 DB 를 건드리지 않고 끝나는 chatUpload 경로를
// 실제 HTTP 응답 본문까지 검사한다. scope/id 검증은 Manager 를 건드리기 전에 반환하고,
// no-file 은 scope=session(작업 아님)이라 엔진·DB 없이 임시 디렉터리만으로 끝난다.
// task-deleting 은 goals 테스트와 같은 삭제 장벽(engine.deleting) 으로 409 를 낸다.
func TestChatUploadHandlerResponsesLocalized(t *testing.T) {
	// 필드만 있고 "file" 이 없는 multipart 본문 — ParseMultipartForm 은 통과하되 파일 0건.
	noFileBody := func() (*bytes.Buffer, string) {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		_ = mw.WriteField("other", "x")
		_ = mw.Close()
		return &buf, mw.FormDataContentType()
	}

	t.Run("scope-invalid", func(t *testing.T) {
		s := &Server{}
		req := httptest.NewRequest(http.MethodPost, "/api/chat/upload?scope=bogus&id=x", nil)
		rec := httptest.NewRecorder()
		s.chatUpload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("状态码 = %d，期望 = 400（正文 %q）", rec.Code, rec.Body.String())
		}
		if got := decodeErrorField(t, rec.Body.Bytes()); got != errChatUploadScopeInvalid {
			t.Fatalf("响应文案 = %q，期望 = %q", got, errChatUploadScopeInvalid)
		}
		assertZhError(t, "scope-invalid", errChatUploadScopeInvalid)
	})

	t.Run("bad-id", func(t *testing.T) {
		s := &Server{}
		// scope=session(작업 아님)이라 Manager 를 건드리기 전에 id 검증에서 반환한다.
		req := httptest.NewRequest(http.MethodPost, "/api/chat/upload?scope=session&id=../evil", nil)
		rec := httptest.NewRecorder()
		s.chatUpload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("状态码 = %d，期望 = 400（正文 %q）", rec.Code, rec.Body.String())
		}
		if got := decodeErrorField(t, rec.Body.Bytes()); got != errChatUploadBadID {
			t.Fatalf("响应文案 = %q，期望 = %q", got, errChatUploadBadID)
		}
		assertZhError(t, "bad-id", errChatUploadBadID)
	})

	t.Run("no-file", func(t *testing.T) {
		s := &Server{m: &Manager{dir: t.TempDir()}}
		body, ctype := noFileBody()
		req := httptest.NewRequest(http.MethodPost, "/api/chat/upload?scope=session&id=sess1", body)
		req.Header.Set("Content-Type", ctype)
		rec := httptest.NewRecorder()
		s.chatUpload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("状态码 = %d，期望 = 400（正文 %q）", rec.Code, rec.Body.String())
		}
		if got := decodeErrorField(t, rec.Body.Bytes()); got != errChatUploadNoFile {
			t.Fatalf("响应文案 = %q，期望 = %q", got, errChatUploadNoFile)
		}
		assertZhError(t, "no-file", errChatUploadNoFile)
	})

	t.Run("task-deleting", func(t *testing.T) {
		s := &Server{m: &Manager{tasks: map[string]*Task{"t1": {ID: "t1"}}}, engine: &Engine{}}
		s.engine.deleting.Store("t1", true)
		req := httptest.NewRequest(http.MethodPost, "/api/chat/upload?scope=task&id=t1", nil)
		rec := httptest.NewRecorder()
		s.chatUpload(rec, req)
		if rec.Code != http.StatusConflict {
			t.Fatalf("状态码 = %d，期望 = 409（正文 %q）", rec.Code, rec.Body.String())
		}
		if got := decodeErrorField(t, rec.Body.Bytes()); got != errChatUploadTaskDeleting {
			t.Fatalf("响应文案 = %q，期望 = %q", got, errChatUploadTaskDeleting)
		}
		assertZhError(t, "task-deleting", errChatUploadTaskDeleting)
	})
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
