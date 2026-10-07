package game

import (
	"math/rand"
	"testing"
)

// TestAISelfPlayIsActive guards against a passive AI (the original
// first-legal-move bot). It does not force a winner: an evenly balanced junqi
// position can legitimately hold, and the GUI ends a no-move stalemate with
// Game.EndDraw.
func TestAISelfPlayIsActive(t *testing.T) {
	for _, level := range []AILevel{AIEasy, AIMedium, AIHard} {
		g := NewGame()
		rng := rand.New(rand.NewSource(7))
		combats, ply, passes := 0, 0, 0
		for !g.GameOver && ply < 4000 {
			move, ok := g.ChooseMove(level, rng)
			if !ok {
				passes++
				if passes >= 4 {
					g.EndDraw()
					break
				}
				g.PassTurn()
				continue
			}
			passes = 0
			if g.Board.GetPiece(move.To) != nil {
				combats++
			}
			g.Move(move.From, move.To)
			ply++
		}
		if combats < 20 {
			t.Fatalf("%v AI 进攻性不足：仅 %d 次吃子", level, combats)
		}
		t.Logf("%v: 手数=%d 吃子=%d gameOver=%v draw=%v", level, ply, combats, g.GameOver, g.Draw)
	}
}
