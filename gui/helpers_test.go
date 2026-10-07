package gui

import (
	"testing"

	"Four-PlayerArmyChess/game"
	"github.com/lxn/walk"
)

func TestFitImage(t *testing.T) {
	wide := fitImage(walk.Rectangle{X: 0, Y: 0, Width: 2000, Height: 300}, boardSourceWidth/boardSourceHeight)
	if wide.Height != 300 {
		t.Fatalf("wide container: expected height 300, got %d", wide.Height)
	}
	tall := fitImage(walk.Rectangle{X: 0, Y: 0, Width: 400, Height: 2000}, boardSourceWidth/boardSourceHeight)
	if tall.Width != 400 {
		t.Fatalf("tall container: expected width 400, got %d", tall.Width)
	}
}

func TestSeatColorAndMaxInt(t *testing.T) {
	for s := game.Seat(0); s <= game.Seat3; s++ {
		_ = seatColor(s)
	}
	if maxInt(2, 3) != 3 || maxInt(5, 1) != 5 {
		t.Fatal("maxInt is wrong")
	}
}

func TestNodePointRoundTrip(t *testing.T) {
	w := &gameWindow{
		game:      game.NewGame(),
		viewer:    game.Seat3,
		imageRect: walk.Rectangle{X: 10, Y: 20, Width: 1054, Height: 994},
	}
	for _, p := range []game.Position{{X: 6, Y: 6}, {X: 0, Y: 7}, {X: 16, Y: 9}, {X: 7, Y: 16}, {X: 8, Y: 8}} {
		pt := w.nodePoint(p)
		got, ok := w.pointToNode(pt.X, pt.Y)
		if !ok || got != p {
			t.Fatalf("round trip failed for %v -> %v (ok=%v)", p, got, ok)
		}
	}
}

func TestPointToNodeRejectsGap(t *testing.T) {
	w := &gameWindow{
		game:      game.NewGame(),
		imageRect: walk.Rectangle{X: 0, Y: 0, Width: 1054, Height: 994},
	}
	// A point far outside the board should be rejected.
	if _, ok := w.pointToNode(-500, -500); ok {
		t.Fatal("point outside the board should be rejected")
	}
}

func TestVisibleGameAndMoveSource(t *testing.T) {
	g := game.NewGame()
	w := &gameWindow{game: g}
	if w.visibleGame() != g {
		t.Fatal("visibleGame should return the live game")
	}
	if w.currentStep() != 0 {
		t.Fatal("currentStep should be 0 for a fresh game")
	}
	if len(w.moveSource()) != 0 {
		t.Fatal("a fresh game should have no moves")
	}

	g.Turn = game.Seat1
	legal := g.AllLegalMoves(game.Seat1)
	if len(legal) == 0 {
		t.Fatal("expected legal moves")
	}
	g.Move(legal[0].From, legal[0].To)
	rec := g.Record()
	if len(rec.Moves) == 0 {
		t.Fatal("expected a recorded move")
	}
	r := game.NewReplay(rec)
	w.replay = r
	w.replayOn = true
	if w.visibleGame() != r.Game() {
		t.Fatal("visibleGame should return the replay game while replaying")
	}
	if len(w.moveSource()) != len(rec.Moves) {
		t.Fatal("moveSource should return replay moves while replaying")
	}
}

func TestSetLevel(t *testing.T) {
	w := &gameWindow{}
	w.setLevel(game.AIHard)
	if w.level != game.AIHard {
		t.Fatal("setLevel should update the level")
	}
}

func TestBoardImagePath(t *testing.T) {
	if boardImagePath() == "" {
		t.Fatal("board image path should not be empty")
	}
}

func TestNilWidgetGuards(t *testing.T) {
	// refreshMoveLog and onMoveSelected must be safe when no widgets are attached.
	w := &gameWindow{game: game.NewGame()}
	w.refreshMoveLog()
	w.onMoveSelected()
	w.exitReplay()
	w.replayStep(1)
	w.togglePlay()
}
