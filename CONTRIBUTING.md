# Contributing to Ledgerline / 贡献指南

Thank you for your interest in contributing!

感谢你对本项目的关注！

## Development Setup / 开发环境

1. **Prerequisites / 前置要求**: Go 1.24+, Node.js 20+
2. **Clone / 克隆**: `git clone` and `cd` into the repo / 克隆仓库并进入目录
3. **Backend / 后端**: `cd backend-go && go build ./cmd/server`
4. **Frontend / 前端**: `cd frontend && npm install`
5. **Run / 运行**: `make dev` for both with hot reload / 热重载启动

## Code Style / 代码风格

- **Go**: Follow standard Go conventions. Run `golangci-lint` before submitting.
  - 遵循 Go 标准规范，提交前运行 `golangci-lint`。
- **TypeScript/React**: Follow existing patterns. Run `npm run lint` in `frontend/`.
  - 遵循现有代码风格，在 `frontend/` 中运行 `npm run lint`。
- **Commits / 提交**: Use conventional commit messages (`feat:`, `fix:`, `docs:`, `chore:`)
  - 使用约定式提交信息。

## Pull Request Process / PR 流程

1. Fork the repository / 复刻仓库
2. Create a feature branch / 创建功能分支 (`git checkout -b feat/my-feature`)
3. Make your changes / 提交更改
4. Run tests / 运行测试: `make test && make frontend-lint`
5. Submit a pull request with a clear description / 提交 PR 并附上清晰的描述

## Reporting Issues / 报告问题

Please use GitHub Issues. Include: / 请使用 GitHub Issues，包含以下信息：
- Steps to reproduce / 复现步骤
- Expected vs actual behavior / 预期与实际行为
- Environment details (OS, Go version, Node version) / 环境信息

## Areas for Contribution / 贡献方向

- New bank statement parsers / 新的银行账单解析器 (see `backend-go/internal/parsers/`)
- UI improvements / UI 改进
- Documentation and translations / 文档和翻译
- Test coverage / 测试覆盖
