package game

import (
	"math/rand"
	"testing"
)

func TestBoardTopologyContainsImageLayout(t *testing.T) {
	b := NewBoard()
	if got := len(b.Nodes()); got != 129 {
		t.Fatalf("passable nodes = %d, want 129", got)
	}
	camps := 0
	for _, p := range []Position{{7, 2}, {9, 2}, {8, 3}, {7, 4}, {9, 4}, {12, 7}, {14, 7}, {13, 8}, {12, 9}, {14, 9}, {7, 12}, {9, 12}, {8, 13}, {7, 14}, {9, 14}, {2, 7}, {4, 7}, {3, 8}, {2, 9}, {4, 9}} {
		if !b.IsCamp(p) {
			t.Fatalf("camp %v is missing", p)
		}
		camps++
	}
	if camps != 20 {
		t.Fatal("want twenty camps")
	}
	for y := 6; y <= 10; y++ {
		for x := 6; x <= 10; x++ {
			want := (x == 6 || x == 8 || x == 10) && (y == 6 || y == 8 || y == 10)
			if b.IsValidPosition(Position{x, y}) != want {
				t.Fatalf("central node (%d,%d) validity is wrong", x, y)
			}
		}
	}
}

func TestAutomaticSetupObeysEverySeatRule(t *testing.T) {
	g := NewGame()
	for seat := Seat1; seat <= Seat4; seat++ {
		for _, piece := range g.Pieces[seat] {
			if piece.Dead {
				t.Fatalf("%s has a dead piece during setup", seat.Name())
			}
			if g.Board.Zone(piece.Position) != seat {
				t.Fatalf("%s piece %s placed outside its home area at %v", seat.Name(), piece.GetName(), piece.Position)
			}
			if g.Board.IsCamp(piece.Position) {
				t.Fatalf("%s piece %s placed in camp at %v", seat.Name(), piece.GetName(), piece.Position)
			}
			if piece.Type == Flag && !g.Board.IsHeadquarters(piece.Position) {
				t.Fatalf("%s flag is not in headquarters", seat.Name())
			}
			if piece.Type == Landmine && !g.isLastTwoRows(seat, piece.Position) {
				t.Fatalf("%s landmine is outside the last two rows", seat.Name())
			}
			if piece.Type == Bomb && g.isFirstRow(seat, piece.Position) {
				t.Fatalf("%s bomb is in the first row", seat.Name())
			}
		}
		if len(g.Pieces[seat]) != 25 {
			t.Fatalf("%s has %d pieces, want 25", seat.Name(), len(g.Pieces[seat]))
		}
	}
}

func TestCombatRules(t *testing.T) {
	if ResolveCombat(&Piece{Type: Engineer}, &Piece{Type: Landmine}, false) != 1 {
		t.Fatal("engineer must clear landmine")
	}
	if ResolveCombat(&Piece{Type: Marshal}, &Piece{Type: Landmine}, false) != -1 {
		t.Fatal("非工兵触雷应阵亡、地雷保留")
	}
	if ResolveCombat(&Piece{Type: Bomb}, &Piece{Type: Landmine}, false) != 0 {
		t.Fatal("炸弹触雷应双方同归于尽")
	}
	if ResolveCombat(&Piece{Type: Bomb}, &Piece{Type: Marshal}, false) != 0 {
		t.Fatal("bomb must destroy both pieces")
	}
	if ResolveCombat(&Piece{Type: Marshal}, &Piece{Type: Flag}, false) != 1 {
		t.Fatal("flag must be capturable")
	}
	if ResolveCombat(&Piece{Type: Marshal}, &Piece{Type: Engineer}, true) != 2 {
		t.Fatal("camp defender must be protected")
	}
}

func TestVisibilityProjectionHidesOtherSeats(t *testing.T) {
	g := NewGame()
	own := &Piece{Type: Marshal, Seat: Seat1}
	enemy := &Piece{Type: Flag, Seat: Seat2}
	g.Board.SetPiece(Position{6, 0}, own)
	g.Board.SetPiece(Position{16, 8}, enemy)
	seen := g.VisiblePieces(Seat1)
	hidden := 0
	visible := 0
	for _, p := range seen {
		if p.Hidden {
			hidden++
		} else {
			visible++
		}
	}
	if hidden != 75 || visible != 25 {
		t.Fatalf("unexpected view: hidden=%d visible=%d", hidden, visible)
	}
}

