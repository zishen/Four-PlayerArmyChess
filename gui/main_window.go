package gui

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"Four-PlayerArmyChess/game"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

const (
	boardSourceWidth  = 1054.0
	boardSourceHeight = 994.0
	firstNodeX        = 81.0
	firstNodeY        = 48.0
	nodeStepX         = (988.0 - firstNodeX) / 16.0
	nodeStepY         = (952.0 - firstNodeY) / 16.0
	aiMoveDelay       = time.Second
	replayDelay       = time.Second
	recordExt         = ".4gj"
)

type gameWindow struct {
	mainWindow *walk.MainWindow
	boardView  *walk.CustomWidget
	status     *walk.Label
	moveList   *walk.ListBox

	game         *game.Game
	viewer       game.Seat
	selected     game.Position
	boardImage   walk.Image
	imageRect    walk.Rectangle
	running      bool
	busy         bool
	effectPos    game.Position
	effectFrom   game.Position
	combatEffect bool

	level        game.AILevel
	rng          *rand.Rand
	aiCfg        aiConfig
	levelActions []*walk.Action
	passStreak   int

	replay       *game.Replay
	replayOn     bool
	playing      bool
	updatingList bool
	autoSaved    bool
	lastSaved    string

	font           *walk.Font
	fontSize       int
	seatBrushes    [4]walk.Brush
	highlightBrush walk.Brush
	destPen        walk.Pen
	effectPen      walk.Pen
	combatPen      walk.Pen
	bgBrush        walk.Brush
}

func Run() {
	image, err := walk.NewImageFromFile(boardImagePath())
	if err != nil {
		panic(err)
	}
	defer image.Dispose()

	w := &gameWindow{
		game:       game.NewGame(),
		viewer:     game.Seat3,
		selected:   game.Position{X: -1, Y: -1},
		effectPos:  game.Position{X: -1, Y: -1},
		effectFrom: game.Position{X: -1, Y: -1},
		boardImage: image,
		level:      game.AIMedium,
		rng:        rand.New(rand.NewSource(time.Now().UnixNano())),
		aiCfg:      loadAIConfig(),
	}
	var mw *walk.MainWindow
	var board *walk.CustomWidget
	var status *walk.Label
	var moveList *walk.ListBox
	var actEasy, actMedium, actHard, actLLM *walk.Action
	err = MainWindow{
		AssignTo: &mw, Title: "四国军棋", MinSize: Size{Width: 1000, Height: 780}, Layout: VBox{MarginsZero: true},
		MenuItems: []MenuItem{
			Menu{Text: "游戏", Items: []MenuItem{
				Action{Text: "开始游戏", OnTriggered: func() { w.startGame() }},
				Action{Text: "结束游戏", OnTriggered: func() { w.endGame() }},
				Separator{},
				Action{Text: "退出", OnTriggered: func() { mw.Close() }},
			}},
			Menu{Text: "难度", Items: []MenuItem{
				Action{AssignTo: &actEasy, Text: "初级", Checkable: true, Checked: w.level == game.AIEasy, OnTriggered: func() { w.setLevel(game.AIEasy) }},
				Action{AssignTo: &actMedium, Text: "中级", Checkable: true, Checked: w.level == game.AIMedium, OnTriggered: func() { w.setLevel(game.AIMedium) }},
				Action{AssignTo: &actHard, Text: "高级", Checkable: true, Checked: w.level == game.AIHard, OnTriggered: func() { w.setLevel(game.AIHard) }},
				Action{AssignTo: &actLLM, Text: "大模型", Checkable: true, Checked: w.level == game.AILLM, OnTriggered: func() { w.setLevel(game.AILLM) }},
			}},
			Menu{Text: "复盘", Items: []MenuItem{
				Action{Text: "打开复盘…", OnTriggered: func() { w.openReplay() }},
				Action{Text: "保存对局…", OnTriggered: func() { w.saveGame() }},
				Separator{},
				Action{Text: "上一步", OnTriggered: func() { w.replayStep(-1) }},
				Action{Text: "下一步", OnTriggered: func() { w.replayStep(1) }},
				Action{Text: "退出复盘", OnTriggered: func() { w.exitReplay() }},
			}},
			Menu{Text: "设置", Items: []MenuItem{
				Action{Text: "大模型设置…", OnTriggered: func() {
					showAISettingsDialog(mw, &w.aiCfg)
					w.updateStatus()
				}},
			}},
		},
		Children: []Widget{
			Composite{Layout: HBox{Margins: Margins{Left: 10, Top: 8, Right: 10, Bottom: 6}}, Children: []Widget{
				PushButton{Text: "开始游戏", OnClicked: func() { w.startGame() }},
				PushButton{Text: "结束游戏", OnClicked: func() { w.endGame() }},
				PushButton{Text: "保存对局", OnClicked: func() { w.saveGame() }},
				PushButton{Text: "打开复盘", OnClicked: func() { w.openReplay() }},
				HSpacer{},
				Label{AssignTo: &status, Text: "点击开始游戏"},
			}},
			Composite{Layout: HBox{Margins: Margins{Left: 10, Top: 0, Right: 10, Bottom: 8}}, Children: []Widget{
				CustomWidget{AssignTo: &board, MinSize: Size{Width: 620, Height: 600}, StretchFactor: 1, DoubleBuffering: true, PaintMode: PaintBuffered, InvalidatesOnResize: true, Paint: func(c *walk.Canvas, bounds walk.Rectangle) error { return w.draw(c, bounds) }},
				Composite{MinSize: Size{Width: 250}, Layout: VBox{}, Children: []Widget{
					Label{Text: "走子记录（点击可跳转复盘）"},
					ListBox{AssignTo: &moveList, MinSize: Size{Width: 240, Height: 470}, OnCurrentIndexChanged: func() { w.onMoveSelected() }},
					Composite{Layout: HBox{}, Children: []Widget{
						PushButton{Text: "◀ 上一步", OnClicked: func() { w.replayStep(-1) }},
						PushButton{Text: "下一步 ▶", OnClicked: func() { w.replayStep(1) }},
					}},
					Composite{Layout: HBox{}, Children: []Widget{
						PushButton{Text: "自动播放", OnClicked: func() { w.togglePlay() }},
						PushButton{Text: "退出复盘", OnClicked: func() { w.exitReplay() }},
					}},
				}},
			}},
		},
	}.Create()
	if err != nil {
		panic(err)
	}
	w.mainWindow, w.boardView, w.status, w.moveList = mw, board, status, moveList
	w.levelActions = []*walk.Action{actEasy, actMedium, actHard, actLLM}
	w.setLevel(w.level)
	board.MouseDown().Attach(func(x, y int, _ walk.MouseButton) { w.click(x, y) })
	w.refreshMoveLog()
	w.updateStatus()
	mw.Run()
}

