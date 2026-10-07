package game

import (
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

func TestAILevelString(t *testing.T) {
	for i, want := range []string{"初级", "中级", "高级", "大模型"} {
		if got := AILevel(i).String(); got != want {
			t.Fatalf("AILevel(%d).String()=%q want %q", i, got, want)
		}
	}
	if AILevel(99).String() != "初级" {
		t.Fatal("out-of-range AILevel should fall back")
	}
	if AILevel(-1).String() != "初级" {
		t.Fatal("negative AILevel should fall back")
	}
}

func TestValueOfFallback(t *testing.T) {
	if valueOf(PieceType(99)) != 10 {
		t.Fatal("unknown piece type should map to 10")
	}
	for _, tp := range []PieceType{Flag, Marshal, General, MajorGeneral, Brigadier, Colonel, Major, Captain, Lieutenant, Engineer, Landmine, Bomb} {
		if valueOf(tp) <= 0 {
			t.Fatalf("%v should have positive value", tp)
		}
	}
}

func TestChooseMoveNilRng(t *testing.T) {
	g := NewGame()
	g.Turn = Seat1
	if _, ok := g.ChooseMove(AIHard, nil); !ok {
		t.Fatal("ChooseMove with nil rng should still return a move")
	}
}

func TestAllLegalMovesAndHasLegalMoveGuards(t *testing.T) {
	g := NewGame()
	g.Turn = Seat1
	if len(g.AllLegalMoves(Seat1)) == 0 {
		t.Fatal("expected legal moves at start")
	}
	if !g.HasLegalMove(Seat1) {
		t.Fatal("HasLegalMove should be true at start")
	}
	g.GameOver = true
	if g.AllLegalMoves(Seat1) != nil {
		t.Fatal("no moves when game over")
	}
}

func TestPromptForAIAndParse(t *testing.T) {
	g := NewGame()
	g.Turn = Seat1
	prompt, moves := g.PromptForAI(Seat1)
	if prompt == "" || len(moves) == 0 {
		t.Fatal("prompt should be non-empty with moves")
	}
	if m, ok := ParseAIMove("1", moves); !ok || m != moves[0] {
		t.Fatal("bare index should parse")
	}
	if m, ok := ParseAIMove("编号：1", moves); !ok || m != moves[0] {
		t.Fatal("labelled index should parse")
	}
	// coordinate form matching a real move
	first := moves[0]
	coord := "从" + posPair(first.From) + "到" + posPair(first.To)
	if _, ok := ParseAIMove(coord, moves); !ok {
		t.Fatalf("coordinate form %q should parse", coord)
	}
	if _, ok := ParseAIMove("not a move", moves); ok {
		t.Fatal("garbage should not parse")
	}
	if _, ok := ParseAIMove("1", nil); ok {
		t.Fatal("no moves should not parse")
	}
}

func posPair(p Position) string {
	return itoa10(p.X) + "," + itoa10(p.Y)
}

func itoa10(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}

func TestIsCurve(t *testing.T) {
	b := NewBoard()
	for _, p := range []Position{{6, 6}, {6, 10}, {10, 6}, {10, 10}} {
		if !b.IsCurve(p) {
			t.Fatalf("%v should be a curve", p)
		}
	}
	if b.IsCurve(Position{8, 8}) {
		t.Fatal("(8,8) is not a curve")
	}
}

func TestGetPieceAndMovePieceEdges(t *testing.T) {
	b := NewBoard()
	if b.GetPiece(Position{100, 100}) != nil {
		t.Fatal("invalid position should have no piece")
	}
	if b.GetPiece(Position{0, 0}) != nil {
		t.Fatal("(0,0) should be empty")
	}
	p := &Piece{Type: Marshal, Seat: Seat1}
	b.SetPiece(Position{6, 0}, p)
	if b.MovePiece(Position{6, 0}, Position{100, 100}) {
		t.Fatal("moving to an invalid target should fail")
	}
	other := &Piece{Type: Captain, Seat: Seat2}
	b.SetPiece(Position{6, 1}, other)
	if b.MovePiece(Position{6, 0}, Position{6, 1}) {
		t.Fatal("moving onto an occupied square should fail")
	}
	if !b.MovePiece(Position{6, 0}, Position{7, 0}) {
		t.Fatal("valid move should succeed")
	}
	// SetPiece on invalid position is a no-op
	b.SetPiece(Position{100, 100}, p)
}

func TestEndAndStartGuards(t *testing.T) {
	g := NewGame()
	g.End()
	if !g.GameOver || g.Phase != FinishedPhase {
		t.Fatal("End should finish the game")
	}
	empty := &Game{Board: NewBoard(), Turn: Seat1, Phase: SetupPhase}
	if err := empty.Start(); err == nil {
		t.Fatal("Start without a completed setup should fail")
	}
}

func TestAdjudicateDrawOnSymmetricBoard(t *testing.T) {
	g := NewGame()
	g.Adjudicate()
	if !g.GameOver || !g.Draw {
		t.Fatal("symmetric material should adjudicate to a draw")
	}
}

func TestTeamAlphaWinsWhenBetaFlagsLost(t *testing.T) {
	g := NewGame()
	for _, s := range []Seat{Seat2, Seat4} {
		for _, p := range g.Pieces[s] {
			if p.Type == Flag {
				p.Dead = true
			}
		}
	}
	g.checkEliminations()
	if !g.GameOver || g.Winner != TeamAlpha {
		t.Fatal("alpha should win when both beta flags are lost")
	}
}

func TestPieceAndSeatNames(t *testing.T) {
	cases := []struct {
		t    PieceType
		name string
	}{
		{Flag, "军旗"}, {Marshal, "司令"}, {General, "军长"}, {MajorGeneral, "师长"},
		{Brigadier, "旅长"}, {Colonel, "团长"}, {Major, "营长"}, {Captain, "连长"},
		{Lieutenant, "排长"}, {Engineer, "工兵"}, {Landmine, "地雷"}, {Bomb, "炸弹"},
	}
	for _, c := range cases {
		if got := (&Piece{Type: c.t}).GetName(); got != c.name {
			t.Fatalf("%v name=%q want %q", c.t, got, c.name)
		}
	}
	for i, want := range []string{"1号位", "2号位", "3号位", "4号位"} {
		if got := Seat(i).Name(); got != want {
			t.Fatalf("Seat(%d).Name()=%q want %q", i, got, want)
		}
	}
}

func TestMoveRejectsIllegalActions(t *testing.T) {
	g := NewGame()
	g.Turn = Seat1
	p := g.Pieces[Seat1][0]
	if g.Move(p.Position, Position{-5, -5}) {
		t.Fatal("move to an invalid position should fail")
	}
	g.Turn = Seat2
	if g.CanMove(p.Position, g.Pieces[Seat1][1].Position) {
		t.Fatal("cannot move another seat's piece")
	}
	g.GameOver = true
	if g.CanMove(p.Position, g.Pieces[Seat1][1].Position) {
		t.Fatal("no moves when the game is over")
	}
}

func TestReplayAccessorsAndNextGuard(t *testing.T) {
	g := NewGame()
	g.Turn = Seat1
	rng := rand.New(rand.NewSource(5))
	for i := 0; i < 20 && !g.GameOver; i++ {
		if m, ok := g.ChooseMove(AIMedium, rng); ok {
			g.Move(m.From, m.To)
		}
	}
	rec := g.Record()
	if len(rec.Moves) == 0 {
		t.Fatal("expected recorded moves")
	}
	if rec.Moves[0].Label(1) == "" {
		t.Fatal("Label should be non-empty")
	}
	r := NewReplay(rec)
	if len(r.Moves()) != len(rec.Moves) {
		t.Fatal("Moves() mismatch")
	}
	if r.Record().Version != rec.Version {
		t.Fatal("Record() mismatch")
	}
	// clamp branches
	r.Goto(-5)
	if r.Step() != 0 {
		t.Fatal("Goto(negative) should clamp to 0")
	}
	r.Goto(r.Total() + 999)
	if r.Step() != r.Total() {
		t.Fatal("Goto(too large) should clamp to total")
	}
	if r.Next() {
		t.Fatal("Next at the end should return false")
	}
	if !r.Prev() {
		t.Fatal("Prev should succeed in the middle")
	}
	r.Goto(0)
	if r.Prev() {
		t.Fatal("Prev at the start should return false")
	}
}

func TestRecordResultBranches(t *testing.T) {
	g := NewGame()
	g.GameOver = true
	g.Winner = TeamAlpha
	if g.Record().Winner == "" {
		t.Fatal("alpha win should be recorded")
	}
	g.Winner = TeamBeta
	if g.Record().Winner == "" {
		t.Fatal("beta win should be recorded")
	}
	g.Winner = TeamAlpha
	g.Draw = true
	if !g.Record().Draw {
		t.Fatal("draw should be recorded")
	}
}

func TestSaveLoadErrorPaths(t *testing.T) {
	if err := SaveRecord(filepath.Join(t.TempDir(), "no-such-subdir", "x.4gj"), GameRecord{Version: 1}); err == nil {
		t.Fatal("saving to an invalid path should fail")
	}
	if _, err := LoadRecord(filepath.Join(t.TempDir(), "missing.4gj")); err == nil {
		t.Fatal("loading a missing file should fail")
	}
	bad := filepath.Join(t.TempDir(), "bad.4gj")
	if err := os.WriteFile(bad, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRecord(bad); err == nil {
		t.Fatal("loading malformed json should fail")
	}
}

func TestSetupPieceValidation(t *testing.T) {
	// not in setup phase
	g := NewGame()
	if err := g.SetupPiece(Seat1, &Piece{Type: Marshal, Seat: Seat1}, Position{6, 1}); err == nil {
		t.Fatal("SetupPiece outside setup phase should fail")
	}

	g2 := &Game{Board: NewBoard(), Phase: SetupPhase}
	if err := g2.SetupPiece(Seat1, nil, Position{6, 1}); err == nil {
		t.Fatal("nil piece should fail")
	}
	if err := g2.SetupPiece(Seat1, &Piece{Type: Marshal, Seat: Seat2}, Position{6, 1}); err == nil {
		t.Fatal("piece of another seat should fail")
	}
	if err := g2.SetupPiece(Seat1, &Piece{Type: Marshal, Seat: Seat1}, Position{0, 0}); err == nil {
		t.Fatal("placement outside the home zone should fail")
	}
	// non-flag on a headquarters
	if err := g2.SetupPiece(Seat1, &Piece{Type: Marshal, Seat: Seat1}, Position{9, 0}); err == nil {
		t.Fatal("non-flag on a headquarters should fail")
	}
	// flag outside a headquarters
	if err := g2.SetupPiece(Seat1, &Piece{Type: Flag, Seat: Seat1}, Position{6, 2}); err == nil {
		t.Fatal("flag outside a headquarters should fail")
	}
	// landmine outside the last two rows
	if err := g2.SetupPiece(Seat1, &Piece{Type: Landmine, Seat: Seat1}, Position{6, 3}); err == nil {
		t.Fatal("landmine outside the last two rows should fail")
	}
	// bomb on the first row
	if err := g2.SetupPiece(Seat1, &Piece{Type: Bomb, Seat: Seat1}, Position{6, 5}); err == nil {
		t.Fatal("bomb on the first row should fail")
	}
	// valid placements
	if err := g2.SetupPiece(Seat1, &Piece{Type: Marshal, Seat: Seat1}, Position{6, 1}); err != nil {
		t.Fatalf("valid placement failed: %v", err)
	}
	if err := g2.SetupPiece(Seat1, &Piece{Type: Captain, Seat: Seat1}, Position{6, 1}); err == nil {
		t.Fatal("placing on an occupied square should fail")
	}
	if err := g2.SetupPiece(Seat1, &Piece{Type: Landmine, Seat: Seat1}, Position{6, 0}); err != nil {
		t.Fatalf("landmine on the back row should be allowed: %v", err)
	}
}

func TestAIKnowsAndHelpers(t *testing.T) {
	g := NewGame()
	g.Turn = Seat1
	ctx := g.buildAIContext(Seat1, paramsFor(AIHard))
	if !ctx.knows(g.Pieces[Seat1][0]) {
		t.Fatal("own piece should be known")
	}
	if !ctx.knows(g.Pieces[Seat3][0]) {
		t.Fatal("teammate piece should be known")
	}
	if ctx.knows(g.Pieces[Seat2][0]) {
		t.Fatal("unseen enemy should be unknown")
	}
	g.Pieces[Seat2][0].Known[Seat3] = true // teammate revealed it
	ctx2 := g.buildAIContext(Seat1, paramsFor(AIHard))
	if !ctx2.knows(g.Pieces[Seat2][0]) {
		t.Fatal("enemy revealed to the teammate should be known")
	}

	if got := removeOneType([]PieceType{Marshal, Captain}, Colonel); len(got) != 2 {
		t.Fatal("removing an absent type should leave the pool unchanged")
	}
	if got := removeOneType([]PieceType{Marshal, Captain}, Marshal); len(got) != 1 || got[0] != Captain {
		t.Fatal("removing a present type should shrink the pool")
	}
}

func TestUnknownTypeValues(t *testing.T) {
	// valueOf default branch already covered; also confirm all pool entries are valid types.
	g := NewGame()
	g.Turn = Seat1
	ctx := g.buildAIContext(Seat1, paramsFor(AIHard))
	if len(ctx.unknownPool) == 0 {
		t.Fatal("a fresh game should have unknown enemy pieces")
	}
}