func TestOppositeSeatsDetermineWinner(t *testing.T) {
	g := NewGame()
	for _, s := range []Seat{Seat1, Seat3} {
		for _, p := range g.Pieces[s] {
			if p.Type == Flag {
				p.Dead = true
			}
		}
	}
	g.checkEliminations()
	if !g.GameOver || g.Winner != TeamBeta {
		t.Fatal("team beta must win when seats 1 and 3 lose their flags")
	}
}

func TestTurnOrderCyclesCounterClockwise(t *testing.T) {
	g := NewGame()
	g.Turn = Seat3
	g.PassTurn()
	if g.Turn != Seat4 {
		t.Fatalf("after seat 3, got %s", g.Turn.Name())
	}
	g.PassTurn()
	if g.Turn != Seat1 {
		t.Fatalf("after seat 4, got %s", g.Turn.Name())
	}
	g.PassTurn()
	if g.Turn != Seat2 {
		t.Fatalf("after seat 1, got %s", g.Turn.Name())
	}
	g.PassTurn()
	if g.Turn != Seat3 {
		t.Fatalf("after seat 2, got %s", g.Turn.Name())
	}
}

func TestOrdinaryPieceSlidesStraightAndStopsAtRightAngle(t *testing.T) {
	g := &Game{Board: NewBoard(), Turn: Seat3, Phase: PlayingPhase}
	from := Position{6, 15}
	g.Board.SetPiece(from, &Piece{Type: Marshal, Seat: Seat3})
	destinations := g.LegalDestinations(from)
	if !containsPosition(destinations, Position{6, 11}) {
		t.Fatal("ordinary piece must move multiple unobstructed railway steps")
	}

	g = &Game{Board: NewBoard(), Turn: Seat3, Phase: PlayingPhase}
	from = Position{8, 11}
	g.Board.SetPiece(from, &Piece{Type: Marshal, Seat: Seat3})
	destinations = g.LegalDestinations(from)
	if containsPosition(destinations, Position{10, 10}) {
		t.Fatal("ordinary piece must not make a right-angle railway turn")
	}
}

func TestRailwayBlockerStopsLongMove(t *testing.T) {
	g := &Game{Board: NewBoard(), Turn: Seat3, Phase: PlayingPhase}
	from := Position{6, 15}
	g.Board.SetPiece(from, &Piece{Type: Marshal, Seat: Seat3})
	g.Board.SetPiece(Position{6, 13}, &Piece{Type: Captain, Seat: Seat3})
	destinations := g.LegalDestinations(from)
	if containsPosition(destinations, Position{6, 12}) || containsPosition(destinations, Position{6, 11}) {
		t.Fatal("piece must not pass through a railway blocker")
	}
}

func TestEngineerCanTurnOnRailwayAndUseRoad(t *testing.T) {
	g := &Game{Board: NewBoard(), Turn: Seat3, Phase: PlayingPhase}
	from := Position{8, 11}
	g.Board.SetPiece(from, &Piece{Type: Engineer, Seat: Seat3})
	destinations := g.LegalDestinations(from)
	if !containsPosition(destinations, Position{10, 10}) {
		t.Fatal("engineer must be able to turn in the central railway network")
	}
}

func TestPieceCanEnterCampDiagonally(t *testing.T) {
	// 从“公路”节点 (13,8) 可斜向进入相邻行营 (12,7)。
	g := &Game{Board: NewBoard(), Turn: Seat3, Phase: PlayingPhase}
	from, camp := Position{13, 8}, Position{12, 7}
	g.Board.SetPiece(from, &Piece{Type: Captain, Seat: Seat3})
	if !containsPosition(g.LegalDestinations(from), camp) {
		t.Fatal("公路上的棋子应能斜向进入相邻行营")
	}
}

func TestRailCannotStepOntoOrdinaryRoad(t *testing.T) {
	// 铁路不能直达普通公路：(11,8) 在铁路 x=11 上，本回合只能沿铁路滑行，
	// 不能一步走上相邻普通公路 (12,8)（最外一排除外，见下条用例）。
	g := &Game{Board: NewBoard(), Turn: Seat1, Phase: PlayingPhase}
	g.Board.SetPiece(Position{11, 8}, &Piece{Type: Marshal, Seat: Seat1})
	dest := g.LegalDestinations(Position{11, 8})
	if containsPosition(dest, Position{12, 8}) {
		t.Fatal("铁路上的棋子不能一步跨到相邻普通公路 (12,8)")
	}
	if !containsPosition(dest, Position{11, 7}) || !containsPosition(dest, Position{11, 9}) {
		t.Fatal("铁路上的棋子应能沿铁路滑行")
	}
}

