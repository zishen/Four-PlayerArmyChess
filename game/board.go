package game

type Position struct{ X, Y int }

type RouteType int

const (
	Road RouteType = iota
	Railway
)

type Edge struct {
	From, To Position
	Route    RouteType
}

// Board is the immutable topology plus mutable occupancy of the image-based
// four-country board. The visible board is a 17x17 cross, not a 9x9 square.
type Board struct {
	Width, Height int
	Grid          [][]*Piece
	passable      map[Position]bool
	zones         map[Position]Seat
	adj           map[Position][]Edge
	camps         map[Position]bool
	headquarters  map[Position]bool
}

func NewBoard() *Board {
	b := &Board{
		Width: 17, Height: 17, Grid: make([][]*Piece, 17),
		passable: map[Position]bool{}, zones: map[Position]Seat{},
		adj: map[Position][]Edge{}, camps: map[Position]bool{}, headquarters: map[Position]bool{},
	}
	for y := range b.Grid {
		b.Grid[y] = make([]*Piece, 17)
	}
	for y := 0; y < 17; y++ {
		for x := 0; x < 17; x++ {
			if (x >= 6 && x <= 10 && y <= 5) ||
				(x >= 11 && y >= 6 && y <= 10) ||
				(x >= 6 && x <= 10 && y >= 11) ||
				(x <= 5 && y >= 6 && y <= 10) ||
				((x == 6 || x == 8 || x == 10) && (y == 6 || y == 8 || y == 10)) {
				b.passable[Position{x, y}] = true
			}
		}
	}
	// Five circular camps visible in each 6x5 home arm.
	for _, p := range []Position{
		{7, 2}, {9, 2}, {8, 3}, {7, 4}, {9, 4}, // top
		{12, 7}, {14, 7}, {13, 8}, {12, 9}, {14, 9}, // right
		{7, 12}, {9, 12}, {8, 13}, {7, 14}, {9, 14}, // bottom
		{2, 7}, {4, 7}, {3, 8}, {2, 9}, {4, 9}, // left
	} {
		b.camps[p] = true
	}
	// Each arm has exactly one headquarters, reserved for the flag. No other
	// piece may be placed on or move onto it.
	for _, p := range []Position{{9, 0}, {16, 9}, {7, 16}, {0, 7}} {
		b.headquarters[p] = true
	}
	for _, p := range b.armNodes(Seat1) {
		b.zones[p] = Seat1
	}
	for _, p := range b.armNodes(Seat2) {
		b.zones[p] = Seat2
	}
	for _, p := range b.armNodes(Seat3) {
		b.zones[p] = Seat3
	}
	for _, p := range b.armNodes(Seat4) {
		b.zones[p] = Seat4
	}
	for p := range b.passable {
		for _, d := range []Position{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {2, 0}, {-2, 0}, {0, 2}, {0, -2}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
			q := Position{p.X + d.X, p.Y + d.Y}
			if !b.passable[q] || (p.X != q.X && p.Y != q.Y && !b.diagonalRoad(p, q)) {
				continue
			}
			if p.X == q.X && abs(p.Y-q.Y) == 2 || p.Y == q.Y && abs(p.X-q.X) == 2 {
				if !b.isCentral(p) || !b.isCentral(q) {
					continue
				}
			}
			route := Road
			if b.isRailwayEdge(p, q) {
				route = Railway
			}
			b.adj[p] = append(b.adj[p], Edge{From: p, To: q, Route: route})
		}
	}
	// The four 弯道 (rounded railway corners) sit at the central 3x3 corners,
	// where two adjacent arms meet. An ordinary piece may change direction
	// only at these nodes (a non-right-angle curve); at any other railway node
	// it must continue straight (a sharp 90 degree turn is not allowed).
	// Engineers are unrestricted.
	return b
}

// IsCurve reports whether p is one of the four rounded railway corners where
// an ordinary piece is allowed to change direction.
func (b *Board) IsCurve(p Position) bool {
	switch p {
	case (Position{6, 6}), (Position{6, 10}), (Position{10, 6}), (Position{10, 10}):
		return true
	}
	return false
}

// CurveTurnAllowed reports whether a piece entering node in direction (dx,dy)
// may leave it in direction (nx,ny). Only the four rounded corners turn, and
// only along the physical curve that joins the two adjacent arms — a piece may
// not cut across the corner into the centre. Travelling straight is handled
// separately by the caller.
func (b *Board) CurveTurnAllowed(node Position, dx, dy, nx, ny int) bool {
	switch node {
	case (Position{6, 6}): // top arm <-> left arm
		return (dx == 0 && dy == 1 && nx == -1 && ny == 0) || (dx == 1 && dy == 0 && nx == 0 && ny == -1)
	case (Position{10, 6}): // top arm <-> right arm
		return (dx == 0 && dy == 1 && nx == 1 && ny == 0) || (dx == -1 && dy == 0 && nx == 0 && ny == -1)
	case (Position{6, 10}): // bottom arm <-> left arm
		return (dx == 0 && dy == -1 && nx == -1 && ny == 0) || (dx == 1 && dy == 0 && nx == 0 && ny == 1)
	case (Position{10, 10}): // bottom arm <-> right arm
		return (dx == 0 && dy == -1 && nx == 1 && ny == 0) || (dx == -1 && dy == 0 && nx == 0 && ny == 1)
	}
	return false
}

func (b *Board) armNodes(s Seat) []Position {
	result := make([]Position, 0, 30)
	for y := 0; y < 17; y++ {
		for x := 0; x < 17; x++ {
			p := Position{x, y}
			if b.passable[p] && b.armOf(p) == s {
				result = append(result, p)
			}
		}
	}
	return result
}