func boardImagePath() string {
	executable, err := os.Executable()
	if err == nil {
		path := filepath.Join(filepath.Dir(executable), "ico", "01-棋盘.jpeg")
		if _, statErr := os.Stat(path); statErr == nil {
			return path
		}
	}
	return filepath.Join("ico", "01-棋盘.jpeg")
}

func (w *gameWindow) visibleGame() *game.Game {
	if w.replayOn && w.replay != nil {
		return w.replay.Game()
	}
	return w.game
}

func (w *gameWindow) setLevel(level game.AILevel) {
	w.level = level
	for i, action := range w.levelActions {
		if action != nil {
			action.SetChecked(game.AILevel(i) == level)
		}
	}
	if w.status != nil {
		w.updateStatus()
	}
}

func (w *gameWindow) startGame() {
	w.stopReplay()
	w.game = game.NewGame()
	w.game.Turn = w.viewer
	w.selected = game.Position{X: -1, Y: -1}
	w.effectPos, w.effectFrom = game.Position{X: -1, Y: -1}, game.Position{X: -1, Y: -1}
	w.running = true
	w.busy = false
	w.passStreak = 0
	w.autoSaved = false
	w.lastSaved = ""
	w.refreshMoveLog()
	w.updateStatus()
	w.boardView.Invalidate()
}

func (w *gameWindow) endGame() {
	if !w.running {
		return
	}
	w.game.End()
	w.running = false
	w.busy = false
	w.selected = game.Position{X: -1, Y: -1}
	w.refreshMoveLog()
	w.updateStatus()
	w.boardView.Invalidate()
}

