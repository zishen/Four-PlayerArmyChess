package game

type PieceType int

const (
	Flag PieceType = iota
	Marshal
	General
	MajorGeneral
	Brigadier
	Colonel
	Major
	Captain
	Lieutenant
	Engineer
	Landmine
	Bomb
)

type Seat int

const (
	Seat1 Seat = iota
	Seat2
	Seat3
	Seat4
)

type Team int

const (
	TeamAlpha Team = iota // Seat1 + Seat3
	TeamBeta              // Seat2 + Seat4
)

type Piece struct {
	ID       int
	Type     PieceType
	Seat     Seat
	Revealed bool
	Dead     bool
	// Immobile is set on a piece that has captured a flag. Such a piece stays
	// on the flag square and can no longer move.
	Immobile bool
	// Known records, per seat, whether that seat has learned this piece's type
	// through an interaction. It supports the fair-information AI.
	Known    [4]bool
	Position Position
}

func (p *Piece) GetRank() int {
	switch p.Type {
	case Marshal:
		return 9
	case General:
		return 8
	case MajorGeneral:
		return 7
	case Brigadier:
		return 6
	case Colonel:
		return 5
	case Major:
		return 4
	case Captain:
		return 3
	case Lieutenant:
		return 2
	case Engineer:
		return 1
	default:
		return 0
	}
}

func (p *Piece) GetName() string {
	names := map[PieceType]string{
		Flag: "军旗", Marshal: "司令", General: "军长", MajorGeneral: "师长",
		Brigadier: "旅长", Colonel: "团长", Major: "营长", Captain: "连长",
		Lieutenant: "排长", Engineer: "工兵", Landmine: "地雷", Bomb: "炸弹",
	}
	return names[p.Type]
}

func (s Seat) Team() Team {
	if s == Seat1 || s == Seat3 {
		return TeamAlpha
	}
	return TeamBeta
}

func (s Seat) Name() string { return []string{"1号位", "2号位", "3号位", "4号位"}[s] }

func nextSeat(s Seat) Seat { return Seat((int(s) + 1) % 4) }

func movable(t PieceType) bool { return t != Flag && t != Landmine }

// Movable reports whether the piece may be moved at all: flags and landmines
// never move, and a piece that has captured a flag stays frozen on the flag
// square (大本营) and cannot move again.
func (p *Piece) Movable() bool { return movable(p.Type) && !p.Immobile }

func pieceLayout() []PieceType {
	result := make([]PieceType, 0, 25)
	for _, item := range []struct {
		t PieceType
		n int
	}{
		{Flag, 1}, {Marshal, 1}, {General, 1}, {MajorGeneral, 2}, {Brigadier, 2},
		{Colonel, 2}, {Major, 2}, {Captain, 3}, {Lieutenant, 3}, {Engineer, 3},
		{Landmine, 3}, {Bomb, 2},
	} {
		for i := 0; i < item.n; i++ {
			result = append(result, item.t)
		}
	}
	return result
}
