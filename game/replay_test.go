package game

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func boardSignature(g *Game) string {
	positions := g.Board.Nodes()
	sort.Slice(positions, func(i, j int) bool {
		if positions[i].Y != positions[j].Y {
			return positions[i].Y < positions[j].Y
		}
		return positions[i].X < positions[j].X
	})
	var b strings.Builder
	for _, pos := range positions {
		p := g.Board.GetPiece(pos)
		if p == nil {
			continue
		}
		fmt.Fprintf(&b, "%d,%d:%d:%d;", pos.X, pos.Y, p.Seat, p.Type)
	}
	return b.String()
}

func TestReplayReproducesGame(t *testing.T) {
	g := NewGame()
	rng := rand.New(rand.NewSource(11))
	for i := 0; i < 60 && !g.GameOver; i++ {
		m, ok := g.ChooseMove(AIMedium, rng)
		if !ok {
			break
		}
		g.Move(m.From, m.To)
	}
	if len(g.History) == 0 {
		t.Fatal("没有产生走子记录")
	}
	if len(g.InitialLayout()) != 100 {
		t.Fatalf("初始布局应有 100 枚棋子, got %d", len(g.InitialLayout()))
	}

	rec := g.Record()
	r := NewReplay(rec)
	r.Goto(r.Total())
	if boardSignature(g) != boardSignature(r.Game()) {
		t.Fatal("复盘重建的棋盘与对局不一致")
	}
	if r.Step() != len(rec.Moves) {
		t.Fatalf("复盘步数 = %d, want %d", r.Step(), len(rec.Moves))
	}
}

func TestReplaySaveLoadAndStepping(t *testing.T) {
	g := NewGame()
	rng := rand.New(rand.NewSource(3))
	for i := 0; i < 40 && !g.GameOver; i++ {
		m, ok := g.ChooseMove(AIHard, rng)
		if !ok {
			break
		}
		g.Move(m.From, m.To)
	}
	rec := g.Record()
	path := filepath.Join(t.TempDir(), "game.4gj")
	if err := SaveRecord(path, rec); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadRecord(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Initial) != len(rec.Initial) || len(loaded.Moves) != len(rec.Moves) {
		t.Fatal("存取后记录长度不一致")
	}

	r := NewReplay(loaded)
	if !r.Next() || !r.Next() || r.Step() != 2 {
		t.Fatalf("前进两步后 step=%d", r.Step())
	}
	if !r.Prev() || r.Step() != 1 {
		t.Fatalf("后退一步后 step=%d", r.Step())
	}
	if !r.Prev() || r.Step() != 0 {
		t.Fatalf("回到起点后 step=%d", r.Step())
	}
	if r.Prev() {
		t.Fatal("起点不应能再后退")
	}
}

func TestReplaySkipsInconsistentMoves(t *testing.T) {
	// A record whose move starts from an empty square (rule drift, hand-edited,
	// or a legacy record) must not crash the replay. Regression test for
	// "导入复盘记录后界面异常".
	rec := GameRecord{
		Version: 1,
		Initial: []PieceSetup{
			{Seat: Seat1, Type: Marshal, X: 6, Y: 2},
			{Seat: Seat1, Type: Flag, X: 9, Y: 0},
			{Seat: Seat2, Type: Flag, X: 16, Y: 9},
			{Seat: Seat3, Type: Flag, X: 7, Y: 16},
			{Seat: Seat4, Type: Flag, X: 0, Y: 7},
		},
		Moves: []MoveRecord{
			{Seat: Seat1, FromX: 7, FromY: 2, ToX: 6, ToY: 2}, // empty origin -> skipped
			{Seat: Seat1, FromX: 6, FromY: 2, ToX: 6, ToY: 1}, // valid
		},
	}
	r := NewReplay(rec)
	r.Goto(r.Total())
	if r.Step() != len(rec.Moves) {
		t.Fatalf("step = %d, want %d", r.Step(), len(rec.Moves))
	}
	if r.Game().Board.GetPiece(Position{X: 6, Y: 1}) == nil {
		t.Fatal("the valid move should have been applied after skipping the bad one")
	}
}