func TestBackRowIsRoadNotRailway(t *testing.T) {
	// 每翼最外一排/列（大本营所在排）整体为公路；铁路矩形内缩一排，
	// 其端点仍为铁路。逐一覆盖四方，避免后续修改破坏这条规则。
	b := NewBoard()
	arms := []struct {
		name  string
		outer []Position
		inner []Position
	}{
		{"上方", []Position{{6, 0}, {7, 0}, {8, 0}, {9, 0}, {10, 0}}, []Position{{6, 1}, {10, 1}}},
		{"右方", []Position{{16, 6}, {16, 7}, {16, 8}, {16, 9}, {16, 10}}, []Position{{15, 6}, {15, 10}}},
		{"下方", []Position{{6, 16}, {7, 16}, {8, 16}, {9, 16}, {10, 16}}, []Position{{6, 15}, {10, 15}}},
		{"左方", []Position{{0, 6}, {0, 7}, {0, 8}, {0, 9}, {0, 10}}, []Position{{1, 6}, {1, 10}}},
	}
	for _, arm := range arms {
		for _, p := range arm.outer {
			if b.IsRailway(p) {
				t.Fatalf("%s翼最外一排的点 %v 应为公路，不应挨着铁路", arm.name, p)
			}
		}
		for _, p := range arm.inner {
			if !b.IsRailway(p) {
				t.Fatalf("%s翼铁路端点 %v 应为铁路", arm.name, p)
			}
		}
	}
}

func TestNoRailwayAdjacentToHeadquarters(t *testing.T) {
	// 大本营位于最外一排（公路），其所有相邻边都是公路（不是铁路边）。
	b := NewBoard()
	for _, hq := range []Position{{9, 0}, {16, 9}, {7, 16}, {0, 7}} {
		edges := b.Edges(hq)
		if len(edges) == 0 {
			t.Fatalf("大本营 %v 应有相邻点", hq)
		}
		for _, e := range edges {
			if e.Route != Road {
				t.Fatalf("大本营 %v 的相邻边 %v 应为公路", hq, e.To)
			}
		}
	}
}

func TestRailCanReachOutermostLine(t *testing.T) {
	// 各翼最外一排（大本营所在排）是公路，但它只与铁路相邻、是大本营的唯一
	// 出口；因此允许铁路上的棋子一步踏上该排（否则任何棋子都无法夺旗）。
	// 普通棋子与工兵都应能踏上该排。
	b := NewBoard()
	arms := []struct {
		name string
		from Position
		out  Position
	}{
		{"上方", Position{6, 1}, Position{6, 0}},
		{"右方", Position{15, 6}, Position{16, 6}},
		{"下方", Position{6, 15}, Position{6, 16}},
		{"左方", Position{1, 6}, Position{0, 6}},
	}
	for _, arm := range arms {
		if !b.IsRailway(arm.from) {
			t.Fatalf("%s翼起点 %v 应为铁路", arm.name, arm.from)
		}
		if !b.IsLastLine(arm.out) {
			t.Fatalf("%s翼落点 %v 应属于最外一排", arm.name, arm.out)
		}
		for _, kind := range []PieceType{Marshal, Engineer} {
			g := &Game{Board: NewBoard(), Turn: Seat1, Phase: PlayingPhase}
			g.Board.SetPiece(arm.from, &Piece{Type: kind, Seat: Seat1})
			if !containsPosition(g.LegalDestinations(arm.from), arm.out) {
				t.Fatalf("%s翼：铁路上的%s应能一步到达最外一排 %v", arm.name, (&Piece{Type: kind}).GetName(), arm.out)
			}
		}
	}
}

func TestHeadquartersReachableAndFlagCapturable(t *testing.T) {
	// 大本营必须可达：铁路上的棋子应能一步冲进敌方大本营并夺旗，夺旗后该
	// 棋子驻留大本营且不可再移动，被夺旗席位立即出局。
	hq := Position{16, 9}
	g := &Game{Board: NewBoard(), Turn: Seat1, Phase: PlayingPhase}
	marshal := &Piece{Type: Marshal, Seat: Seat1}
	flag := &Piece{Type: Flag, Seat: Seat2}
	g.Pieces[Seat1] = []*Piece{marshal}
	g.Pieces[Seat2] = []*Piece{flag}
	g.Pieces[Seat3] = []*Piece{{Type: Flag, Seat: Seat3, Position: g.Board.FlagAnchor(Seat3)}}
	g.Pieces[Seat4] = []*Piece{{Type: Flag, Seat: Seat4, Position: g.Board.FlagAnchor(Seat4)}}
	g.Board.SetPiece(Position{15, 9}, marshal) // 铁路一侧
	g.Board.SetPiece(hq, flag)
	if !containsPosition(g.LegalDestinations(Position{15, 9}), hq) {
		t.Fatal("铁路上的棋子应能一步冲进大本营夺旗")
	}
	if !g.Move(Position{15, 9}, hq) {
		t.Fatal("夺旗走法应合法")
	}
	if !g.Eliminated[Seat2] {
		t.Fatal("2 号位军旗被夺后应立即出局")
	}
	if !marshal.Immobile || marshal.Movable() {
		t.Fatal("占领军旗的棋子（驻留大本营）应不可再移动")
	}
}

