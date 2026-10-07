# 测试计划、记录与需求追踪

## 1. 测试策略

按风险从低层到高层验证：`game` 包自动测试负责稳定的规则逻辑；构建检查负责依赖和 Windows 编译；人工冒烟负责 Walk 窗口、图片、点击、动画、声音和制品布局。

规则或棋盘拓扑变更应运行全部测试。纯文档变更至少检查链接、命令和需求状态一致性。

## 2. 自动测试追踪

| 需求 | 测试证据 | 覆盖状态 |
| --- | --- | --- |
| FR-002、FR-003 | `TestAutomaticSetupObeysEverySeatRule` | 已覆盖 |
| FR-005 | `TestOrdinaryPieceSlidesStraightAndStopsAtRightAngle`、`TestOrdinaryPieceFollowsRailwayCurveButNotRightAngle`、`TestRailwayBlockerStopsLongMove`、`TestEngineerCanTurnOnRailwayAndUseRoad`、`TestEngineerReachesWholeRailwayNetwork`、`TestPieceCanEnterCampDiagonally`、`TestRailCannotStepOntoOrdinaryRoad`、`TestRailPieceCanStepIntoAdjacentCamp`、`TestBackRowIsRoadNotRailway`、`TestNoRailwayAdjacentToHeadquarters`、`TestRailCanReachOutermostLine`、`TestHeadquartersReachableAndFlagCapturable`、`TestFlagCannotMove`，以及 e2e `TestDocumentedFlowsEndToEnd` 的 `outermost line is a road, reachable only from the railway` 子用例 | 部分覆盖 |
| FR-006、FR-007 | `TestCombatRules` | 部分覆盖，仅验证判定函数 |
| FR-008、NFR-006 | `TestVisibilityProjectionHidesOtherSeats` | 已覆盖活动对局 |
| FR-009 | `TestTurnOrderCyclesCounterClockwise` | 已覆盖基础轮转 |
| FR-011 | `TestOppositeSeatsDetermineWinner` | 部分覆盖 |
| 棋盘业务规则 | `TestBoardTopologyContainsImageLayout` | 已覆盖节点、行营和中心节点 |
| FR-001、FR-004、FR-010、FR-012 | 无自动化 GUI 测试 | 人工验证 |

## 3. 待补自动测试

- 无效坐标、移动己方目标、非当前席位、军旗/地雷移动等拒绝路径。
- 攻击行营后双方位置不变但回合轮转的完整 `Game.Move` 行为。
- 每一种等级比较、双方炸弹、旗帜消除席位、两队胜利分支。
- 普通铁路在转弯、敌方阻挡、己方阻挡和多入口场景的边界。
- 所有席位无合法移动时的 AI 调度终止策略。
- 游戏结束和回放阶段的可见性策略。

## 4. 人工冒烟用例

| ID | 步骤 | 预期结果 |
| --- | --- | --- |
| MT-001 | 从发布目录启动 `Four-PlayerArmyChess.exe` | 窗口正常出现，棋盘图片完整，无启动错误 |
| MT-002 | 点击“开始游戏” | 显示棋子，用户视角为下方绿色 3 号位，其他棋子显示 `?` |
| MT-003 | 选中己方可移动棋子 | 选中高亮，合法落点显示，不合法位置不执行移动 |
| MT-004 | 完成一次合法移动 | 出现移动效果和提示音，三个 AI 依次行动后回到用户回合 |
| MT-005 | 触发吃子 | 执行正确结果，出现战斗效果和战斗提示音 |
| MT-006 | 点击“结束游戏” | 对局停止，后续棋盘点击和 AI 行动不再改变状态 |
| MT-007 | 从不同工作目录启动制品 | 仍从可执行文件旁的 `ico/01-棋盘.jpeg` 加载图片 |
| MT-008 | 检查 `game.log` | 包含启动/退出或错误信息，不包含敌方棋子身份 |

## 5. 2026-08-06 验证记录

| 检查 | 结果 | 证据/限制 |
| --- | --- | --- |
| 本地 Go 版本 | 通过 | `go version` 为 `go1.26.5 windows/amd64`，满足 `go.mod` 的 Go 1.25 要求 |
| `go test ./game` | 通过 | `ok Four-PlayerArmyChess/game (cached)` |
| `go vet ./game` | 通过 | 无输出、退出码为 0 |
| `gofmt -d .` | 通过 | 无格式差异 |
| `go test ./...` | 未执行完成 | `github.com/lxn/walk` 无法从 `proxy.golang.org` 下载；`Four-PlayerArmyChess/game` 已通过 |
| `go vet ./...` | 未执行完成 | 被同一 Walk 依赖下载失败阻塞 |
| Windows 构建 | 未执行完成 | 被同一 Walk 依赖下载失败阻塞 |
| 人工 GUI 冒烟 | 未执行 | 本次仅建立工程文档基线 |

环境恢复后应依次执行：

```powershell
go test ./...
go vet ./...
go build -ldflags="-H windowsgui" -o Four-PlayerArmyChess.exe .
```

## 6. 通过准则

- 自动测试、静态检查和构建全部通过。
- P0/P1 缺陷为零，P2 缺陷有明确接受人和处置版本。
- MT-001 至 MT-008 在声明支持的 Windows 版本通过。
- 所有“已实现”需求至少有自动测试或人工用例证据。
