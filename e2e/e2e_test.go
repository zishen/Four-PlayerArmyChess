// Package e2e contains end-to-end tests that drive the game domain through
// complete flows: a full four-seat AI game, save/load, and replay. These are
// meant to be run automatically on commit (see .githooks/pre-commit) and in CI
// so that a change to one rule cannot silently break another flow.
package e2e

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"Four-PlayerArmyChess/game"
)

func boardSig(g *game.Game) string {
	positions := g.Board.Nodes()
	sort.Slice(positions, func(i, j int) bool {
		if positions[i].Y != positions[j].Y {
			return positions[i].Y < positions[j].Y
		}
		return positions[i].X < positions[j].X
	})
	var b strings.Builder
	for _, p := range positions {
		pc := g.Board.GetPiece(p)
		if pc == nil {
			continue
		}
		fmt.Fprintf(&b, "%d,%d:%d:%d;", p.X, p.Y, pc.Seat, pc.Type)
	}
	return b.String()
}

// TestFullAIGameTerminatesReplaysAndRoundTrips plays a whole game driven by the
// AI at each difficulty, then saves, loads and replays it, asserting the replay
// reproduces the final position exactly.
func TestFullAIGameTerminatesReplaysAndRoundTrips(t *testing.T) {
	for _, level := range []game.AILevel{game.AIEasy, game.AIMedium, game.AIHard} {
		g := game.NewGame()
		rng := rand.New(rand.NewSource(7))
		ply, passes, combats := 0, 0, 0
		for !g.GameOver && ply < 4000 {
			move, ok := g.ChooseMove(level, rng)
			if !ok {
				passes++
				if passes >= 4 {
					g.Adjudicate()
					break
				}
				g.PassTurn()
				continue
			}
			passes = 0
			if g.Board.GetPiece(move.To) != nil {
				combats++
			}
			if !g.Move(move.From, move.To) {
				t.Fatalf("%v: AI produced an illegal move %v", level, move)
			}
			ply++
		}
		if !g.GameOver {
			// A balanced self-play can legitimately hold; settle it the same
			// way the GUI does on stalemate so the flow still completes.
			t.Logf("%v: reached the ply cap without a winner; adjudicating", level)
			g.Adjudicate()
		}
		if combats == 0 {
			t.Fatalf("%v: no captures occurred during the whole game", level)
		}

		rec := g.Record()
		path := filepath.Join(t.TempDir(), "game.4gj")
		if err := game.SaveRecord(path, rec); err != nil {
			t.Fatalf("%v: save failed: %v", level, err)
		}
		loaded, err := game.LoadRecord(path)
		if err != nil {
			t.Fatalf("%v: load failed: %v", level, err)
		}
		r := game.NewReplay(loaded)
		r.Goto(r.Total())
		if boardSig(g) != boardSig(r.Game()) {
			t.Fatalf("%v: replay did not reproduce the final position", level)
		}
		t.Logf("%v: plies=%d captures=%d", level, ply, combats)
	}
}

func emptyGame() *game.Game {
	g := game.NewGame()
	for _, p := range g.Board.Nodes() {
		g.Board.SetPiece(p, nil)
	}
	for s := game.Seat1; s <= game.Seat4; s++ {
		g.Pieces[s] = nil
	}
	g.Turn = game.Seat1
	return g
}

func place(g *game.Game, seat game.Seat, p *game.Piece, pos game.Position) {
	p.Seat = seat
	g.Pieces[seat] = append(g.Pieces[seat], p)
	g.Board.SetPiece(pos, p)
}