func TestFlagCannotMove(t *testing.T) {
	// 军旗不可移动：无论位于大本营，都无法产生任何合法落点。
	for s := Seat1; s <= Seat4; s++ {
		g := &Game{Board: NewBoard(), Turn: s, Phase: PlayingPhase}
		flag := &Piece{Type: Flag, Seat: s, Position: g.Board.FlagAnchor(s)}
		g.Pieces[s] = []*Piece{flag}
		g.Board.SetPiece(g.Board.FlagAnchor(s), flag)
		if dest := g.LegalDestinations(g.Board.FlagAnchor(s)); len(dest) != 0 {
			t.Fatalf("%s 的军旗不应有任何合法落点，实际 %v", s.Name(), dest)
		}
		if g.CanMove(g.Board.FlagAnchor(s), Position{g.Board.FlagAnchor(s).X + 1, g.Board.FlagAnchor(s).Y}) {
			t.Fatalf("%s 的军旗不应能移动", s.Name())
		}
	}
}

func TestRailPieceCanStepIntoAdjacentCamp(t *testing.T) {
	// 铁路上的棋子可以一步进入相邻行营（ISS-025 曾误将行营一并封锁，
	// 导致铁路侧的行营无法进入）。普通棋子与工兵都应能进入。
	for _, kind := range []PieceType{Marshal, Captain, Engineer} {
		g := &Game{Board: NewBoard(), Turn: Seat1, Phase: PlayingPhase}
		g.Board.SetPiece(Position{10, 12}, &Piece{Type: kind, Seat: Seat1})
		if !containsPosition(g.LegalDestinations(Position{10, 12}), Position{9, 12}) {
			t.Fatalf("%s 在铁路上应能一步进入相邻行营 (9,12)", (&Piece{Type: kind}).GetName())
		}

		g2 := &Game{Board: NewBoard(), Turn: Seat1, Phase: PlayingPhase}
		g2.Board.SetPiece(Position{11, 8}, &Piece{Type: kind, Seat: Seat1})
		d := g2.LegalDestinations(Position{11, 8})
		for _, camp := range []Position{{12, 7}, {12, 9}} {
			if !containsPosition(d, camp) {
				t.Fatalf("%s 在铁路上应能一步进入相邻行营 %v", (&Piece{Type: kind}).GetName(), camp)
			}
		}
	}
}

func TestRailwayTurnOnlyAtCurves(t *testing.T) {
	// 弯道（非直角）可以不停下、直接转向；直角路口必须停下。
	// 用户例子 1：(9,11) 本回合只能沿直线到 (10,11)（(10,11) 是直角口）。
	g := &Game{Board: NewBoard(), Turn: Seat1, Phase: PlayingPhase}
	g.Board.SetPiece(Position{9, 11}, &Piece{Type: Marshal, Seat: Seat1})
	dest := g.LegalDestinations(Position{9, 11})
	if !containsPosition(dest, Position{10, 11}) {
		t.Fatal("(9,11) 应能沿直线到 (10,11)")
	}
	for _, forbidden := range []Position{{10, 10}, {11, 10}, {11, 9}} {
		if containsPosition(dest, forbidden) {
			t.Fatalf("直角口 (10,11) 应停下，不应到达 %v", forbidden)
		}
	}

	// (10,11) 直行到弯道 (10,10) 后可拐到 (11,10)，但 (11,10) 是直角口要停。
	g2 := &Game{Board: NewBoard(), Turn: Seat1, Phase: PlayingPhase}
	g2.Board.SetPiece(Position{10, 11}, &Piece{Type: Marshal, Seat: Seat1})
	d2 := g2.LegalDestinations(Position{10, 11})
	if !containsPosition(d2, Position{10, 10}) || !containsPosition(d2, Position{11, 10}) {
		t.Fatal("(10,11) 应能到 (10,10) 并在弯道拐到 (11,10)")
	}
	if containsPosition(d2, Position{11, 9}) {
		t.Fatal("(11,10) 是直角口应停下")
	}

	// 用户例子 2：(10,13) 经弯道 (10,10) 可一步直达 (13,10)。
	g3 := &Game{Board: NewBoard(), Turn: Seat1, Phase: PlayingPhase}
	g3.Board.SetPiece(Position{10, 13}, &Piece{Type: Marshal, Seat: Seat1})
	if !containsPosition(g3.LegalDestinations(Position{10, 13}), Position{13, 10}) {
		t.Fatal("(10,13) 应能经弯道一步直达 (13,10)")
	}

	// 弯道 (6,6) 连接上方翼与左方翼：从上方向下到 (6,6) 应能拐入左翼 (5,6)。
	g4 := &Game{Board: NewBoard(), Turn: Seat1, Phase: PlayingPhase}
	g4.Board.SetPiece(Position{6, 1}, &Piece{Type: Marshal, Seat: Seat1})
	if !containsPosition(g4.LegalDestinations(Position{6, 1}), Position{5, 6}) {
		t.Fatal("自上向下到弯道 (6,6) 应能拐入左翼 (5,6)")
	}
}

