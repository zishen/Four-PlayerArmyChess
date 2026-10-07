package game

import (
	"fmt"
	"math/rand"
)

func (g *Game) SetupPiece(seat Seat, piece *Piece, pos Position) error {
	if g.Phase != SetupPhase {
		return fmt.Errorf("不在布阵阶段")
	}
	if piece == nil || piece.Seat != seat {
		return fmt.Errorf("棋子不属于该席位")
	}
	if !g.Board.IsValidPosition(pos) || g.Board.Zone(pos) != seat {
		return fmt.Errorf("只能布置在本方区域")
	}
	if g.Board.GetPiece(pos) != nil {
		return fmt.Errorf("交叉点不可叠放棋子")
	}
	if g.Board.IsCamp(pos) {
		return fmt.Errorf("行营不能布子")
	}
	if piece.Type == Flag && !g.Board.IsHeadquarters(pos) {
		return fmt.Errorf("军旗必须位于大本营")
	}
	if piece.Type == Landmine && g.Board.IsHeadquarters(pos) {
		return fmt.Errorf("地雷不能位于大本营")
	}
	if piece.Type != Flag && g.Board.IsHeadquarters(pos) {
		return fmt.Errorf("大本营只能放置军旗")
	}
	if piece.Type == Landmine && !g.isLastTwoRows(seat, pos) {
		return fmt.Errorf("地雷只能位于本方最后两行")
	}
	if piece.Type == Bomb && g.isFirstRow(seat, pos) {
		return fmt.Errorf("炸弹不能位于第一行")
	}
	g.Board.SetPiece(pos, piece)
	return nil
}

func (g *Game) isLastTwoRows(s Seat, p Position) bool {
	switch s {
	case Seat1:
		return p.Y >= 0 && p.Y <= 1
	case Seat2:
		return p.X >= 15 && p.X <= 16
	case Seat3:
		return p.Y >= 15 && p.Y <= 16
	default:
		return p.X >= 0 && p.X <= 1
	}
}
func (g *Game) isFirstRow(s Seat, p Position) bool {
	switch s {
	case Seat1:
		return p.Y == 5
	case Seat2:
		return p.X == 11
	case Seat3:
		return p.Y == 11
	default:
		return p.X == 5
	}
}

func (g *Game) SetupComplete(seat Seat) bool {
	flag := false
	count := 0
	for _, p := range g.Pieces[seat] {
		if !p.Dead && p.Position != (Position{-1, -1}) {
			count++
			if p.Type == Flag {
				flag = true
			}
		}
	}
	return flag && count == len(g.Pieces[seat])
}

// SetupPieces creates a legal private setup for each seat. A network client
// can replace this with SubmitSetup commands without changing the rule model.
func (g *Game) SetupPieces() {
	r := rand.New(rand.NewSource(42))
	for seat := Seat1; seat <= Seat4; seat++ {
		positions := make([]Position, 0, 25)
		for _, p := range g.Board.Nodes() {
			if g.Board.Zone(p) == seat && !g.Board.IsCamp(p) && !g.Board.IsHeadquarters(p) {
				positions = append(positions, p)
			}
		}
		r.Shuffle(len(positions), func(i, j int) { positions[i], positions[j] = positions[j], positions[i] })
		remaining := make([]*Piece, 0, 24)
		var flag *Piece
		for _, p := range g.Pieces[seat] {
			if p.Type == Flag {
				flag = p
			} else {
				remaining = append(remaining, p)
			}
		}
		_ = g.SetupPiece(seat, flag, g.Board.FlagAnchor(seat))
		available := make([]Position, 0, len(positions))
		available = append(available, positions...)
		place := func(kind PieceType, predicate func(Position) bool) {
			for _, piece := range remaining {
				if piece.Type != kind {
					continue
				}
				for i, p := range available {
					if predicate(p) {
						_ = g.SetupPiece(seat, piece, p)
						available = append(available[:i], available[i+1:]...)
						break
					}
				}
			}
		}
		place(Landmine, func(p Position) bool { return g.isLastTwoRows(seat, p) })
		place(Bomb, func(p Position) bool { return !g.isFirstRow(seat, p) })
		for _, piece := range remaining {
			if piece.Position != (Position{-1, -1}) {
				continue
			}
			if len(available) == 0 {
				break
			}
			_ = g.SetupPiece(seat, piece, available[0])
			available = available[1:]
		}
	}
	g.initial = g.initial[:0]
	for s := Seat1; s <= Seat4; s++ {
		for _, p := range g.Pieces[s] {
			g.initial = append(g.initial, PieceSetup{Seat: s, Type: p.Type, X: p.Position.X, Y: p.Position.Y})
		}
	}
}
