package workspace

import (
	"os"
	"path/filepath"
	"sort"
)

type FileStatus struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

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

func Content(r Result, path string) ([]byte, bool) {
	for _, f := range r.Files {
		if f.Path == path {
			return f.Content, true
		}
	}
	return nil, false
}
