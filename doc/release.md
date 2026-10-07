# 构建、发布与维护指南

## 1. 发布前提

- Windows/amd64 环境。
- Go 1.25.x，与 `go.mod` 一致。
- 可访问或已缓存 `go.sum` 中的依赖。
- 工作区仅包含本次发布认可的源码、资源和文档。

## 2. 候选版本检查

```powershell
go version
go mod verify
go test ./...
go vet ./...
go build -trimpath -ldflags="-H windowsgui" -o Four-PlayerArmyChess.exe .
```

构建完成后创建如下发布目录：

```text
Four-PlayerArmyChess-release/
├── Four-PlayerArmyChess.exe
├── ico/
│   └── 01-棋盘.jpeg
├── README.md
└── RELEASE_NOTES.md
```

不要把 `game.log`、`output.log`、`output.txt`、IDE 配置或源码目录作为运行必需文件发布。

## 3. 发布门禁

1. 记录版本号、日期、Go 版本、目标系统、需求和缺陷清单。
2. 执行 `doc/test-and-traceability.md` 中全部自动检查和人工冒烟。
3. 确认从发布目录和其他工作目录均能启动并加载图片。
4. 计算并记录 `Four-PlayerArmyChess.exe` 与棋盘图片的 SHA-256。
5. 保留上一个已验证发布目录，作为当前无数据迁移场景下的回退制品。

生成校验和示例：

```powershell
Get-FileHash -Algorithm SHA256 .\Four-PlayerArmyChess.exe
Get-FileHash -Algorithm SHA256 .\ico\01-棋盘.jpeg
```

## 4. 发布说明模板

```markdown
# FourJun <version>

- 发布日期：YYYY-MM-DD
- 目标平台：Windows/amd64
- Go 版本：go1.25.x
- 新增/变更：关联需求 ID
- 修复：关联问题 ID
- 已知问题：未关闭 P2/P3
- 验证：测试、静态检查、构建、人工冒烟结果
- 制品校验和：SHA-256
```

## 5. 回退

当前程序没有持久化数据和升级迁移。发生发布阻断时，停止新版本分发，恢复上一个完整发布目录，并记录失败现象、日志、系统版本和复现步骤。禁止只替换可执行文件而保留未知版本资源。

## 6. 维护

用户问题统一登记到 `doc/issues.md`，先分级再修改。修复完成后补充回归测试、验证记录和发布说明。涉及隐藏信息的日志或截图在共享前应检查是否暴露棋子身份。