func (w *gameWindow) updateStatus() {
	switch {
	case w.replayOn && w.replay != nil:
		result := ""
		rec := w.replay.Record()
		if rec.Draw {
			result = "  结果：和局"
		} else if rec.Winner != "" {
			result = "  结果：" + rec.Winner + "获胜"
		}
		w.status.SetText(fmt.Sprintf("复盘：第 %d / %d 步%s", w.replay.Step(), w.replay.Total(), result))
	case w.game.GameOver && w.game.Draw:
		w.status.SetText("和局（双方均无子可动）")
	case w.game.GameOver:
		winner := "α队(1、3号位)"
		if w.game.Winner == game.TeamBeta {
			winner = "β队(2、4号位)"
		}
		suffix := ""
		if w.lastSaved != "" {
			suffix = "  已保存：" + filepath.Base(w.lastSaved)
		}
		w.status.SetText("对局结束，" + winner + "获胜" + suffix)
	case !w.running:
		w.status.SetText("点击开始游戏")
	case w.game.Turn == w.viewer:
		w.status.SetText("轮到您走棋（下方绿色）｜难度：" + w.level.String())
	default:
		w.status.SetText("当前回合：" + w.game.Turn.Name() + "｜难度：" + w.level.String())
	}
	w.autoSaveIfOver()
}

func (w *gameWindow) autoSaveIfOver() {
	if !w.game.GameOver || w.replayOn || w.autoSaved || len(w.game.History) == 0 {
		return
	}
	w.autoSaved = true
	path := filepath.Join(filepath.Dir(aiConfigPath()), "对局_"+time.Now().Format("20060102_150405")+recordExt)
	if err := game.SaveRecord(path, w.game.Record()); err == nil {
		w.lastSaved = path
	}
}

func (w *gameWindow) draw(c *walk.Canvas, bounds walk.Rectangle) error {
	// PaintFunc 的 bounds 只是本次无效区域（WM_PAINT 的 RcPaint），局部重绘时可能
	// 只是一小块；始终以控件完整客户区计算棋盘矩形，避免局部重绘把棋盘缩进小块里。
	if w.boardView != nil {
		if cb := w.boardView.ClientBounds(); cb.Width > 0 && cb.Height > 0 {
			bounds = cb
		}
	}
	if err := w.ensureResources(); err != nil {
		return err
	}
	// lxn/walk 的 PaintBuffered 不会自动清除缓冲区位图（其内容未定义），绘制函数
	// 必须自己填充背景；否则棋盘图片按比例缩放后四周的留白会残留未初始化内容，
	// 表现为界面出现异常/重复区域。
	c.FillRectangle(w.bgBrush, bounds)
	w.imageRect = fitImage(bounds, boardSourceWidth/boardSourceHeight)
	if err := c.DrawImageStretchedPixels(w.boardImage, w.imageRect); err != nil {
		return err
	}
	g := w.visibleGame()
	if !w.running && !g.GameOver && !w.replayOn {
		return nil
	}
	for _, piece := range g.VisiblePieces(w.viewer) {
		center := w.nodePoint(piece.Position)
		pieceWidth := maxInt(20, int(math.Round(44*float64(w.imageRect.Width)/boardSourceWidth)))
		pieceHeight := maxInt(16, int(math.Round(32*float64(w.imageRect.Height)/boardSourceHeight)))
		r := walk.Rectangle{X: center.X - pieceWidth/2, Y: center.Y - pieceHeight/2, Width: pieceWidth, Height: pieceHeight}
		if piece.Position == w.selected {
			c.FillRectangle(w.highlightBrush, walk.Rectangle{X: r.X - 3, Y: r.Y - 3, Width: r.Width + 6, Height: r.Height + 6})
		}
		c.FillRectangle(w.seatBrushes[piece.Seat], r)
		text := "?"
		if !piece.Hidden {
			text = (&game.Piece{Type: piece.Type}).GetName()
		}
		c.DrawText(text, w.font, walk.RGB(20, 20, 20), r, walk.TextCenter|walk.TextVCenter)
	}
	if w.selected.X >= 0 {
		for _, destination := range g.LegalDestinations(w.selected) {
			center := w.nodePoint(destination)
			c.DrawEllipse(w.destPen, walk.Rectangle{X: center.X - 9, Y: center.Y - 9, Width: 18, Height: 18})
		}
	}
	if w.effectPos.X >= 0 {
		center := w.nodePoint(w.effectPos)
		size := maxInt(24, int(math.Round(50*float64(w.imageRect.Width)/boardSourceWidth)))
		pen := w.effectPen
		if w.combatEffect {
			pen = w.combatPen
		}
		if w.effectFrom.X >= 0 {
			c.DrawLine(pen, w.nodePoint(w.effectFrom), center)
		}
		c.DrawEllipse(pen, walk.Rectangle{X: center.X - size/2, Y: center.Y - size/2, Width: size, Height: size})
	}
	return nil
}

