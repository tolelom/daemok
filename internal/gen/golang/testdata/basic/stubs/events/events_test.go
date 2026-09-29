package events

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	in := AssetMintedData{AssetID: "a1", Owner: "o", Amount: 1 << 60, Players: []string{"p", "q"}, Kind: MintKindNormal}
	got, err := AssetMintedDataFrom(in.Map())
	if err != nil || !reflect.DeepEqual(got, in) {
		t.Fatalf("direct: %+v %v", got, err)
	}
	raw, _ := json.Marshal(AssetMintedData{AssetID: "a1", Owner: "o", Amount: 7, Players: []string{"p"}, Kind: MintKindBonus}.Map())
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	got, err = AssetMintedDataFrom(m)
	if err != nil || got.Amount != 7 || got.Players[0] != "p" {
		t.Fatalf("json: %+v %v", got, err)
	}
	if _, err := AssetMintedDataFrom(map[string]any{"asset_id": "a"}); err == nil {
		t.Fatal("키가 없으면 오류여야 한다")
	}
	if _, err := BlockCommitDataFrom(map[string]any{"hash": "h", "txs": 1.5, "block": 1}); err == nil {
		t.Fatal("정수가 아닌 txs는 오류여야 한다")
	}
}

func TestOptionalEnumAndRange(t *testing.T) {
	memo := "m"
	in := AssetMintedData{AssetID: "a", Owner: "o", Amount: 3, Players: []string{}, Kind: MintKindBonus, Memo: &memo}
	raw, _ := json.Marshal(in.Map())
	var m map[string]any
	json.Unmarshal(raw, &m)
	got, err := AssetMintedDataFrom(m)
	if err != nil || got.Kind != MintKindBonus || got.Memo == nil || *got.Memo != "m" || got.Count != nil {
		t.Fatalf("json: %+v %v", got, err)
	}
	// optional 키가 없어도 된다
	delete(m, "memo")
	delete(m, "count")
	if got, err := AssetMintedDataFrom(m); err != nil || got.Memo != nil {
		t.Fatalf("optional 없음: %+v %v", got, err)
	}
	// 음수는 uint64가 될 수 없다
	m["amount"] = -1
	if _, err := AssetMintedDataFrom(m); err == nil {
		t.Fatal("음수 amount는 오류여야 한다")
	}
	m["amount"] = 2.5
	if _, err := AssetMintedDataFrom(m); err == nil {
		t.Fatal("정수가 아닌 amount는 오류여야 한다")
	}
}

func TestOptionalManyAndEnumValid(t *testing.T) {
	in := AssetMintedData{AssetID: "a", Owner: "o", Players: []string{}, Kind: MintKindNormal}
	m := in.Map()
	if _, ok := m["tags"]; ok {
		t.Fatal("nil인 optional many는 넣지 않는다")
	}
	got, err := AssetMintedDataFrom(m)
	if err != nil || got.Tags != nil {
		t.Fatalf("%+v %v", got, err)
	}
	m["tags"] = []any{"x"}
	if got, err := AssetMintedDataFrom(m); err != nil || len(got.Tags) != 1 {
		t.Fatalf("%+v %v", got, err)
	}
	m["kind"] = "bogus"
	if _, err := AssetMintedDataFrom(m); err == nil {
		t.Fatal("설계에 없는 enum 값은 오류여야 한다")
	}
}