func TestBombCannotCutAcrossCenterCorner(t *testing.T) {
	// 炸弹 (14,6) 沿右翼 y=6 直行到弯道 (10,6)；该弯道只连接上方翼与右方翼，
	// 不能直插中央（否则会像工兵一样横穿棋盘去 (10,13) 与司令同归于尽）。
	g := &Game{Board: NewBoard(), Turn: Seat1, Phase: PlayingPhase}
	g.Board.SetPiece(Position{14, 6}, &Piece{Type: Bomb, Seat: Seat1})
	dest := g.LegalDestinations(Position{14, 6})
	if containsPosition(dest, Position{10, 13}) {
		t.Fatal("炸弹不应经弯道直插中央到达 (10,13)")
	}
	if containsPosition(dest, Position{10, 8}) {
		t.Fatal("弯道 (10,6) 不应允许拐入中央 (10,8)")
	}
	// 但应能沿弯道拐入上方翼（(10,5) 方向）。
	if !containsPosition(dest, Position{10, 5}) {
		t.Fatal("弯道 (10,6) 应允许拐入上方翼 (10,5)")
	}
}

func TestEngineerReachesWholeRailwayNetwork(t *testing.T) {
	g := &Game{Board: NewBoard(), Turn: Seat3, Phase: PlayingPhase}
	from := Position{6, 15}
	g.Board.SetPiece(from, &Piece{Type: Engineer, Seat: Seat3})
	dest := g.LegalDestinations(from)
	// 铁路网络内缩一排：最外一排为公路，(6,1)/(1,6)/(15,6) 是各翼铁路的端点。
	for _, want := range []Position{{6, 1}, {10, 5}, {1, 6}, {15, 6}, {8, 8}, {10, 15}} {
		if !containsPosition(dest, want) {
			t.Fatalf("工兵应能沿铁轨到达 %v", want)
		}
	}
}

func TestTeammatesCannotCaptureEachOther(t *testing.T) {
	g := &Game{Board: NewBoard(), Turn: Seat1, Phase: PlayingPhase}
	mine := &Piece{Type: Marshal, Seat: Seat1, Position: Position{6, 1}}
	ally := &Piece{Type: Captain, Seat: Seat3, Position: Position{7, 1}}
	enemy := &Piece{Type: Lieutenant, Seat: Seat2, Position: Position{6, 2}}
	g.Pieces[Seat1] = []*Piece{mine}
	g.Pieces[Seat3] = []*Piece{ally}
	g.Pieces[Seat2] = []*Piece{enemy}
	g.Board.SetPiece(Position{6, 1}, mine)
	g.Board.SetPiece(Position{7, 1}, ally)
	g.Board.SetPiece(Position{6, 2}, enemy)

	dest := g.LegalDestinations(Position{6, 1})
	if containsPosition(dest, Position{7, 1}) {
		t.Fatal("对家（队友）的棋子不应可被吃")
	}
	if !containsPosition(dest, Position{6, 2}) {
		t.Fatal("紧邻的对手棋子应可被吃")
	}
	if g.CanMove(Position{6, 1}, Position{7, 1}) {
		t.Fatal("CanMove 不应允许移动到队友棋子所在点")
	}
}

