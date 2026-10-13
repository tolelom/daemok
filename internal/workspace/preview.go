package workspace

import (
	"os"
	"path/filepath"
	"sort"
)

// FileStatus는 gen이 생성 파일 하나에 할 일이다. Status는 same(그대로), new(새로 씀), changed(내용이 바뀜),
// delete(설계에 없어 지움), refuse(생성 경로에 생성 헤더 없는 파일 등이 있어 쓰지 않음)다.
type FileStatus struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// Preview는 Plan이 있는 결과에서 생성 파일마다 상태를 경로 순으로 준다. 쓰지 않는다.
func Preview(r Result, root string) []FileStatus {
	status := map[string]FileStatus{}
	for _, f := range r.Files {
		status[f.Path] = FileStatus{Path: f.Path, Status: "same"}
	}
	for _, f := range r.Plan.Write {
		st := "changed"
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(f.Path))); err != nil {
			st = "new"
		}
		status[f.Path] = FileStatus{Path: f.Path, Status: st}
	}
	for _, d := range r.Plan.Delete {
		status[d] = FileStatus{Path: d, Status: "delete"}
	}
	for _, f := range r.Plan.Refuse {
		status[f.Path] = FileStatus{Path: f.Path, Status: "refuse", Reason: f.Reason}
	}
	out := []FileStatus{}
	for _, f := range status {
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// Content는 이번 생성 결과에서 path의 생성할 내용을 준다.
func Content(r Result, path string) ([]byte, bool) {
	for _, f := range r.Files {
		if f.Path == path {
			return f.Content, true
		}
	}
	return nil, false
}