// TestDocumentedFlowsEndToEnd exercises the rule flows highlighted in
// doc/rules.md through the public game API.
func TestDocumentedFlowsEndToEnd(t *testing.T) {
	t.Run("engineer clears a mine and survives", func(t *testing.T) {
		g := emptyGame()
		eng := &game.Piece{Type: game.Engineer}
		mine := &game.Piece{Type: game.Landmine}
		place(g, game.Seat1, eng, game.Position{X: 6, Y: 1})
		place(g, game.Seat2, mine, game.Position{X: 6, Y: 2})
		for s := game.Seat1; s <= game.Seat4; s++ {
			place(g, s, &game.Piece{Type: game.Flag}, game.Position{X: -1, Y: -1})
		}
		if !g.Move(game.Position{X: 6, Y: 1}, game.Position{X: 6, Y: 2}) {
			t.Fatal("engineer move should be legal")
		}
		if eng.Dead || !mine.Dead {
			t.Fatal("engineer must survive and the mine must be removed")
		}
	})

	t.Run("capturing the flag eliminates the seat", func(t *testing.T) {
		g := emptyGame()
		attacker := &game.Piece{Type: game.Marshal}
		flag := &game.Piece{Type: game.Flag}
		place(g, game.Seat1, attacker, game.Position{X: 16, Y: 8})
		place(g, game.Seat2, flag, g.Board.FlagAnchor(game.Seat2))
		for _, s := range []game.Seat{game.Seat1, game.Seat3, game.Seat4} {
			place(g, s, &game.Piece{Type: game.Flag}, game.Position{X: -1, Y: -1})
		}
		if !g.Move(game.Position{X: 16, Y: 8}, g.Board.FlagAnchor(game.Seat2)) {
			t.Fatal("flag capture move should be legal")
		}
		if !g.Eliminated[game.Seat2] {
			t.Fatal("seat 2 should be eliminated after losing its flag")
		}
		if !attacker.Immobile {
			t.Fatal("the capturing piece should become immobile")
		}
	})

	t.Run("rail cannot step straight onto a road", func(t *testing.T) {
		g := emptyGame()
		piece := &game.Piece{Type: game.Marshal}
		place(g, game.Seat1, piece, game.Position{X: 11, Y: 8})
		for s := game.Seat1; s <= game.Seat4; s++ {
			place(g, s, &game.Piece{Type: game.Flag}, game.Position{X: -1, Y: -1})
		}
		dest := g.LegalDestinations(game.Position{X: 11, Y: 8})
		for _, d := range dest {
			if d == (game.Position{X: 12, Y: 8}) {
				t.Fatal("a piece on a railway must not step straight onto an ordinary road")
			}
		}
	})

	t.Run("rail piece may step into an adjacent camp", func(t *testing.T) {
		g := emptyGame()
		piece := &game.Piece{Type: game.Marshal}
		place(g, game.Seat1, piece, game.Position{X: 10, Y: 12})
		for s := game.Seat1; s <= game.Seat4; s++ {
			place(g, s, &game.Piece{Type: game.Flag}, game.Position{X: -1, Y: -1})
		}
		found := false
		for _, d := range g.LegalDestinations(game.Position{X: 10, Y: 12}) {
			if d == (game.Position{X: 9, Y: 12}) {
				found = true
			}
		}
		if !found {
			t.Fatal("a piece on a railway must be able to step into an adjacent camp")
		}
	})

	t.Run("outermost line is a road, reachable only from the railway", func(t *testing.T) {
		for _, from := range []game.Position{{X: 6, Y: 1}, {X: 15, Y: 6}, {X: 6, Y: 15}, {X: 1, Y: 6}} {
			g := emptyGame()
			piece := &game.Piece{Type: game.Marshal}
			place(g, game.Seat1, piece, from)
			for s := game.Seat1; s <= game.Seat4; s++ {
				place(g, s, &game.Piece{Type: game.Flag}, game.Position{X: -1, Y: -1})
			}
			if !g.Board.IsRailway(from) {
				t.Fatalf("%v should be a railway node", from)
			}
			for _, d := range g.LegalDestinations(from) {
				if !g.Board.IsRailway(d) && !g.Board.IsCamp(d) && !g.Board.IsLastLine(d) {
					t.Fatalf("a railway piece at %v must not step onto an ordinary road %v", from, d)
				}
			}
		}

		// 大本营所在的最外一排必须可达，否则无法夺旗。
		g := emptyGame()
		place(g, game.Seat1, &game.Piece{Type: game.Marshal}, game.Position{X: 9, Y: 1})
		g.Board.SetPiece(game.Position{X: 9, Y: 0}, &game.Piece{Type: game.Flag, Seat: game.Seat2})
		found := false
		for _, d := range g.LegalDestinations(game.Position{X: 9, Y: 1}) {
			if d == (game.Position{X: 9, Y: 0}) {
				found = true
			}
		}
		if !found {
			t.Fatal("a railway piece must be able to step onto the enemy headquarters to capture the flag")
		}
	})
}