func TestPieceThatCapturesFlagBecomesImmobile(t *testing.T) {
	g := &Game{Board: NewBoard(), Turn: Seat1, Phase: PlayingPhase}
	marshal := &Piece{Type: Marshal, Seat: Seat1, Position: Position{6, 1}}
	flag := &Piece{Type: Flag, Seat: Seat2, Position: Position{6, 2}}
	g.Pieces[Seat1] = []*Piece{marshal, {Type: Flag, Seat: Seat1, Position: Position{-1, -1}}}
	g.Pieces[Seat2] = []*Piece{flag}
	g.Pieces[Seat3] = []*Piece{{Type: Flag, Seat: Seat3, Position: Position{-1, -1}}}
	g.Pieces[Seat4] = []*Piece{{Type: Flag, Seat: Seat4, Position: Position{-1, -1}}}
	g.Board.SetPiece(Position{6, 1}, marshal)
	g.Board.SetPiece(Position{6, 2}, flag)

	if !g.Move(Position{6, 1}, Position{6, 2}) {
		t.Fatal("应能吃军旗")
	}
	if !marshal.Immobile {
		t.Fatal("占领军旗的棋子应被标记为不可移动")
	}
	if marshal.Dead {
		t.Fatal("占领军旗的棋子不应死亡")
	}
	g.GameOver, g.Phase, g.Turn = false, PlayingPhase, Seat1
	if len(g.LegalDestinations(Position{6, 2})) != 0 {
		t.Fatal("占领军旗后的棋子不能再移动")
	}
	if g.CanMove(Position{6, 2}, Position{6, 3}) {
		t.Fatal("CanMove 不应允许移动占领军旗后的棋子")
	}
}

func TestNonEngineerDiesAndLandmineRemains(t *testing.T) {
	g := &Game{Board: NewBoard(), Turn: Seat1, Phase: PlayingPhase}
	marshal := &Piece{Type: Marshal, Seat: Seat1, Position: Position{6, 1}}
	mine := &Piece{Type: Landmine, Seat: Seat2, Position: Position{6, 2}}
	g.Pieces[Seat1] = []*Piece{marshal, {Type: Flag, Seat: Seat1, Position: Position{-1, -1}}}
	g.Pieces[Seat2] = []*Piece{mine, {Type: Flag, Seat: Seat2, Position: Position{-1, -1}}}
	g.Pieces[Seat3] = []*Piece{{Type: Flag, Seat: Seat3, Position: Position{-1, -1}}}
	g.Pieces[Seat4] = []*Piece{{Type: Flag, Seat: Seat4, Position: Position{-1, -1}}}
	g.Board.SetPiece(Position{6, 1}, marshal)
	g.Board.SetPiece(Position{6, 2}, mine)

	if !g.Move(Position{6, 1}, Position{6, 2}) {
		t.Fatal("元帅应能移动到地雷位置")
	}
	if !marshal.Dead || g.Board.GetPiece(Position{6, 1}) != nil {
		t.Fatal("触雷的非工兵棋子应消失")
	}
	if mine.Dead || g.Board.GetPiece(Position{6, 2}) != mine {
		t.Fatal("地雷应保留在原地")
	}
}

func TestBombAndLandmineDestroyEachOther(t *testing.T) {
	g := &Game{Board: NewBoard(), Turn: Seat1, Phase: PlayingPhase}
	bomb := &Piece{Type: Bomb, Seat: Seat1, Position: Position{6, 1}}
	mine := &Piece{Type: Landmine, Seat: Seat2, Position: Position{6, 2}}
	g.Pieces[Seat1] = []*Piece{bomb, {Type: Flag, Seat: Seat1, Position: Position{-1, -1}}}
	g.Pieces[Seat2] = []*Piece{mine, {Type: Flag, Seat: Seat2, Position: Position{-1, -1}}}
	g.Pieces[Seat3] = []*Piece{{Type: Flag, Seat: Seat3, Position: Position{-1, -1}}}
	g.Pieces[Seat4] = []*Piece{{Type: Flag, Seat: Seat4, Position: Position{-1, -1}}}
	g.Board.SetPiece(Position{6, 1}, bomb)
	g.Board.SetPiece(Position{6, 2}, mine)

	if !g.Move(Position{6, 1}, Position{6, 2}) {
		t.Fatal("炸弹应能移动到地雷位置")
	}
	if !bomb.Dead || !mine.Dead {
		t.Fatal("炸弹与地雷应同归于尽")
	}
	if g.Board.GetPiece(Position{6, 1}) != nil || g.Board.GetPiece(Position{6, 2}) != nil {
		t.Fatal("炸弹与地雷都应从棋盘移除")
	}
}

