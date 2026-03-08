<div align="center">

<img src="docs/assets/banner.png" alt="Ledgerline" width="600" />

<br/>

**Privacy-first personal finance tracker / 隐私优先的个人财务工具**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](backend-go/)
[![React](https://img.shields.io/badge/React-18-61DAFB?logo=react&logoColor=white)](frontend/)
[![SQLite](https://img.shields.io/badge/SQLite-local-003B57?logo=sqlite&logoColor=white)]()
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white)](Dockerfile)
[![CI](https://github.com/MrDuan-DLy/ledgerline/actions/workflows/ci.yml/badge.svg)](https://github.com/MrDuan-DLy/ledgerline/actions/workflows/ci.yml)

[English](#features--功能) · [中文](README_CN.md)

</div>

## Features / 功能

- **Bank statement import** — Parse PDF statements with Gemini AI extraction, CSV import with auto-deduplication
  - 银行账单导入 — 使用 Gemini AI 提取 PDF 账单，CSV 导入自动去重
- **Transaction management** — View, search, filter, and categorize transactions
  - 交易管理 — 查看、搜索、筛选和分类交易
- **Receipt capture** — Upload receipt images with OCR extraction and smart matching
  - 收据识别 — 上传收据图片，OCR 提取并智能匹配
- **Auto-classification** — Rule-based transaction categorization with merchant learning
  - 自动分类 — 基于规则的交易分类，支持商户学习
- **Budgets** — Set and track monthly spending by category
  - 预算追踪 — 按类别设置和跟踪月度支出
- **Dashboard** — Monthly spend charts, spending pace, category breakdown
  - 数据看板 — 月度支出图表、消费节奏、分类统计
- **Privacy-first** — All data stored locally in SQLite, no cloud sync
  - 隐私优先 — 所有数据本地 SQLite 存储，无云同步

## Screenshots / 截图

> Coming soon. / 即将添加。

## Quick Start / 快速开始

### Docker (recommended / 推荐)

```bash
# Clone the repo / 克隆仓库
git clone https://github.com/MrDuan-DLy/ledgerline.git
cd ledgerline

# Configure / 配置
cp .env.example .env
# Edit .env — add your Gemini API key (optional, for AI features)
# 编辑 .env — 添加 Gemini API 密钥（可选，用于 AI 功能）

# Run / 运行
docker compose up -d
```

Open [http://localhost:8000](http://localhost:8000).

### From Source / 从源码构建

Prerequisites / 前置要求: Go 1.24+, Node.js 20+

```bash
# Build and run / 构建并运行
make all    # Build backend + frontend
make run    # Start server

# Development with hot reload / 开发热重载
make dev
```

## Configuration / 配置

Copy `.env.example` to `.env`:

| Variable | Description / 说明 | Default |
|----------|-------------------|---------|
| `GEMINI_API_KEY` | Google Gemini API key (for PDF/receipt AI) / Gemini API 密钥 | — |
| `GEMINI_MODEL` | Gemini model / 模型 | `gemini-2.5-flash` |
| `PORT` | Server port / 端口 | `8000` |
| `DATA_DIR` | Data directory / 数据目录 | `./data` |
| `DEFAULT_CURRENCY` | Currency code / 货币代码 | `GBP` |
| `DEFAULT_CURRENCY_SYMBOL` | Currency symbol / 货币符号 | `£` |
| `DEFAULT_LOCALE` | Locale / 区域设置 | `en-GB` |
| `CORS_ORIGINS` | Allowed CORS origins / 允许的跨域来源 | `http://localhost:5173` |

## Architecture / 架构

```
backend-go/           # Go backend (Chi router, sqlx, modernc/sqlite)
├── cmd/server/       # Entry point / 入口
├── internal/
│   ├── config/       # Configuration / 配置
│   ├── database/     # DB + goose migrations / 数据库 + 迁移
│   ├── handlers/     # HTTP handlers / HTTP 处理器
│   ├── middleware/    # CORS, logging, body limits / 中间件
│   ├── models/       # Data models / 数据模型
│   ├── parsers/      # Bank statement parsers / 账单解析器
│   └── services/     # Business logic / 业务逻辑
frontend/             # React + Vite + TypeScript
├── src/
│   ├── pages/        # Dashboard, Transactions, Import, Receipts, etc.
│   ├── components/   # Shared UI components / 共享组件
│   ├── contexts/     # App configuration context / 应用配置
│   └── api/          # API client / API 客户端
data/                 # SQLite database + uploads (gitignored)
```

The Go backend serves both the REST API (`/api/*`) and frontend static files. SQLite via `modernc.org/sqlite` (pure Go, no CGo).

## Adding a Custom Bank Parser / 添加自定义银行解析器

1. Create a new parser in `backend-go/internal/parsers/`
2. Implement the `Parser` interface:
   ```go
   type Parser interface {
       Parse(content []byte) (*ParseResult, error)
   }
   ```
3. Register the parser in `handlers/statements.go` Upload handler
4. The parser should return `ParseResult` with transactions, period dates, and balances

For PDF statements, the built-in Gemini AI parser works with any bank format.
对于 PDF 账单，内置的 Gemini AI 解析器支持任意银行格式。

## Merchant Classification / 商户分类配置

See `configs/examples/` for YAML configuration templates:
- `categories.yaml` — Transaction categories / 交易分类
- `rules_uk.yaml` — Classification rules / 分类规则
- `merchants_uk.yaml` — Merchant definitions / 商户定义

## API / 接口

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/transactions` | GET | List transactions (paginated) |
| `/api/transactions/{id}` | PATCH | Update transaction |
| `/api/transactions/bulk-classify` | POST | Bulk classify |
| `/api/categories` | GET/POST | Category CRUD |
| `/api/rules` | GET/POST | Classification rules |
| `/api/statements/upload` | POST | Upload bank statement |
| `/api/imports/upload` | POST | Upload for AI review |
| `/api/receipts/upload` | POST | Upload receipt image |
| `/api/budgets` | GET/POST | Budget management |
| `/api/merchants` | GET/POST | Merchant management |
| `/health` | GET | Health check |

## Development / 开发

```bash
make dev              # Backend + frontend with hot reload / 热重载开发
make test             # Run Go tests / 运行测试
make lint             # Run Go linter / 代码检查
make frontend-lint    # Frontend ESLint + typecheck / 前端检查
make docker-build     # Build Docker image / 构建 Docker 镜像
```

## License / 许可证

[MIT](LICENSE)

## Contributing / 贡献

See [CONTRIBUTING.md](CONTRIBUTING.md).
