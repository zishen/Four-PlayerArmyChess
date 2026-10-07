package game

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
)

// AILevel selects how the computer controlled seats play.
type AILevel int

const (
	AIEasy AILevel = iota
	AIMedium
	AIHard
	AILLM
)

var aiLevelNames = []string{"初级", "中级", "高级", "大模型"}

func (l AILevel) String() string {
	if int(l) < 0 || int(l) >= len(aiLevelNames) {
		return aiLevelNames[0]
	}
	return aiLevelNames[l]
}

// Move is a single legal action of one piece.
type Move struct {
	From, To Position
}

type aiParams struct {
	aggression float64
	advance    float64
	safety     float64
	defense    float64
	coop       float64
	anticycle  float64
	noise      float64
	randomness float64
}

func paramsFor(level AILevel) aiParams {
	switch level {
	case AIEasy:
		return aiParams{aggression: 0.15, advance: 1.0, safety: 0.4, defense: 0.4, coop: 0.2, anticycle: 0, noise: 26, randomness: 0.35}
	case AIMedium:
		return aiParams{aggression: 0.6, advance: 2.2, safety: 0.9, defense: 0.9, coop: 0.7, anticycle: 8, noise: 7, randomness: 0.06}
	default: // AIHard and AILLM fallback
		return aiParams{aggression: 1.3, advance: 3.2, safety: 0.9, defense: 1.1, coop: 1.0, anticycle: 20, noise: 3, randomness: 0}
	}
}

// valueOf returns the strategic worth of a piece type used by the AI.
func valueOf(t PieceType) float64 {
	switch t {
	case Flag:
		return 1000
	case Marshal:
		return 120
	case General:
		return 95
	case MajorGeneral:
		return 75
	case Brigadier:
		return 55
	case Colonel:
		return 42
	case Major:
		return 30
	case Captain:
		return 20
	case Lieutenant:
		return 14
	case Engineer:
		return 26
	case Landmine:
		return 30
	case Bomb:
		return 45
	default:
		return 10
	}
}

// AllLegalMoves lists every legal move of a seat without changing the turn.
func (g *Game) AllLegalMoves(seat Seat) []Move {
	if g.Phase != PlayingPhase || g.GameOver {
		return nil
	}
	saved := g.Turn
	g.Turn = seat
	moves := make([]Move, 0, 64)
	for _, from := range g.Board.Nodes() {
		p := g.Board.GetPiece(from)
		if p == nil || p.Seat != seat || !movable(p.Type) {
			continue
		}
		for _, to := range g.LegalDestinations(from) {
			moves = append(moves, Move{From: from, To: to})
		}
	}
	g.Turn = saved
	return moves
}

