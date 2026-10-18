package server

import (
	"fmt"
	"strings"
	"testing"
	"unicode"
)

// TestSkillZipErrorsLocalized guards the F3b skill_zip.go bundle: every user-facing
// skill-upload error response (fsUploadSkill surfaces these via
// writeErr(400, err.Error()), see server_mgmt.go) must be Korean, and no zip-method
// display name may stay Chinese. Reverting any of these literals to Chinese fails here.
func TestSkillZipErrorsLocalized(t *testing.T) {
	// Message constants rendered with sample arguments.
	assertZhError(t, "errSkillZipParse",
		fmt.Errorf(errSkillZipParse, fmt.Errorf("boom")).Error())
	assertZhError(t, "errSkillZipEncrypted",
		fmt.Sprintf(errSkillZipEncrypted, "demo/SKILL.md"))
	assertZhError(t, "errSkillZipUnsupported",
		fmt.Sprintf(errSkillZipUnsupported, "LZMA", 14, "demo/SKILL.md"))

	// A non-zip upload must reach the user as a Korean hint, not a raw stdlib error.
	if _, err := newSkillZipReader([]byte("this is not a zip archive")); err == nil {
		t.Fatal("非 zip 输入应返回错误")
	} else {
		assertZhError(t, "newSkillZipReader", err.Error())
		if !strings.Contains(err.Error(), "压缩包") {
			t.Fatalf("parse error = %q, want '压缩包' 提示", err.Error())
		}
	}

	// No zip-method display name may carry leftover Hangul; the two
	// translated ones (AES, unknown fallback) must be Chinese.
	for m, name := range zipMethodNames {
		for _, r := range name {
			if unicode.Is(unicode.Hangul, r) {
				t.Fatalf("zipMethodNames[%d] = %q 仍残留韩文", m, name)
			}
		}
	}
	assertZhError(t, "zipMethodName(AES)", zipMethodName(zipMethodAES))
	assertZhError(t, "zipMethodName(unknown)", zipMethodName(0xffff))
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
