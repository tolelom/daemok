package designdiff

import "testing"

func TestMarkdown(t *testing.T) {
	r := &Report{
		Settings: []string{"tiers: +edge"},
		Added:    []NodeRef{{"modules", "edge"}},
		Renamed:  []Rename{{"types", "post", "article"}},
		Modified: []Modified{{NodeRef{"modules", "web"}, []string{"tier: b → c", "depends_on: +edge -auth"}}, {NodeRef{"system", ""}, []string{"summary 추가: y"}}},
		Removed:  []NodeRef{{"errors", "ErrOld"}},
	}
	want := "## 설계 변경 (HEAD 대비)\n\n### 설정\n- tiers: +edge\n\n### 추가\n- 모듈 `edge`\n\n### 이름 바뀜\n- 타입 `post` → `article`\n\n### 수정\n- 모듈 `web`\n  - tier: b → c\n  - depends_on: +edge -auth\n- 시스템\n  - summary 추가: y\n\n### 삭제\n- 에러 `ErrOld`\n"
	if got := r.Markdown("HEAD"); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	if got := (&Report{}).Markdown("main"); got != "설계 변경 없음 (main 대비)\n" {
		t.Fatal(got)
	}
}

func TestMarkdownCodeSpan(t *testing.T) {
	r := &Report{Added: []NodeRef{{"modules", "a`b"}}}
	if got := r.Markdown("HEAD"); got != "## 설계 변경 (HEAD 대비)\n\n### 추가\n- 모듈 ``a`b``\n" {
		t.Fatal(got)
	}
}
