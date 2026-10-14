package designdiff

import (
	"fmt"
	"strings"
)

var labels = map[string]string{
	"requirements": "요구사항", "system": "시스템", "actors": "액터", "interfaces": "인터페이스", "executables": "실행 단위",
	"modules": "모듈", "groups": "그룹", "members": "그룹 멤버", "enums": "enum", "consts": "상수", "errors": "에러",
	"types": "타입", "contracts": "계약", "handler_kinds": "핸들러 종류", "event_kinds": "이벤트 종류", "events": "이벤트",
	"funcs": "함수", "flows": "흐름",
}

func name(n NodeRef) string {
	if n.Kind == "system" {
		return labels[n.Kind]
	}
	return labels[n.Kind] + " " + code(n.ID)
}

// code는 s를 코드 스팬으로 감싼다. s에 백틱이 있으면 더 긴 백틱 묶음으로 감싼다.
func code(s string) string {
	fence := "`"
	for strings.Contains(s, fence) {
		fence += "`"
	}
	if strings.HasPrefix(s, "`") || strings.HasSuffix(s, "`") {
		return fence + " " + s + " " + fence
	}
	return fence + s + fence
}

// Markdown은 PR 설명에 붙일 변경 요약이다. base는 제목에 쓰는 기준 ref다.
func (r *Report) Markdown(base string) string {
	if r.Empty() {
		return fmt.Sprintf("설계 변경 없음 (%s 대비)\n", base)
	}
	var sections []string
	add := func(title string, lines []string) {
		if len(lines) > 0 {
			sections = append(sections, "### "+title+"\n"+strings.Join(lines, "\n"))
		}
	}
	var set, added, renamed, modified, removed []string
	for _, s := range r.Settings {
		set = append(set, "- "+s)
	}
	for _, n := range r.Added {
		added = append(added, "- "+name(n))
	}
	for _, n := range r.Renamed {
		renamed = append(renamed, "- "+labels[n.Kind]+" "+code(n.From)+" → "+code(n.To))
	}
	for _, m := range r.Modified {
		modified = append(modified, "- "+name(m.NodeRef))
		for _, l := range m.Lines {
			modified = append(modified, "  - "+l)
		}
	}
	for _, n := range r.Removed {
		removed = append(removed, "- "+name(n))
	}
	add("설정", set)
	add("추가", added)
	add("이름 바뀜", renamed)
	add("수정", modified)
	add("삭제", removed)
	return fmt.Sprintf("## 설계 변경 (%s 대비)\n\n%s\n", base, strings.Join(sections, "\n\n"))
}