// ensureResources creates the fonts, brushes and pens used by draw once and
// keeps them for every later repaint.
func (w *gameWindow) ensureResources() error {
	if w.highlightBrush == nil {
		for i := 0; i < 4; i++ {
			brush, err := walk.NewSolidColorBrush(seatColor(game.Seat(i)))
			if err != nil {
				return err
			}
			w.seatBrushes[i] = brush
		}
		var err error
		if w.highlightBrush, err = walk.NewSolidColorBrush(walk.RGB(255, 230, 80)); err != nil {
			return err
		}
		if w.destPen, err = walk.NewCosmeticPen(walk.PenDash, walk.RGB(245, 215, 45)); err != nil {
			return err
		}
		if w.effectPen, err = walk.NewCosmeticPen(walk.PenSolid, walk.RGB(255, 220, 40)); err != nil {
			return err
		}
		if w.combatPen, err = walk.NewCosmeticPen(walk.PenSolid, walk.RGB(220, 55, 45)); err != nil {
			return err
		}
		if w.bgBrush, err = walk.NewSolidColorBrush(walk.RGB(255, 255, 255)); err != nil {
			return err
		}
	}
	fontSize := int(math.Max(7, float64(w.imageRect.Width)/90))
	if w.font == nil || w.fontSize != fontSize {
		if w.font != nil {
			w.font.Dispose()
			w.font = nil
		}
		font, err := walk.NewFont("Microsoft YaHei", fontSize, walk.FontBold)
		if err != nil {
			return err
		}
		w.font = font
		w.fontSize = fontSize
	}
	return nil
}

func fitImage(bounds walk.Rectangle, aspect float64) walk.Rectangle {
	width, height := bounds.Width, int(math.Round(float64(bounds.Width)/aspect))
	if height > bounds.Height {
		height = bounds.Height
		width = int(math.Round(float64(height) * aspect))
	}
	return walk.Rectangle{X: bounds.X + (bounds.Width-width)/2, Y: bounds.Y + (bounds.Height-height)/2, Width: width, Height: height}
}

func (w *gameWindow) nodePoint(p game.Position) walk.Point {
	sx := float64(w.imageRect.Width) / boardSourceWidth
	sy := float64(w.imageRect.Height) / boardSourceHeight
	return walk.Point{X: w.imageRect.X + int(math.Round((firstNodeX+float64(p.X)*nodeStepX)*sx)), Y: w.imageRect.Y + int(math.Round((firstNodeY+float64(p.Y)*nodeStepY)*sy))}
}

func (w *gameWindow) pointToNode(x, y int) (game.Position, bool) {
	sx := float64(w.imageRect.Width) / boardSourceWidth
	sy := float64(w.imageRect.Height) / boardSourceHeight
	fx := (float64(x-w.imageRect.X)/sx - firstNodeX) / nodeStepX
	fy := (float64(y-w.imageRect.Y)/sy - firstNodeY) / nodeStepY
	p := game.Position{X: int(math.Round(fx)), Y: int(math.Round(fy))}
	if math.Abs(fx-float64(p.X)) > .42 || math.Abs(fy-float64(p.Y)) > .42 || !w.visibleGame().Board.IsValidPosition(p) {
		return game.Position{}, false
	}
	return p, true
}

func seatColor(s game.Seat) walk.Color {
	return []walk.Color{walk.RGB(215, 154, 154), walk.RGB(149, 190, 232), walk.RGB(155, 211, 163), walk.RGB(240, 178, 105)}[s]
}

func (w *gameWindow) click(x, y int) {
	if w.replayOn || !w.running || w.busy || w.game.GameOver || w.game.Turn != w.viewer {
		return
	}
	p, ok := w.pointToNode(x, y)
	if !ok {
		return
	}
	if w.selected.X < 0 {
		piece := w.game.Board.GetPiece(p)
		if piece != nil && piece.Seat == w.viewer && piece.Movable() {
			w.selected = p
			w.boardView.Invalidate()
		}
		return
	}
	if p == w.selected {
		w.selected = game.Position{X: -1, Y: -1}
		w.boardView.Invalidate()
		return
	}
	combat := w.game.Board.GetPiece(p) != nil
	from := w.selected
	if w.game.Move(from, p) {
		playActionSound(combat)
		w.selected = game.Position{X: -1, Y: -1}
		w.effectFrom, w.effectPos, w.combatEffect, w.busy = from, p, combat, true
		w.passStreak = 0
		w.refreshMoveLog()
		w.boardView.Invalidate()
		w.mainWindow.Synchronize(func() { w.clearEffectAndRunAI() })
	} else {
		piece := w.game.Board.GetPiece(p)
		if piece != nil && piece.Seat == w.viewer && piece.Movable() {
			w.selected = p
		} else {
			w.selected = game.Position{X: -1, Y: -1}
		}
		w.boardView.Invalidate()
	}
}

