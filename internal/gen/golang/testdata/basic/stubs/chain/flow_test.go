package chain

import (
	"errors"
	"reflect"
	"testing"
)

type rec struct {
	calls    []string
	failAt   string
	undoFail string
}

func (r *rec) do(name string) error {
	r.calls = append(r.calls, name)
	if name == r.failAt || name == r.undoFail {
		return errors.New("boom")
	}
	return nil
}

func (r *rec) Lock() error                  { return r.do("lock") }
func (r *rec) Unlock() error                { return r.do("unlock") }
func (r *rec) Snapshot() error              { return r.do("snapshot") }
func (r *rec) Revert() error                { return r.do("revert") }
func (r *rec) BufferEvents() error          { return r.do("buffer_events") }
func (r *rec) DiscardEvents() error         { return r.do("discard_events") }
func (r *rec) ExecuteTxs() error            { return r.do("execute_txs") }
func (r *rec) AddBlock() error              { return r.do("add_block") }
func (r *rec) CommitState() error           { return r.do("commit_state") }
func (r *rec) FlushEvents() error           { return r.do("flush_events") }
func (r *rec) EmitBlockCommit() error       { return r.do("emit_block_commit") }
func (r *rec) Fatal(step string, err error) { r.calls = append(r.calls, "FATAL:"+step) }

func TestProduceBlockPaths(t *testing.T) {
	cases := []struct {
		failAt, undoFail string
		want             []string
	}{
		{"snapshot", "", []string{"lock", "snapshot", "unlock"}},
		{"add_block", "", []string{"lock", "snapshot", "buffer_events", "execute_txs", "add_block", "discard_events", "revert", "unlock"}},
		{"add_block", "revert", []string{"lock", "snapshot", "buffer_events", "execute_txs", "add_block", "discard_events", "revert", "FATAL:revert", "unlock"}},
		{"commit_state", "", []string{"lock", "snapshot", "buffer_events", "execute_txs", "add_block", "commit_state", "FATAL:commit_state"}},
		{"flush_events", "", []string{"lock", "snapshot", "buffer_events", "execute_txs", "add_block", "commit_state", "unlock", "flush_events"}},
		{"", "", []string{"lock", "snapshot", "buffer_events", "execute_txs", "add_block", "commit_state", "unlock", "flush_events", "emit_block_commit"}},
	}
	for _, c := range cases {
		r := &rec{failAt: c.failAt, undoFail: c.undoFail}
		err := RunProduceBlock(r)
		if !reflect.DeepEqual(r.calls, c.want) {
			t.Errorf("fail at %q:\n got %v\nwant %v", c.failAt, r.calls, c.want)
		}
		var fe *FlowError
		switch {
		case c.failAt == "" && err != nil:
			t.Errorf("unexpected %v", err)
		case c.failAt != "" && (!errors.As(err, &fe) || fe.Step != c.failAt || fe.Flow != "produce_block"):
			t.Errorf("fail at %q: FlowError가 아니다: %v", c.failAt, err)
		}
	}
}