func (b *Board) armOf(p Position) Seat {
	switch {
	case p.Y < 6:
		return Seat1
	case p.X > 10:
		return Seat2
	case p.Y > 10:
		return Seat3
	case p.X < 6:
		return Seat4
	default:
		return Seat(-1)
	}
}

func (b *Board) diagonalRoad(a, c Position) bool {
	if b.armOf(a) != b.armOf(c) || b.armOf(a) < Seat1 {
		return false
	}
	return b.camps[a] || b.camps[c] || b.camps[Position{a.X, c.Y}] || b.camps[Position{c.X, a.Y}] || b.camps[Position{(a.X + c.X) / 2, (a.Y + c.Y) / 2}]
}

func (b *Board) isCentral(p Position) bool {
	return (p.X == 6 || p.X == 8 || p.X == 10) && (p.Y == 6 || p.Y == 8 || p.Y == 10)
}

func (b *Board) isRailwayEdge(a, c Position) bool {
	if b.isCentral(a) && b.isCentral(c) {
		return true
	}
	if b.isCentral(a) || b.isCentral(c) {
		return true
	}
	// 每翼的铁路由两条纵向线、两条横向线围成一个矩形，且相对最外一排整体
	// 内缩一排：最外一排（大本营所在排）是纯公路，铁路不与它直接相连。
	// 因此任何以最外一排/列为端点的边都只能是公路。
	switch b.armOf(a) {
	case Seat1:
		if a.Y == 0 || c.Y == 0 {
			return false
		}
		return a.X == c.X && (a.X == 6 || a.X == 10) || a.Y == c.Y && (a.Y == 1 || a.Y == 5)
	case Seat2:
		if a.X == 16 || c.X == 16 {
			return false
		}
		return a.Y == c.Y && (a.Y == 6 || a.Y == 10) || a.X == c.X && (a.X == 11 || a.X == 15)
	case Seat3:
		if a.Y == 16 || c.Y == 16 {
			return false
		}
		return a.X == c.X && (a.X == 6 || a.X == 10) || a.Y == c.Y && (a.Y == 11 || a.Y == 15)
	case Seat4:
		if a.X == 0 || c.X == 0 {
			return false
		}
		return a.Y == c.Y && (a.Y == 6 || a.Y == 10) || a.X == c.X && (a.X == 1 || a.X == 5)
	default:
		return false
	}
}

func (b *Board) IsValidPosition(p Position) bool {
	return p.X >= 0 && p.X < 17 && p.Y >= 0 && p.Y < 17 && b.passable[p]
}
func (b *Board) GetPiece(p Position) *Piece {
	if !b.IsValidPosition(p) {
		return nil
	}
	return b.Grid[p.Y][p.X]
}
func (b *Board) SetPiece(p Position, piece *Piece) {
	if b.IsValidPosition(p) {
		b.Grid[p.Y][p.X] = piece
		if piece != nil {
			piece.Position = p
		}
	}
}
func (b *Board) MovePiece(from, to Position) bool {
	p := b.GetPiece(from)
	if p == nil || !b.IsValidPosition(to) || b.GetPiece(to) != nil {
		return false
	}
	b.SetPiece(from, nil)
	b.SetPiece(to, p)
	return true
}
func (b *Board) IsCamp(p Position) bool         { return b.camps[p] }
func (b *Board) IsHeadquarters(p Position) bool { return b.headquarters[p] }

// IsLastLine reports whether p lies on its arm's outermost line — the row or
// column that holds the headquarters (大本营: 上 y=0 / 右 x=16 / 下 y=16 / 左 x=0).
// The headquarters can only be reached from the railway across this line, so a
// piece on the railway is allowed to step onto it (unlike other ordinary roads);
// otherwise no piece could ever capture the flag.
func (b *Board) IsLastLine(p Position) bool {
	switch b.armOf(p) {
	case Seat1:
		return p.Y == 0
	case Seat2:
		return p.X == 16
	case Seat3:
		return p.Y == 16
	case Seat4:
		return p.X == 0
	}
	return false
}
func (b *Board) IsRailway(p Position) bool {
	for _, edge := range b.adj[p] {
		if edge.Route == Railway {
			return true
		}
	}
	return false
}
func (b *Board) Zone(p Position) Seat {
	seat, ok := b.zones[p]
	if !ok {
		return Seat(-1)
	}
	return seat
}
func (b *Board) Edges(p Position) []Edge { return b.adj[p] }
func (b *Board) Nodes() []Position {
	result := make([]Position, 0, len(b.passable))
	for p := range b.passable {
		result = append(result, p)
	}
	return result
}

// FlagAnchor returns the single headquarters of a seat, the only legal
// position for its flag.
func (b *Board) FlagAnchor(s Seat) Position {
	return []Position{{9, 0}, {16, 9}, {7, 16}, {0, 7}}[s]
}

// HeadquartersOf returns the headquarters of a seat (one per arm).
func (b *Board) HeadquartersOf(s Seat) []Position {
	return []Position{b.FlagAnchor(s)}
}

// DistanceField returns the number of graph steps from every passable node to
// the nearest source. Nodes that cannot be reached are absent from the map.
func (b *Board) DistanceField(sources []Position) map[Position]int {
	dist := map[Position]int{}
	queue := make([]Position, 0, len(sources))
	for _, s := range sources {
		if b.passable[s] {
			dist[s] = 0
			queue = append(queue, s)
		}
	}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		for _, e := range b.adj[p] {
			if _, ok := dist[e.To]; ok {
				continue
			}
			dist[e.To] = dist[p] + 1
			queue = append(queue, e.To)
		}
	}
	return dist
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
