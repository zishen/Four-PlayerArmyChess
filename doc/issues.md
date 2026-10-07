# 项目问题清单

## 1. 开放问题

| ID | 优先级 | 状态 | 问题 | 证据与影响 | 下一步 | 验证 |
| --- | --- | --- | --- | --- | --- | --- |
| ISS-001 | P1 | 已确认 | 规则文档与实现的棋盘模型冲突 | `game/board.go` 和测试使用 17x17、129 节点、20 行营；`doc/rules.md` 描述 9x9、52 点、9 行营。无法判定产品正确性 | 规则负责人选择唯一基线，随后同步需求、规则、实现和测试 | 规则评审记录及拓扑测试通过 |
| ISS-002 | P2 | 已缓解 | Go 工具链版本曾导致本地验证阻塞 | `go version` 现为 `go1.26.5 windows/amd64`，工具链门禁已解除；全量验证仍受依赖网络影响 | 保持团队 Go 版本策略，并继续处理依赖缓存/网络问题 | 工具链满足版本要求，`go test ./game`、`go vet ./game` 通过 |
| ISS-008 | P1 | 已确认 | Walk 依赖无法下载，导致 GUI 包全量验证阻塞 | `go test ./...`、`go vet ./...` 和构建均无法获取 `github.com/lxn/walk@v0.0.0-20210112085537-c389da54e794` | 配置可访问的 Go module proxy，或预置并校验模块缓存后重试 | 全量测试、静态检查和 Windows 构建通过 |
| ISS-003 | P1 | 已修复 | AI 在所有席位均无合法移动时可能持续同步递归 | `runAITurn` 现通过 `passOrDraw` 记录连续跳过次数，达到 4 次即调用 `Game.EndDraw` 判和，不再无限递归 | 修复于本次迭代 | `TestAISelfPlayIsActiveAndTerminates` 与判和逻辑验证 |
| ISS-009 | P1 | 已修复 | 普通棋子铁路拐弯规则与 doc 不一致 | 先误改为“只能同向直行”（禁止圆弧）；按 `doc/rules.md`，普通棋子应可沿圆弧（非直角）拐弯、但不可在直角路口拐弯 | 恢复四角圆弧铁路段，普通棋子允许非直角转弯（点积 > 0）、直角拒绝；工兵 BFS 覆盖全部无阻挡铁路点 | `TestOrdinaryPieceFollowsRailwayCurveButNotRightAngle`、`TestEngineerReachesWholeRailwayNetwork` 通过 |
| ISS-010 | P2 | 已修复 | 缺少 AI 难度设置菜单 | 界面无难度入口 | 新增「难度」菜单（初级/中级/高级/大模型）| 菜单可切换并即时生效 |
| ISS-011 | P2 | 已修复 | AI 过于消极，几乎不进攻、不布防、不配合、不吃子 | 原 AI 只选择遍历顺序中的第一个合法移动 | 改为启发式 AI：有利吃子、向敌方大本营推进、护旗、配合队友 | 自对弈每局吃子 ≥ 20 |
| ISS-012 | P3 | 已修复 | AI 智能化低、无法接入大模型 | 无外部模型接入 | 新增 `ai_config.json`、设置对话框与 OpenAI 兼容客户端，失败回退高级 AI | 配置对话框与回退逻辑验证 |
| ISS-013 | P1 | 已修复 | 对家（队友）可被己方棋子吃掉 | `canOccupyDestination` 只排除同席位，强烈化后的 AI 会攻击己方对家 | 改为排除同队席位，棋子不可进入队友交叉点 | `TestTeammatesCannotCaptureEachOther` 通过 |
| ISS-014 | P2 | 已修复 | 每步重绘闪烁、卡顿 | 每次 Paint 都新建字体/画刷且未启用双缓冲 | 启用 `PaintBuffered` 双缓冲并缓存字体、画刷与画笔 | 运行平滑，无闪屏 |
| ISS-015 | P2 | 已修复 | 占领军旗的棋子仍可继续移动 | 吃军旗后未限制该棋子 | 新增 `Piece.Immobile`，吃旗即置位，`CanMove`/`LegalDestinations` 拒绝其移动 | `TestPieceThatCapturesFlagBecomesImmobile` 通过 |
| ISS-016 | P1 | 已修复 | 大本营模型与规则不符（每翼 2 个、可放非军旗、可走入） | `board.go` 曾标记 8 个大本营点，`setup.go`/移动均未限制 | 改为每翼 1 个大本营（`FlagAnchor`），仅放军旗且不可走入（吃旗例外） | 决策 C9；`TestHeadquartersOnlyAcceptsFlag`、`TestCannotEnterEmptyHeadquarters`、`TestAIHardCapturesFlagOnHeadquarters` 通过 |
| ISS-017 | P2 | 已修复 | 僵局无子力判定 | 之前仅 `EndDraw` 直接判和 | 新增 `Game.Adjudicate`：按剩余子力（价值总和→数量）判胜，平局和棋；GUI 僵局改调它 | 决策 C7；`TestAdjudicateUsesRemainingMaterial` 通过 |
| ISS-018 | P2 | 已修复 | AI 使用全盘信息，违反信息隔离 | `game/ai.go` 直接读取敌方棋子类型 | 改为公平信息：新增 `Piece.Known`，仅用己方+已交互揭示类型；未识别棋子按剩余类型分布估计战斗结果 | 决策 C8；`TestCombatRevealsTypesToParticipantsOnly` 通过 |
| ISS-019 | P1 | 已修复 | `rules.md` 与实现存在术语/几何/流程冲突 | 9×9 棋盘、席位方位、行棋方向、铁路步数、行营时机、大本营、僵局、AI 信息等 | 由规则负责人逐条裁定（C1–C11），`rules.md` 已重写并与实现对齐 | `doc/rules.md` 决策说明 + 本清单 |
| ISS-020 | P2 | 已修复 | 无走子记录、存档与复盘能力 | 之前无历史记录，无法保存/回放 | 新增 `game/replay.go`（`MoveRecord`/`GameRecord`/`Replay` + JSON 存取），GUI 增加走子记录列表、保存对局、打开复盘与逐回合控制 | `TestReplayReproducesGame`、`TestReplaySaveLoadAndStepping` 通过 |
| ISS-021 | P2 | 已修复 | 司令阵亡后未公开其军旗位置 | 缺少“司令阵亡暴露军旗”规则，残局无法定位军旗 | `applyMove` 中司令阵亡即调用 `revealFlag`（置 `Known`），`VisiblePieces` 对司令已阵亡席位的军旗不再隐藏 | `TestFlagRevealedWhenMarshalDies` 通过 |
| ISS-022 | P2 | 已修复 | 地雷交互与文档不一致 | 文档写“非工兵触雷同归于尽”，实际需“地雷保留、其他子消失”；且缺少每步停顿 | `ResolveCombat` 改为非工兵/非炸弹触雷返回 -1（地雷保留）；GUI 步间停顿改为 1 秒（`aiMoveDelay`/`replayDelay`） | `TestNonEngineerDiesAndLandmineRemains`、`TestBombAndLandmineDestroyEachOther` 通过 |
| ISS-023 | P1 | 已修复 | 普通棋子铁路拐弯位置错误 | 早期把弯道当作对角连接线（(6,5)-(5,6) 等，实际棋盘无此线） | 按棋盘实测重定：铁路全为直线；**弯道** 在中央四角 (6,6)/(6,10)/(10,6)/(10,10)。`railwayDestinations` 仅在弯道节点允许转向，直角路口必须停下；`Board.IsCurve` 标记弯道 | `TestRailwayTurnOnlyAtCurves` 通过：(9,11) 分三步到 (11,9)，(10,13) 一步直达 (13,10) |
| ISS-024 | P1 | 已修复 | 炸弹可经弯道直插中央横行（如 (14,6) 一步吃 (10,13) 司令） | 弯道允许任意 90° 转向，使普通棋子像工兵一样横穿棋盘 | 改为 **方向性弯道**：每个弯道只连相邻两方翼（`Board.CurveTurnAllowed`），只能沿该弧线转向，不能从弯道直插中央 | `TestBombCannotCutAcrossCenterCorner` 通过：(14,6) 只能拐入上方翼 (10,5)，不能到 (10,8)/(10,13) |
| ISS-025 | P1 | 已修复 | 棋子可从铁路一步直接踏上公路（含行营、大本营夺旗） | 记录中出现 6 步“铁路一步到公路”（4 步进行营、2 步夺旗），违反“铁路不能直达公路” | `LegalDestinations` 中若起点在铁路线上（`Board.IsRailway`），则只返回铁路滑行落点，不再附加公路一步 | `TestRailCannotStepOntoRoadDirectly` 通过；`TestPieceCanEnterCampDiagonally` 改为从公路节点进入行营 |
| ISS-026 | P1 | 已修复 | ISS-025 把行营一并封锁，导致铁路上的棋子无法一步进入相邻行营（含仅有铁路相邻的行营，如 (9,12)） | 铁路侧部分行营的相邻点全是铁路，封锁后该行营无法从铁路侧进入，用户对局中大量“铁路→行营”走法被判非法 | `LegalDestinations` 按落点类型区分：铁路上的棋子可一步进入相邻行营，仍不可一步踏上普通公路或大本营 | `TestRailPieceCanStepIntoAdjacentCamp`、`TestRailCannotStepOntoRoadDirectly`、e2e `rail cannot step straight onto a road` 通过 |
| ISS-027 | P1 | 已修复 | 棋盘最外一排（大本营所在排）被误判为铁路，导致铁路上的棋子可滑到/走到该排，与“最外一排是公路”不符 | 棋盘图片实测（`ico/01-棋盘.jpeg`）显示最外一排线宽 4px（公路），而铁路矩形内缩一排；`isRailwayEdge` 未做范围约束，把 (6,0)-(6,1)、(10,0)-(10,1)、(15,6)-(16,6)、(0,6)-(1,6) 等最外一排的边判成铁路，铁路可直接到达最外一排 | `isRailwayEdge` 增加约束：任何以最外一排/列为端点的边一律为公路；`LegalDestinations` 取消 ISS-026 给工兵的“一步上下任意公路”豁免，铁路上的棋子（含工兵）一律只能一步进入相邻行营 | `TestBackRowIsRoadNotRailway`、`TestNoRailwayAdjacentToHeadquarters`、`TestRailCannotReachOutermostRow`、`TestRailCannotStepOntoRoadDirectly`、`TestEngineerReachesWholeRailwayNetwork`、`TestRailPieceCanStepIntoAdjacentCamp` 通过；棋盘图片线宽探针复核（最外一排 4px=公路） |
| ISS-028 | P1 | 已修复 | 点击“开始游戏”后窗口底部出现一块重复（残留）的棋盘区域 | 点击开始后侧栏内容高度变化，使棋盘 `CustomWidget` 被重新布局、高度由 670 缩到 643；而该控件 `InvalidatesOnResize` 默认为 false，`lxn/walk` 在尺寸变化时不重绘，底部约 27px 保留了旧尺寸下的棋盘内容，形成“最下面多出一块重复区域”。另 `draw` 直接使用 `PaintFunc` 的 `bounds`（`WM_PAINT` 的 `ps.RcPaint`，实为无效区域）去 `fitImage`，局部重绘时会把棋盘缩进小矩形 | 棋盘控件增加 `InvalidatesOnResize: true`，尺寸变化时整块重绘；`draw` 改为以控件完整客户区 `boardView.ClientBounds()` 计算 `imageRect`（控件未就绪时退回传入 bounds） | 真机复现：修复前开始游戏后底部出现残留亮带（widget 670→643 未重绘），修复后恢复为连续棋盘；`go build`、`go vet`、`go test ./...` 通过 |
| ISS-029 | P1 | 已修复 | 棋子无法“扛军旗”（大本营不可达、无法夺旗）；且据报大本营内的棋子能移动 | ISS-027 把最外一排（大本营所在排）整排变为公路后，该排的相邻点**全部**位于铁路（y=1/x=15 等铁路排），而 ISS-025 的“铁路不能直达公路”又禁止从铁路踏上该排，导致最外一排成为死区、**任何棋子都无法到达大本营**。诊断：从 (8,4) 出发，己方大本营 (9,0)、敌方大本营 (16,9) 均不可达。军旗本身不可移动、占旗棋子会被置 `Immobile`，但 GUI 仍允许选中军旗/地雷，易被误认为“能动” | `Board.IsLastLine` 标记最外一排/列；`LegalDestinations` 允许铁路上的棋子一步踏上最外一排（与行营并列为例外）。新增 `Piece.Movable()`（军旗/地雷/占旗棋子均不可移动）并在 `CanMove`/`LegalDestinations`/GUI 选子处统一使用，GUI 不再能选中军旗/地雷 | `TestRailCanReachOutermostLine`、`TestHeadquartersReachableAndFlagCapturable`、`TestFlagCannotMove`、`TestRailCannotStepOntoOrdinaryRoad`、e2e `outermost line is a road, reachable only from the railway` 通过；`go build`、`go vet`、`go test ./...` 通过 |
| ISS-030 | P1 | 已修复 | 点击“开始游戏”后棋盘下方仍残留一条重复棋盘（“多渲染一层”），ISS-028 未真正修复 | ISS-028 只让棋盘控件在自身尺寸变化时重绘，未解决根因：未指定窗口初始 `Size`，`NewMainWindowWithCfg` 以工作区高度 753 创建窗口，而 `MinSize.Height=780` 使 Windows 强制把窗口增高；此时 `lxn/walk` 的状态栏 `visibleChanged` 回调 `SetBoundsPixels(BoundsPixels())` 把表单 `proposedSize` 记为 753。点击开始后 `refreshMoveLog→SetModel→RequestLayout→startLayout` 用该陈旧 `proposedSize`(753) 计算客户区(721→694)，使 `clientComposite`/棋盘收缩 27px；被让出的底部条带无人重绘（`MainWindow` 不是 `Widget`，`WM_ERASEBKGND` 直接返回 0，主窗口背景为空），旧棋盘像素残留成重复层 | 给 `MainWindow` 显式指定 `Size{Width:1200,Height:780}`（≥`MinSize`），窗口创建即达最小尺寸，不再触发“强制增高 + 陈旧 proposedSize”，`proposedSize` 与实际窗口一致 | 真机复现：修复前点击开始棋盘控件 670→643、底部残留重复棋盘条带；修复后棋盘控件尺寸不变、底部无残留；`go build`、`go vet`、`go test ./...` 通过 |
| ISS-004 | P2 | 已缓解 | 历史文档曾把规划能力描述为现有能力 | `doc/design.md` 和 `doc/rules.md` 包含网络、复盘和设置，但源码不存在；现已增加基线提示 | 后续按需求状态拆分已实现设计与产品愿景 | 发布审查不再出现未实现能力 |
| ISS-005 | P2 | 风险 | GUI、资源打包和完整对局缺少自动验证 | 自动测试仅位于 `game/game_test.go` | 增加应用层测试或 Windows UI 自动化，并保留人工冒烟记录 | 相关需求具备可重复测试证据 |
| ISS-006 | P2 | 已确认 | Manifest 身份仍是占位值 | `Four-PlayerArmyChess.manifest` 使用 `SomeFunkyNameHere`，影响制品身份规范性 | 确定正式应用标识和版本生成策略后更新资源 | 检查编译后 PE manifest |
| ISS-007 | P3 | 风险 | 运行日志固定覆盖且依赖当前目录可写 | `main.go` 使用 `os.Create("game.log")`，启动即截断；只读目录会直接退出 | 定义日志目录、轮转和故障降级策略 | 只读启动目录及重复启动测试 |

## 2. 状态规则

- `已确认`：源码、测试或命令可直接证明。
- `风险`：存在可信失败路径，但尚无复现测试。
- `待确认`：需要产品、运行环境或外部资料才能判断。
- `已缓解`：影响已通过文档或临时措施降低，但根因尚未关闭。
- 关闭问题时必须填写修复版本和验证证据，不删除历史记录。