func (w *gameWindow) clearEffectAndRunAI() {
	if w.game.GameOver {
		w.effectPos, w.effectFrom = game.Position{X: -1, Y: -1}, game.Position{X: -1, Y: -1}
		w.busy = false
		w.updateStatus()
		w.boardView.Invalidate()
		return
	}
	time.AfterFunc(aiMoveDelay, func() {
		w.mainWindow.Synchronize(func() {
			w.effectPos, w.effectFrom = game.Position{X: -1, Y: -1}, game.Position{X: -1, Y: -1}
			w.boardView.Invalidate()
			w.runAITurn()
		})
	})
}

func (w *gameWindow) runAITurn() {
	if w.replayOn {
		return
	}
	if w.game.GameOver {
		w.busy = false
		w.updateStatus()
		w.boardView.Invalidate()
		return
	}
	if w.game.Turn == w.viewer {
		if !w.game.HasLegalMove(w.viewer) {
			w.passOrDraw()
			return
		}
		w.busy = false
		w.updateStatus()
		w.boardView.Invalidate()
		return
	}
	if w.level == game.AILLM && w.aiCfg.ready() {
		prompt, moves := w.game.PromptForAI(w.game.Turn)
		cfg := w.aiCfg
		w.busy = true
		go func() {
			answer, err := requestLLMMove(cfg, prompt)
			w.mainWindow.Synchronize(func() {
				if w.game.GameOver || w.replayOn {
					return
				}
				move, ok := game.Move{}, false
				if err == nil {
					move, ok = game.ParseAIMove(answer, moves)
				}
				if !ok {
					move, ok = w.game.ChooseMove(game.AIHard, w.rng)
				}
				w.commitAIMove(move, ok)
			})
		}()
		return
	}
	move, ok := w.game.ChooseMove(w.level, w.rng)
	w.commitAIMove(move, ok)
}

func (w *gameWindow) commitAIMove(move game.Move, ok bool) {
	if w.game.GameOver || w.replayOn {
		return
	}
	if !ok {
		w.passOrDraw()
		return
	}
	w.passStreak = 0
	combat := w.game.Board.GetPiece(move.To) != nil
	if w.game.Move(move.From, move.To) {
		playActionSound(combat)
		w.effectFrom, w.effectPos, w.combatEffect = move.From, move.To, combat
		w.refreshMoveLog()
		w.boardView.Invalidate()
	}
	time.AfterFunc(aiMoveDelay, func() {
		w.mainWindow.Synchronize(func() {
			w.effectPos, w.effectFrom = game.Position{X: -1, Y: -1}, game.Position{X: -1, Y: -1}
			w.boardView.Invalidate()
			w.runAITurn()
		})
	})
}

func (w *gameWindow) passOrDraw() {
	if w.game.GameOver {
		return
	}
	w.passStreak++
	if w.passStreak >= 4 {
		w.game.Adjudicate()
		w.busy = false
		w.updateStatus()
		w.boardView.Invalidate()
		return
	}
	w.game.PassTurn()
	if w.game.Turn == w.viewer && w.game.HasLegalMove(w.viewer) {
		w.busy = false
		w.updateStatus()
		w.boardView.Invalidate()
		return
	}
	w.mainWindow.Synchronize(func() { w.runAITurn() })
}

// --- move log ---

func (w *gameWindow) moveSource() []game.MoveRecord {
	if w.replayOn && w.replay != nil {
		return w.replay.Moves()
	}
	return w.game.History
}

func (w *gameWindow) currentStep() int {
	if w.replayOn && w.replay != nil {
		return w.replay.Step()
	}
	return len(w.game.History)
}

func (w *gameWindow) refreshMoveLog() {
	if w.moveList == nil {
		return
	}
	moves := w.moveSource()
	items := make([]string, len(moves))
	for i, m := range moves {
		items[i] = m.Label(i + 1)
	}
	w.updatingList = true
	_ = w.moveList.SetModel(items)
	_ = w.moveList.SetCurrentIndex(w.currentStep() - 1)
	w.updatingList = false
}