func TestFlagRevealedWhenMarshalDies(t *testing.T) {
	g := &Game{Board: NewBoard(), Turn: Seat1, Phase: PlayingPhase}
	marshal := &Piece{Type: Marshal, Seat: Seat1, Position: Position{6, 1}}
	bomb := &Piece{Type: Bomb, Seat: Seat2, Position: Position{6, 2}}
	ownFlag := &Piece{Type: Flag, Seat: Seat1, Position: g.Board.FlagAnchor(Seat1)}
	g.Pieces[Seat1] = []*Piece{marshal, ownFlag}
	g.Pieces[Seat2] = []*Piece{bomb, {Type: Flag, Seat: Seat2, Position: g.Board.FlagAnchor(Seat2)}}
	g.Pieces[Seat3] = []*Piece{{Type: Flag, Seat: Seat3, Position: g.Board.FlagAnchor(Seat3)}}
	g.Pieces[Seat4] = []*Piece{{Type: Flag, Seat: Seat4, Position: g.Board.FlagAnchor(Seat4)}}
	g.Board.SetPiece(Position{6, 1}, marshal)
	g.Board.SetPiece(Position{6, 2}, bomb)
	g.Board.SetPiece(g.Board.FlagAnchor(Seat1), ownFlag)
	g.Board.SetPiece(g.Board.FlagAnchor(Seat2), g.Pieces[Seat2][1])
	g.Board.SetPiece(g.Board.FlagAnchor(Seat3), g.Pieces[Seat3][0])
	g.Board.SetPiece(g.Board.FlagAnchor(Seat4), g.Pieces[Seat4][0])

	if !g.Move(Position{6, 1}, Position{6, 2}) {
		t.Fatal("司令应能攻击炸弹")
	}
	if !marshal.Dead {
		t.Fatal("司令应与炸弹同归于尽")
	}
	if !g.MarshalDead(Seat1) {
		t.Fatal("司令阵亡后 MarshalDead 应为真")
	}
	if !ownFlag.Known[Seat2] {
		t.Fatal("司令阵亡后军旗应对外公开")
	}
	revealed := false
	for _, v := range g.VisiblePieces(Seat2) {
		if v.Position == g.Board.FlagAnchor(Seat1) {
			revealed = !v.Hidden && v.Type == Flag
		}
	}
	if !revealed {
		t.Fatal("司令阵亡后，1 号位军旗位置应对 2 号位可见")
	}
}

func TestAllAILevelsReturnLegalMove(t *testing.T) {
	for _, level := range []AILevel{AIEasy, AIMedium, AIHard, AILLM} {
		g := NewGame()
		g.Turn = Seat1
		m, ok := g.ChooseMove(level, rand.New(rand.NewSource(int64(level)+1)))
		if !ok {
			t.Fatalf("%v 未返回任何走法", level)
		}
		if !g.CanMove(m.From, m.To) {
			t.Fatalf("%v 返回了非法走法 %v", level, m)
		}
	}
}

func TestAIHardCapturesFlagOnHeadquarters(t *testing.T) {
	// Fair information: the AI cannot see the enemy flag, but it can deduce
	// that the only piece that may occupy a headquarters is the flag.
	g := &Game{Board: NewBoard(), Turn: Seat1, Phase: PlayingPhase}
	enemyFlagPos := g.Board.FlagAnchor(Seat2)
	marshal := &Piece{Type: Marshal, Seat: Seat1, Position: Position{16, 8}}
	flag := &Piece{Type: Flag, Seat: Seat2, Position: enemyFlagPos}
	g.Pieces[Seat1] = []*Piece{marshal, {Type: Flag, Seat: Seat1, Position: Position{-1, -1}}}
	g.Pieces[Seat2] = []*Piece{flag}
	g.Pieces[Seat3] = []*Piece{{Type: Flag, Seat: Seat3, Position: Position{-1, -1}}}
	g.Pieces[Seat4] = []*Piece{{Type: Flag, Seat: Seat4, Position: Position{-1, -1}}}
	g.Board.SetPiece(Position{16, 8}, marshal)
	g.Board.SetPiece(enemyFlagPos, flag)

	m, ok := g.ChooseMove(AIHard, rand.New(rand.NewSource(1)))
	if !ok || m.To != enemyFlagPos {
		t.Fatalf("高级 AI 应吃下大本营上的敌方军旗，实际为 %v ok=%v", m, ok)
	}
}

func TestHeadquartersOnlyAcceptsFlag(t *testing.T) {
	g := &Game{Board: NewBoard(), Phase: SetupPhase}
	anchor := g.Board.FlagAnchor(Seat1)
	if err := g.SetupPiece(Seat1, &Piece{Type: Marshal, Seat: Seat1}, anchor); err == nil {
		t.Fatal("大本营不应允许放置非军旗棋子")
	}
	if err := g.SetupPiece(Seat1, &Piece{Type: Flag, Seat: Seat1}, anchor); err != nil {
		t.Fatalf("军旗应可放置在大本营: %v", err)
	}
}

