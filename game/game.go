package game

import "fmt"

type Phase int

const (
	SetupPhase Phase = iota
	PlayingPhase
	FinishedPhase
	ReplayPhase
)

type Game struct {
	Board      *Board
	Turn       Seat
	Phase      Phase
	Pieces     [4][]*Piece
	Selected   Position
	GameOver   bool
	Winner     Team
	Draw       bool
	Eliminated [4]bool
	lastFrom   [4]Position
	lastSet    [4]bool

	// History records every applied move in order; initial stores the setup
	// layout so a game can be saved and replayed.
	History []MoveRecord
	initial []PieceSetup
}

// InitialLayout returns the setup layout captured when the game started.
func (g *Game) InitialLayout() []PieceSetup {
	return append([]PieceSetup(nil), g.initial...)
}

func NewGame() *Game {
	g := &Game{Board: NewBoard(), Turn: Seat1, Phase: SetupPhase, Selected: Position{-1, -1}}
	for s := Seat1; s <= Seat4; s++ {
		for i, t := range pieceLayout() {
			g.Pieces[s] = append(g.Pieces[s], &Piece{ID: int(s)*100 + i, Type: t, Seat: s, Position: Position{-1, -1}})
		}
	}
	g.SetupPieces()
	_ = g.Start()
	return g
}

func (g *Game) Start() error {
	for s := Seat1; s <= Seat4; s++ {
		if !g.SetupComplete(s) {
			return fmt.Errorf("%s尚未完成布阵", s.Name())
		}
	}
	g.Phase = PlayingPhase
	return nil
}

func (g *Game) End() {
	g.GameOver = true
	g.Phase = FinishedPhase
}

func (g *Game) PassTurn() {
	if g.Phase == PlayingPhase && !g.GameOver {
		g.Turn = nextSeat(g.Turn)
	}
}

// EndDraw finishes the game without a winner, used when no seat can move.
func (g *Game) EndDraw() {
	g.GameOver = true
	g.Draw = true
	g.Phase = FinishedPhase
}

// Adjudicate settles a stalemate (no seat can move) by remaining material:
// the team with the higher total piece value wins, then the higher piece
// count, otherwise the game is a draw.
func (g *Game) Adjudicate() {
	var teamValue [2]float64
	var teamCount [2]int
	for s := Seat1; s <= Seat4; s++ {
		for _, p := range g.Pieces[s] {
			if p.Dead || p.Position == (Position{-1, -1}) {
				continue
			}
			team := p.Seat.Team()
			teamValue[team] += valueOf(p.Type)
			if p.Type != Flag {
				teamCount[team]++
			}
		}
	}
	g.GameOver = true
	g.Phase = FinishedPhase
	switch {
	case teamValue[TeamAlpha] > teamValue[TeamBeta]:
		g.Winner = TeamAlpha
	case teamValue[TeamBeta] > teamValue[TeamAlpha]:
		g.Winner = TeamBeta
	case teamCount[TeamAlpha] > teamCount[TeamBeta]:
		g.Winner = TeamAlpha
	case teamCount[TeamBeta] > teamCount[TeamAlpha]:
		g.Winner = TeamBeta
	default:
		g.Draw = true
	}
}

// HasLegalMove reports whether a seat has at least one legal move.
func (g *Game) HasLegalMove(seat Seat) bool {
	return len(g.AllLegalMoves(seat)) > 0
}

func (g *Game) CanMove(from, to Position) bool {
	if g.Phase != PlayingPhase || g.GameOver || !g.Board.IsValidPosition(from) || !g.Board.IsValidPosition(to) {
		return false
	}
	if target := g.Board.GetPiece(to); target != nil && target.Seat.Team() == g.Turn.Team() {
		return false
	}
	p := g.Board.GetPiece(from)
	if p == nil || p.Seat != g.Turn || !p.Movable() {
		return false
	}
	for _, q := range g.LegalDestinations(from) {
		if q == to {
			return true
		}
	}
	return false
}