// ChooseMove picks a move for the current turn's seat.
func (g *Game) ChooseMove(level AILevel, rng *rand.Rand) (Move, bool) {
	if level == AILLM {
		level = AIHard
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	moves := g.AllLegalMoves(g.Turn)
	if len(moves) == 0 {
		return Move{}, false
	}
	p := paramsFor(level)
	if p.randomness > 0 && rng.Float64() < p.randomness {
		return moves[rng.Intn(len(moves))], true
	}
	ctx := g.buildAIContext(g.Turn, p)
	best := moves[0]
	bestScore := math.Inf(-1)
	for _, m := range moves {
		s := ctx.score(m)
		s += (rng.Float64() - 0.5) * p.noise
		if s > bestScore {
			bestScore = s
			best = m
		}
	}
	return best, true
}

type enemyThreat struct {
	piece *Piece
	dests map[Position]bool
}

type aiContext struct {
	game               *Game
	seat               Seat
	teammate           Seat
	params             aiParams
	distToEnemyHQ      map[Position]int
	distToOwnFlag      map[Position]int
	distToTeammateFlag map[Position]int
	enemies            []*Piece
	threats            []enemyThreat
	unknownPool        []PieceType
	flagThreatened     bool
	teammateThreatened bool
}

func (g *Game) buildAIContext(seat Seat, p aiParams) *aiContext {
	ctx := &aiContext{game: g, seat: seat, teammate: Seat((int(seat) + 2) % 4), params: p}
	var enemyHQ []Position
	for s := Seat1; s <= Seat4; s++ {
		if s.Team() == seat.Team() {
			continue
		}
		enemyHQ = append(enemyHQ, g.Board.HeadquartersOf(s)...)
	}
	ctx.distToEnemyHQ = g.Board.DistanceField(enemyHQ)
	ctx.distToOwnFlag = g.Board.DistanceField(g.livingFlagPositions(seat))
	ctx.distToTeammateFlag = g.Board.DistanceField(g.teammateFlagPositions(seat))

	saved := g.Turn
	for s := Seat1; s <= Seat4; s++ {
		if s.Team() == seat.Team() {
			continue
		}
		for _, pc := range g.Pieces[s] {
			if pc.Dead || pc.Position == (Position{-1, -1}) {
				continue
			}
			ctx.enemies = append(ctx.enemies, pc)
			g.Turn = s
			dests := map[Position]bool{}
			for _, d := range g.LegalDestinations(pc.Position) {
				dests[d] = true
			}
			ctx.threats = append(ctx.threats, enemyThreat{piece: pc, dests: dests})
		}
	}
	g.Turn = saved

	// Fair information: build the marginal type distribution of an as-yet
	// unidentified enemy piece by removing every type this seat (or its
	// teammate) has learned through an interaction.
	pool := make([]PieceType, 0, 50)
	for s := Seat1; s <= Seat4; s++ {
		if s.Team() == seat.Team() {
			continue
		}
		pool = append(pool, pieceLayout()...)
	}
	for s := Seat1; s <= Seat4; s++ {
		if s.Team() == seat.Team() {
			continue
		}
		for _, pc := range g.Pieces[s] {
			if pc.Known[seat] || pc.Known[ctx.teammate] {
				pool = removeOneType(pool, pc.Type)
			}
		}
	}
	ctx.unknownPool = pool

	ctx.flagThreatened = minDist(ctx.distToOwnFlag, ctx.enemies) <= 7
	ctx.teammateThreatened = minDist(ctx.distToTeammateFlag, ctx.enemies) <= 7
	return ctx
}

func (c *aiContext) knows(p *Piece) bool {
	if p.Seat.Team() == c.seat.Team() {
		return true
	}
	return p.Known[c.seat] || p.Known[c.teammate]
}

func removeOneType(pool []PieceType, t PieceType) []PieceType {
	for i, v := range pool {
		if v == t {
			return append(pool[:i], pool[i+1:]...)
		}
	}
	return pool
}

func (g *Game) livingFlagPositions(seat Seat) []Position {
	for _, pc := range g.Pieces[seat] {
		if pc.Type == Flag && !pc.Dead && pc.Position != (Position{-1, -1}) {
			return []Position{pc.Position}
		}
	}
	return g.Board.HeadquartersOf(seat)
}

func (g *Game) teammateFlagPositions(seat Seat) []Position {
	var result []Position
	for s := Seat1; s <= Seat4; s++ {
		if s != seat && s.Team() == seat.Team() {
			result = append(result, g.livingFlagPositions(s)...)
		}
	}
	return result
}

func minDist(field map[Position]int, pieces []*Piece) int {
	best := 999
	for _, pc := range pieces {
		if d, ok := field[pc.Position]; ok && d < best {
			best = d
		}
	}
	return best
}

func (c *aiContext) score(m Move) float64 {
	g := c.game
	mover := g.Board.GetPiece(m.From)
	target := g.Board.GetPiece(m.To)
	if mover == nil {
		return math.Inf(-1)
	}
	score := 0.0
	if target != nil {
		if c.knows(target) {
			switch ResolveCombat(mover, target, g.Board.IsCamp(m.To)) {
			case 1:
				score += valueOf(target.Type) * (1.2 + c.params.aggression)
				if target.Type == Flag {
					score += 100000
				}
				if target.Type == Landmine && mover.Type == Engineer {
					score += 15
				}
			case -1:
				score -= valueOf(mover.Type) * (1.0 + c.params.aggression)
			case 0:
				score += valueOf(target.Type) - valueOf(mover.Type)
			case 2:
				score -= 10 * (0.3 + c.params.aggression)
			}
		} else {
			score += c.expectedFightScore(mover, m.To)
			// Only the flag may occupy a headquarters, so an unidentified
			// enemy piece sitting on one is the flag: capture it.
			if g.Board.IsHeadquarters(m.To) {
				score += 100000
			}
		}
	} else {
		if g.Board.IsCamp(m.To) {
			score += 6
		}
		if g.Board.IsRailway(m.To) {
			score += 1.5
		}
		// reward moving away from being boxed in: more mobility is good
		if len(g.Board.Edges(m.To)) >= 2 {
			score += 0.5
		}
	}
	// Approach the enemy command in every case; this is what makes the AI
	// advance and apply pressure instead of shuffling at home.
	if d, ok := c.distToEnemyHQ[m.To]; ok {
		score += c.params.advance * float64(22-d)
	}
	// Avoid moving into a square that an enemy can punish next turn.
	score -= c.params.safety * c.expectedLoss(m.To, mover)
	score += c.defenseScore(mover, m)
	score += c.coopScore(mover, target, m)
	// Discourage immediately undoing the previous move; this breaks pointless
	// oscillations and keeps the AI probing instead of shuffling forever.
	if g.lastSet[c.seat] && g.lastFrom[c.seat] == m.To {
		score -= c.params.anticycle
	}
	return score
}

// expectedFightScore averages the outcome of capturing an unidentified enemy
// piece over the remaining possible enemy types (fair information).
func (c *aiContext) expectedFightScore(mover *Piece, to Position) float64 {
	if len(c.unknownPool) == 0 {
		return 0
	}
	camp := c.game.Board.IsCamp(to)
	sum := 0.0
	for _, t := range c.unknownPool {
		switch ResolveCombat(mover, &Piece{Type: t}, camp) {
		case 1:
			sum += valueOf(t)
		case -1:
			sum -= valueOf(mover.Type)
		case 0:
			sum += valueOf(t) - valueOf(mover.Type)
		}
	}
	return sum / float64(len(c.unknownPool)) * (1.0 + c.params.aggression)
}

func (c *aiContext) expectedLoss(to Position, mover *Piece) float64 {
	loss := 0.0
	for _, th := range c.threats {
		if !th.dests[to] {
			continue
		}
		if c.knows(th.piece) {
			switch ResolveCombat(th.piece, mover, c.game.Board.IsCamp(to)) {
			case 1:
				if v := valueOf(mover.Type); v > loss {
					loss = v
				}
			case 0:
				if v := valueOf(mover.Type) - valueOf(th.piece.Type)*0.5; v > loss {
					loss = v
				}
			}
			continue
		}
		if len(c.unknownPool) == 0 {
			continue
		}
		exp := 0.0
		for _, t := range c.unknownPool {
			switch ResolveCombat(&Piece{Type: t}, mover, c.game.Board.IsCamp(to)) {
			case 1:
				exp += valueOf(mover.Type)
			case 0:
				if d := valueOf(mover.Type) - valueOf(t)*0.5; d > 0 {
					exp += d
				}
			}
		}
		if v := exp / float64(len(c.unknownPool)); v > loss {
			loss = v
		}
	}
	return loss
}

func (c *aiContext) defenseScore(mover *Piece, m Move) float64 {
	if !c.flagThreatened {
		return 0
	}
	s := 0.0
	before, okBefore := c.distToOwnFlag[m.From]
	after, okAfter := c.distToOwnFlag[m.To]
	if okBefore && okAfter && after < before {
		s += c.params.defense * float64(before-after) * 3
	}
	return s
}

func (c *aiContext) coopScore(mover, target *Piece, m Move) float64 {
	if c.params.coop <= 0 || target == nil {
		return 0
	}
	if target.Seat.Team() == c.seat.Team() {
		return 0
	}
	dOwn, okOwn := c.distToOwnFlag[m.To]
	dMate, okMate := c.distToTeammateFlag[m.To]
	if okOwn && okMate && dMate < dOwn {
		// Removing an enemy that presses the teammate is a cooperative move.
		return c.params.coop * 14
	}
	return 0
}

// PromptForAI renders the current position for an LLM. Only information the
// acting seat may legitimately reason about is included: its own pieces, the
// board occupancy, and its legal moves.
func (g *Game) PromptForAI(seat Seat) (string, []Move) {
	moves := g.AllLegalMoves(seat)
	var b strings.Builder
	b.WriteString("你在四国军棋中执第")
	b.WriteString(strconv.Itoa(int(seat) + 1))
	b.WriteString("号位，队友是第")
	b.WriteString(strconv.Itoa(int((seat+2)%4) + 1))
	b.WriteString("号位。请从下列合法走法中选一步，优先吃掉或逼迫敌方棋子、保护己方军旗、配合队友形成攻势。\n")
	b.WriteString("己方棋子（类型@坐标x,y）：\n")
	for _, pc := range g.Pieces[seat] {
		if pc.Dead || pc.Position == (Position{-1, -1}) {
			continue
		}
		fmt.Fprintf(&b, "%s@%d,%d ", pc.GetName(), pc.Position.X, pc.Position.Y)
	}
	b.WriteString("\n棋盘上可见的棋子位置（含未翻开的他方棋子）：\n")
	for _, pos := range g.Board.Nodes() {
		pc := g.Board.GetPiece(pos)
		if pc == nil {
			continue
		}
		label := "未知"
		if pc.Seat == seat || pc.Known[seat] || pc.Known[Seat((int(seat)+2)%4)] {
			label = pc.GetName()
		}
		fmt.Fprintf(&b, "[%d,%d]%s(第%d号位) ", pos.X, pos.Y, label, int(pc.Seat)+1)
	}
	b.WriteString("\n合法走法（编号: 从x,y 到x,y）：\n")
	for i, m := range moves {
		fmt.Fprintf(&b, "%d: %d,%d -> %d,%d\n", i+1, m.From.X, m.From.Y, m.To.X, m.To.Y)
	}
	b.WriteString("只回答一个编号，例如：7")
	return b.String(), moves
}

// ParseAIMove extracts a move selection from an LLM answer. It accepts a bare
// index, "编号: N", or a coordinate pair such as "3,4 -> 5,6".
func ParseAIMove(answer string, moves []Move) (Move, bool) {
	if len(moves) == 0 {
		return Move{}, false
	}
	answer = strings.TrimSpace(answer)
	if idx, err := strconv.Atoi(answer); err == nil && idx >= 1 && idx <= len(moves) {
		return moves[idx-1], true
	}
	nums := extractInts(answer)
	// A single number such as "编号：2" is an index into the move list.
	if len(nums) == 1 && nums[0] >= 1 && nums[0] <= len(moves) {
		return moves[nums[0]-1], true
	}
	// Otherwise look for "从x,y 到x,y" style coordinates.
	if len(nums) >= 4 {
		from := Position{X: nums[len(nums)-4], Y: nums[len(nums)-3]}
		to := Position{X: nums[len(nums)-2], Y: nums[len(nums)-1]}
		for _, m := range moves {
			if m.From == from && m.To == to {
				return m, true
			}
		}
	}
	return Move{}, false
}

func extractInts(s string) []int {
	nums := make([]int, 0, 4)
	i := 0
	for i < len(s) {
		if s[i] >= '0' && s[i] <= '9' {
			j := i
			for j < len(s) && s[j] >= '0' && s[j] <= '9' {
				j++
			}
			if n, err := strconv.Atoi(s[i:j]); err == nil {
				nums = append(nums, n)
			}
			i = j
			continue
		}
		i++
	}
	return nums
}