func TestCannotEnterEmptyHeadquarters(t *testing.T) {
	g := &Game{Board: NewBoard(), Turn: Seat1, Phase: PlayingPhase}
	g.Board.SetPiece(Position{9, 1}, &Piece{Type: Marshal, Seat: Seat1})
	if containsPosition(g.LegalDestinations(Position{9, 1}), Position{9, 0}) {
		t.Fatal("不可走入空的大本营")
	}
}

func TestAdjudicateUsesRemainingMaterial(t *testing.T) {
	g := NewGame()
	// Cripple team beta so team alpha is clearly ahead on material.
	for _, s := range []Seat{Seat2, Seat4} {
		for _, p := range g.Pieces[s] {
			if p.Type == Flag || p.Dead {
				continue
			}
			if valueOf(p.Type) >= valueOf(Brigadier) {
				p.Dead = true
				if g.Board.GetPiece(p.Position) == p {
					g.Board.SetPiece(p.Position, nil)
				}
				p.Position = Position{-1, -1}
			}
		}
	}
	g.Adjudicate()
	if !g.GameOver {
		t.Fatal("判和/判胜后应结束对局")
	}
	if g.Draw || g.Winner != TeamAlpha {
		t.Fatalf("子力占优的 α 队应获胜, winner=%v draw=%v", g.Winner, g.Draw)
	}
}

func TestCombatRevealsTypesToParticipantsOnly(t *testing.T) {
	g := &Game{Board: NewBoard(), Turn: Seat1, Phase: PlayingPhase}
	a := &Piece{Type: Marshal, Seat: Seat1, Position: Position{6, 1}}
	d := &Piece{Type: Captain, Seat: Seat2, Position: Position{6, 2}}
	g.Pieces[Seat1] = []*Piece{a, {Type: Flag, Seat: Seat1, Position: Position{-1, -1}}}
	g.Pieces[Seat2] = []*Piece{d, {Type: Flag, Seat: Seat2, Position: Position{-1, -1}}}
	g.Pieces[Seat3] = []*Piece{{Type: Flag, Seat: Seat3, Position: Position{-1, -1}}}
	g.Pieces[Seat4] = []*Piece{{Type: Flag, Seat: Seat4, Position: Position{-1, -1}}}
	g.Board.SetPiece(Position{6, 1}, a)
	g.Board.SetPiece(Position{6, 2}, d)

	if !g.Move(Position{6, 1}, Position{6, 2}) {
		t.Fatal("移动失败")
	}
	if !a.Known[Seat2] {
		t.Fatal("防守方应得知攻击方棋子类型")
	}
	if !d.Known[Seat1] {
		t.Fatal("攻击方应得知防守方棋子类型")
	}
	if a.Known[Seat4] || d.Known[Seat3] {
		t.Fatal("非参与方不应得知棋子类型")
	}
}

func TestAIHardAdvancesTowardEnemyFlag(t *testing.T) {
	g := &Game{Board: NewBoard(), Turn: Seat1, Phase: PlayingPhase}
	marshal := &Piece{Type: Marshal, Seat: Seat1, Position: Position{8, 0}}
	g.Pieces[Seat1] = []*Piece{marshal}
	g.Board.SetPiece(Position{8, 0}, marshal)

	field := g.Board.DistanceField(g.Board.HeadquartersOf(Seat3))
	before := field[Position{8, 0}]
	m, ok := g.ChooseMove(AIHard, rand.New(rand.NewSource(2)))
	if !ok {
		t.Fatal("高级 AI 未返回走法")
	}
	after := field[m.To]
	if after >= before {
		t.Fatalf("高级 AI 应向敌方推进：before=%d after=%d move=%v", before, after, m)
	}
}

func TestParseAIMove(t *testing.T) {
	moves := []Move{{From: Position{1, 1}, To: Position{1, 2}}, {From: Position{3, 4}, To: Position{5, 6}}}
	if m, ok := ParseAIMove("2", moves); !ok || m.To != (Position{5, 6}) {
		t.Fatalf("应解析编号 2")
	}
	if m, ok := ParseAIMove("从3,4到5,6", moves); !ok || m.To != (Position{5, 6}) {
		t.Fatalf("应解析坐标")
	}
	if _, ok := ParseAIMove("无法判断", moves); ok {
		t.Fatal("无效回答不应解析出移动")
	}
}

func containsPosition(positions []Position, wanted Position) bool {
	for _, position := range positions {
		if position == wanted {
			return true
		}
	}
	return false
}
