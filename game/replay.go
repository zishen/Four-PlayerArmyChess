package game

import (
	"encoding/json"
	"fmt"
	"os"
)

// PieceSetup is one placed piece in the initial private setup.
type PieceSetup struct {
	Seat Seat      `json:"seat"`
	Type PieceType `json:"type"`
	X    int       `json:"x"`
	Y    int       `json:"y"`
}

// MoveRecord is one recorded move plus its public interaction text.
type MoveRecord struct {
	Seat  Seat   `json:"seat"`
	FromX int    `json:"from_x"`
	FromY int    `json:"from_y"`
	ToX   int    `json:"to_x"`
	ToY   int    `json:"to_y"`
	Note  string `json:"note"`
}

// From returns the origin of the move.
func (m MoveRecord) From() Position { return Position{X: m.FromX, Y: m.FromY} }

// To returns the destination of the move.
func (m MoveRecord) To() Position { return Position{X: m.ToX, Y: m.ToY} }

// Label renders the move for the move-log list.
func (m MoveRecord) Label(number int) string {
	return fmt.Sprintf("%d. %s (%d,%d)->(%d,%d) %s", number, m.Seat.Name(), m.FromX, m.FromY, m.ToX, m.ToY, m.Note)
}

// GameRecord is the persisted form of a finished (or in-progress) game.
type GameRecord struct {
	Version int          `json:"version"`
	Initial []PieceSetup `json:"initial"`
	Moves   []MoveRecord `json:"moves"`
	Winner  string       `json:"winner,omitempty"`
	Draw    bool         `json:"draw,omitempty"`
}

// Record snapshots the current game for persistence.
func (g *Game) Record() GameRecord {
	rec := GameRecord{
		Version: 1,
		Initial: g.InitialLayout(),
		Moves:   append([]MoveRecord(nil), g.History...),
	}
	if g.GameOver {
		switch {
		case g.Draw:
			rec.Draw = true
		case g.Winner == TeamAlpha:
			rec.Winner = "α队(1、3号位)"
		case g.Winner == TeamBeta:
			rec.Winner = "β队(2、4号位)"
		}
	}
	return rec
}

// SaveRecord writes a game record to path as JSON.
func SaveRecord(path string, rec GameRecord) error {
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadRecord reads a game record from path.
func LoadRecord(path string) (GameRecord, error) {
	var rec GameRecord
	data, err := os.ReadFile(path)
	if err != nil {
		return rec, err
	}
	if err := json.Unmarshal(data, &rec); err != nil {
		return rec, err
	}
	return rec, nil
}

// newGameFromSetup rebuilds a game from a recorded setup layout.
func newGameFromSetup(setup []PieceSetup) *Game {
	g := &Game{Board: NewBoard(), Turn: Seat1, Phase: PlayingPhase, Selected: Position{X: -1, Y: -1}}
	for _, ps := range setup {
		p := &Piece{Type: ps.Type, Seat: ps.Seat}
		g.Pieces[ps.Seat] = append(g.Pieces[ps.Seat], p)
		g.Board.SetPiece(Position{X: ps.X, Y: ps.Y}, p)
	}
	return g
}

// Replay steps through a recorded game. All piece types are visible while a
// replay is active.
type Replay struct {
	rec  GameRecord
	game *Game
	step int
}

// NewReplay creates a replay positioned at the initial setup.
func NewReplay(rec GameRecord) *Replay {
	r := &Replay{rec: rec}
	r.rebuild(0)
	return r
}

func (r *Replay) rebuild(step int) {
	if step < 0 {
		step = 0
	}
	if step > len(r.rec.Moves) {
		step = len(r.rec.Moves)
	}
	g := newGameFromSetup(r.rec.Initial)
	for i := 0; i < step; i++ {
		m := r.rec.Moves[i]
		if g.Board.GetPiece(m.From()) == nil {
			// Skip moves that cannot be replayed from the rebuilt position
			// (e.g. records created under older rule sets) so importing any
			// record is safe.
			continue
		}
		g.applyMove(m.From(), m.To())
	}
	g.Phase = ReplayPhase
	g.GameOver = false
	g.Selected = Position{X: -1, Y: -1}
	r.game = g
	r.step = step
}

// Game returns the position at the current step.
func (r *Replay) Game() *Game { return r.game }

// Step returns how many moves have been applied.
func (r *Replay) Step() int { return r.step }

// Total returns the number of recorded moves.
func (r *Replay) Total() int { return len(r.rec.Moves) }

// Moves returns the recorded moves.
func (r *Replay) Moves() []MoveRecord { return r.rec.Moves }

// Record returns the underlying record.
func (r *Replay) Record() GameRecord { return r.rec }

// Goto jumps to an absolute step (0 = initial setup).
func (r *Replay) Goto(step int) { r.rebuild(step) }

// Next advances one move; it reports false when already at the end.
func (r *Replay) Next() bool {
	if r.step >= len(r.rec.Moves) {
		return false
	}
	r.rebuild(r.step + 1)
	return true
}

// Prev goes back one move; it reports false when already at the start.
func (r *Replay) Prev() bool {
	if r.step <= 0 {
		return false
	}
	r.rebuild(r.step - 1)
	return true
}