func (w *gameWindow) onMoveSelected() {
	if w.updatingList || !w.replayOn || w.replay == nil {
		return
	}
	idx := w.moveList.CurrentIndex()
	if idx < 0 {
		return
	}
	w.playing = false
	w.replay.Goto(idx + 1)
	w.selected = game.Position{X: -1, Y: -1}
	w.updateStatus()
	w.boardView.Invalidate()
}

// --- save / replay ---

func (w *gameWindow) saveGame() {
	var rec game.GameRecord
	if w.replayOn && w.replay != nil {
		rec = w.replay.Record()
	} else {
		rec = w.game.Record()
	}
	if len(rec.Initial) == 0 {
		walk.MsgBox(w.mainWindow, "保存对局", "当前没有可保存的对局。", walk.MsgBoxIconInformation)
		return
	}
	dlg := &walk.FileDialog{
		Title:    "保存对局",
		Filter:   "四国军棋记录 (*" + recordExt + ")|*" + recordExt + "|所有文件 (*.*)|*.*",
		FilePath: "对局_" + time.Now().Format("20060102_150405") + recordExt,
	}
	ok, err := dlg.ShowSave(w.mainWindow)
	if err != nil || !ok {
		return
	}
	path := dlg.FilePath
	if !strings.EqualFold(filepath.Ext(path), recordExt) {
		path += recordExt
	}
	if err := game.SaveRecord(path, rec); err != nil {
		walk.MsgBox(w.mainWindow, "保存失败", err.Error(), walk.MsgBoxIconError)
		return
	}
	w.lastSaved = path
	walk.MsgBox(w.mainWindow, "保存成功", "已保存到：\n"+path, walk.MsgBoxIconInformation)
}

func (w *gameWindow) openReplay() {
	dlg := &walk.FileDialog{
		Title:  "打开复盘",
		Filter: "四国军棋记录 (*" + recordExt + ")|*" + recordExt + "|所有文件 (*.*)|*.*",
	}
	ok, err := dlg.ShowOpen(w.mainWindow)
	if err != nil || !ok {
		return
	}
	rec, err := game.LoadRecord(dlg.FilePath)
	if err != nil {
		walk.MsgBox(w.mainWindow, "打开失败", err.Error(), walk.MsgBoxIconError)
		return
	}
	w.replay = game.NewReplay(rec)
	w.replayOn = true
	w.playing = false
	w.running = false
	w.busy = false
	w.selected = game.Position{X: -1, Y: -1}
	w.effectPos, w.effectFrom = game.Position{X: -1, Y: -1}, game.Position{X: -1, Y: -1}
	w.refreshMoveLog()
	w.updateStatus()
	w.boardView.Invalidate()
}

func (w *gameWindow) stopReplay() {
	w.replay = nil
	w.replayOn = false
	w.playing = false
}

func (w *gameWindow) exitReplay() {
	if !w.replayOn {
		return
	}
	w.stopReplay()
	w.selected = game.Position{X: -1, Y: -1}
	w.refreshMoveLog()
	w.updateStatus()
	w.boardView.Invalidate()
}

func (w *gameWindow) replayStep(delta int) {
	if !w.replayOn || w.replay == nil {
		return
	}
	w.playing = false
	if delta < 0 {
		w.replay.Prev()
	} else {
		w.replay.Next()
	}
	w.selected = game.Position{X: -1, Y: -1}
	w.refreshMoveLog()
	w.updateStatus()
	w.boardView.Invalidate()
}

func (w *gameWindow) togglePlay() {
	if !w.replayOn || w.replay == nil {
		return
	}
	w.playing = !w.playing
	if w.playing {
		w.autoStep()
	}
}

func (w *gameWindow) autoStep() {
	if !w.playing || !w.replayOn || w.replay == nil {
		return
	}
	if !w.replay.Next() {
		w.playing = false
		w.updateStatus()
		return
	}
	w.refreshMoveLog()
	w.updateStatus()
	w.boardView.Invalidate()
	time.AfterFunc(replayDelay, func() {
		w.mainWindow.Synchronize(func() { w.autoStep() })
	})
}

func playActionSound(combat bool) {
	beep := syscall.NewLazyDLL("kernel32.dll").NewProc("Beep")
	go func() {
		if combat {
			beep.Call(420, 90)
			beep.Call(260, 130)
			return
		}
		beep.Call(850, 45)
		beep.Call(1050, 55)
	}()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