func (g *Game) LegalDestinations(from Position) []Position {
	p := g.Board.GetPiece(from)
	if p == nil || p.Seat != g.Turn || !p.Movable() {
		return nil
	}
	result := []Position{}
	seen := map[Position]bool{}
	add := func(to Position) {
		if !seen[to] {
			seen[to] = true
			result = append(result, to)
		}
	}
	onRail := g.Board.IsRailway(from)
	// 公路一步：不在铁路上时，可走入任意相邻公路；在铁路上时，本回合须沿铁路
	// 滑行，不能一步踏上相邻普通公路，但有两类例外可一步进入：相邻行营（铁路侧
	// 必须保持可达的安全点），以及本方/他方大本营所在的最外一排（该排只与铁路
	// 相邻，否则任何棋子都无法到达大本营、无法夺旗）。此限制对所有棋子（含工兵）
	// 一律适用。
	for _, e := range g.Board.Edges(from) {
		if e.Route != Road || !g.canOccupyDestination(e.To, p.Seat) {
			continue
		}
		if onRail && !g.Board.IsCamp(e.To) && !g.Board.IsLastLine(e.To) {
			continue
		}
		add(e.To)
	}
	if p.Type == Engineer {
		for _, to := range g.engineerDestinations(from, p.Seat) {
			add(to)
		}
		return result
	}
	for _, to := range g.railwayDestinations(from, p.Seat) {
		add(to)
	}
	return result
}

// canOccupyDestination reports whether a piece of seat may move onto to. A
// square occupied by the seat itself or by its teammate (the opposite seat)
// cannot be entered: allies are never capturable in four-country military
// chess.
func (g *Game) canOccupyDestination(to Position, seat Seat) bool {
	target := g.Board.GetPiece(to)
	if target == nil {
		// An empty headquarters may not be entered; only the enemy flag there
		// can be captured.
		return !g.Board.IsHeadquarters(to)
	}
	return target.Seat.Team() != seat.Team()
}

func (g *Game) railwayDestinations(from Position, seat Seat) []Position {
	result := []Position{}
	seen := map[railState]bool{}
	destinations := map[Position]bool{}
	for _, edge := range g.Board.Edges(from) {
		if edge.Route != Railway || !g.canOccupyDestination(edge.To, seat) {
			continue
		}
		stepX, stepY := sign(edge.To.X-from.X), sign(edge.To.Y-from.Y)
		queue := []railState{{edge.To, stepX, stepY}}
		for len(queue) > 0 {
			state := queue[0]
			queue = queue[1:]
			if seen[state] {
				continue
			}
			seen[state] = true
			target := g.Board.GetPiece(state.pos)
			if g.canOccupyDestination(state.pos, seat) && !destinations[state.pos] {
				result = append(result, state.pos)
				destinations[state.pos] = true
			}
			if target != nil {
				continue
			}
			for _, next := range g.Board.Edges(state.pos) {
				if next.Route != Railway {
					continue
				}
				nx, ny := sign(next.To.X-state.pos.X), sign(next.To.Y-state.pos.Y)
				nextState := railState{next.To, nx, ny}
				// 普通棋子沿铁路直行；只有在“弯道”（四个内角）处，且方向符合
				// 该弯道所连接的两翼（沿物理弧线转弯）时，才可以不停下改变方向；
				// 直角口、或从弯道直插中央都不允许。
				straight := nx == state.dx && ny == state.dy
				if !straight && !g.Board.CurveTurnAllowed(state.pos, state.dx, state.dy, nx, ny) {
					continue
				}
				if !g.canOccupyDestination(next.To, seat) || seen[nextState] {
					continue
				}
				queue = append(queue, nextState)
			}
		}
	}
	return result
}

type railState struct {
	pos    Position
	dx, dy int
}

func sign(value int) int {
	if value < 0 {
		return -1
	}
	if value > 0 {
		return 1
	}
	return 0
}

// engineerDestinations lists every railway point an engineer can reach, turning
// as many times as needed, until blocked by a piece. doc/rules.md: 工兵可以
// 走到没有阻碍的铁轨任意位置.
func (g *Game) engineerDestinations(from Position, seat Seat) []Position {
	seen := map[Position]bool{from: true}
	queue := []Position{from}
	result := []Position{}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		for _, e := range g.Board.Edges(p) {
			if e.Route != Railway || seen[e.To] {
				continue
			}
			target := g.Board.GetPiece(e.To)
			if target != nil {
				if g.canOccupyDestination(e.To, seat) {
					result = append(result, e.To)
				}
				continue
			}
			if !g.canOccupyDestination(e.To, seat) {
				continue
			}
			seen[e.To] = true
			queue = append(queue, e.To)
			result = append(result, e.To)
		}
	}
	return result
}

func (g *Game) Move(from, to Position) bool {
	if !g.CanMove(from, to) {
		return false
	}
	seat := g.Turn
	note := g.applyMove(from, to)
	g.History = append(g.History, MoveRecord{
		Seat: seat, FromX: from.X, FromY: from.Y, ToX: to.X, ToY: to.Y, Note: note,
	})
	return true
}

// applyMove performs a (already validated) move, resolves combat and advances
// the turn. It returns the public interaction text for the move log.
func (g *Game) applyMove(from, to Position) string {
	a := g.Board.GetPiece(from)
	if a == nil {
		// A record built under different rules (or otherwise inconsistent) can
		// contain a move whose origin is empty; skip it instead of panicking.
		return "移动"
	}
	d := g.Board.GetPiece(to)
	note := "移动"
	if d == nil {
		g.Board.MovePiece(from, to)
	} else {
		// Both participants observe each other's piece type.
		a.Known[d.Seat] = true
		d.Known[a.Seat] = true
		switch ResolveCombat(a, d, g.Board.IsCamp(to)) {
		case 1:
			d.Dead = true
			g.Board.SetPiece(to, nil)
			g.Board.MovePiece(from, to)
			switch {
			case d.Type == Flag:
				a.Immobile = true
				note = "占领军旗"
			case d.Type == Landmine:
				note = "工兵排雷"
			default:
				note = "吃子成功"
			}
		case -1:
			a.Dead = true
			g.Board.SetPiece(from, nil)
			if d.Type == Landmine {
				note = "触雷阵亡"
			} else {
				note = "被吃"
			}
		case 0:
			a.Dead = true
			d.Dead = true
			g.Board.SetPiece(from, nil)
			g.Board.SetPiece(to, nil)
			note = "同归于尽"
		case 2:
			note = "攻击无效(行营)"
		}
	}
	// Once a seat's marshal is gone (bombed, mined, or traded with another
	// marshal), that seat's flag position is exposed to everyone.
	if a != nil && a.Type == Marshal && a.Dead {
		g.revealFlag(a.Seat)
	}
	if d != nil && d.Type == Marshal && d.Dead {
		g.revealFlag(d.Seat)
	}
	mover := g.Turn
	g.lastFrom[mover] = from
	g.lastSet[mover] = true
	g.checkEliminations()
	if !g.GameOver {
		g.Turn = nextSeat(g.Turn)
	}
	return note
}

// revealFlag makes a living flag known to every seat.
func (g *Game) revealFlag(seat Seat) {
	for _, p := range g.Pieces[seat] {
		if p.Type == Flag && !p.Dead {
			for i := range p.Known {
				p.Known[i] = true
			}
		}
	}
}

// MarshalDead reports whether a seat's marshal has been destroyed.
func (g *Game) MarshalDead(seat Seat) bool {
	for _, p := range g.Pieces[seat] {
		if p.Type == Marshal && p.Dead {
			return true
		}
	}
	return false
}

func ResolveCombat(a, d *Piece, protected bool) int {
	if protected {
		return 2
	}
	if d.Type == Flag {
		return 1
	}
	if a.Type == Bomb || d.Type == Bomb {
		return 0
	}
	if d.Type == Landmine {
		if a.Type == Engineer {
			// Engineer clears the mine and survives.
			return 1
		}
		// Any other non-bomb piece is destroyed; the landmine stays.
		return -1
	}
	if a.GetRank() > d.GetRank() {
		return 1
	}
	if a.GetRank() < d.GetRank() {
		return -1
	}
	return 0
}

func (g *Game) checkEliminations() {
	for s := Seat1; s <= Seat4; s++ {
		alive := false
		for _, p := range g.Pieces[s] {
			if p.Type == Flag && !p.Dead {
				alive = true
			}
		}
		if !alive && !g.Eliminated[s] {
			g.Eliminated[s] = true
			for _, p := range g.Pieces[s] {
				if !p.Dead {
					p.Dead = true
					g.Board.SetPiece(p.Position, nil)
				}
			}
		}
	}
	if g.Eliminated[Seat1] && g.Eliminated[Seat3] {
		g.GameOver = true
		g.Phase = FinishedPhase
		g.Winner = TeamBeta
	}
	if g.Eliminated[Seat2] && g.Eliminated[Seat4] {
		g.GameOver = true
		g.Phase = FinishedPhase
		g.Winner = TeamAlpha
	}
}

type VisiblePiece struct {
	Position Position
	Seat     Seat
	Type     PieceType
	Hidden   bool
}

func (g *Game) VisiblePieces(viewer Seat) []VisiblePiece {
	result := []VisiblePiece{}
	for _, p := range g.Board.Nodes() {
		x := g.Board.GetPiece(p)
		if x == nil || x.Dead {
			continue
		}
		v := VisiblePiece{Position: p, Seat: x.Seat}
		if x.Seat == viewer || g.Phase == ReplayPhase || g.GameOver || (x.Type == Flag && g.MarshalDead(x.Seat)) {
			// The flag of a seat whose marshal has died is public.
			v.Type = x.Type
		} else {
			v.Hidden = true
		}
		result = append(result, v)
	}
	return result
}
