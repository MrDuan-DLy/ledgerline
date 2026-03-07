<div align="center">

# Ledgerline

**隐私优先的个人财务工具**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](backend-go/)
[![React](https://img.shields.io/badge/React-18-61DAFB?logo=react&logoColor=white)](frontend/)
[![SQLite](https://img.shields.io/badge/SQLite-local-003B57?logo=sqlite&logoColor=white)]()
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white)](Dockerfile)
[![CI](https://github.com/MrDuan-DLy/ledgerline/actions/workflows/ci.yml/badge.svg)](https://github.com/MrDuan-DLy/ledgerline/actions/workflows/ci.yml)

[English](README.md) · **中文**

</div>

## 功能特性

- **银行账单导入** — 使用 Gemini AI 提取 PDF 账单，CSV 导入自动去重
- **交易管理** — 查看、搜索、筛选和分类交易
- **收据识别** — 上传收据图片，OCR 提取并智能匹配交易
- **自动分类** — 基于规则的交易分类，支持商户学习
- **预算追踪** — 按类别设置和跟踪月度支出
- **数据看板** — 月度支出图表、消费节奏、分类统计
- **隐私优先** — 所有数据本地 SQLite 存储，无云同步

## 截图

> 即将添加。

## 快速开始

### Docker（推荐）

```bash
# 克隆仓库
git clone https://github.com/MrDuan-DLy/ledgerline.git
cd ledgerline

# 配置
cp .env.example .env
# 编辑 .env — 添加 Gemini API 密钥（可选，用于 AI 功能）

# 运行
docker compose up -d
```

打开 [http://localhost:8000](http://localhost:8000)。

### 从源码构建

前置要求：Go 1.24+、Node.js 20+

```bash
# 构建并运行
make all    # 构建后端 + 前端
make run    # 启动服务器

# 开发热重载
make dev
```

## 配置说明

将 `.env.example` 复制为 `.env`：

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `GEMINI_API_KEY` | Google Gemini API 密钥（用于 PDF/收据 AI） | — |
| `GEMINI_MODEL` | Gemini 模型 | `gemini-2.5-flash` |
| `PORT` | 服务端口 | `8000` |
| `DATA_DIR` | 数据目录 | `./data` |
| `DEFAULT_CURRENCY` | 货币代码 | `GBP` |
| `DEFAULT_CURRENCY_SYMBOL` | 货币符号 | `£` |
| `DEFAULT_LOCALE` | 区域设置 | `en-GB` |
| `CORS_ORIGINS` | 允许的跨域来源 | `http://localhost:5173` |

## 架构概览

```
backend-go/           # Go 后端（Chi 路由、sqlx、modernc/sqlite）
├── cmd/server/       # 入口
├── internal/
│   ├── config/       # 配置
│   ├── database/     # 数据库 + goose 迁移
│   ├── handlers/     # HTTP 处理器
│   ├── middleware/    # 中间件（CORS、日志、请求限制）
│   ├── models/       # 数据模型
│   ├── parsers/      # 账单解析器
│   └── services/     # 业务逻辑
frontend/             # React + Vite + TypeScript
├── src/
│   ├── pages/        # 页面组件
│   ├── components/   # 共享 UI 组件
│   ├── contexts/     # 应用配置上下文
│   └── api/          # API 客户端
data/                 # SQLite 数据库 + 上传文件（已 gitignore）
```

Go 后端同时提供 REST API（`/api/*`）和前端静态文件。使用 `modernc.org/sqlite`（纯 Go，无需 CGo）。

## 添加自定义银行解析器

1. 在 `backend-go/internal/parsers/` 中创建新解析器
2. 实现 `Parser` 接口：
   ```go
   type Parser interface {
       Parse(content []byte) (*ParseResult, error)
   }
   ```
3. 在 `handlers/statements.go` 的上传处理器中注册
4. 解析器应返回包含交易、日期范围和余额的 `ParseResult`

对于 PDF 账单，内置的 Gemini AI 解析器支持任意银行格式。

## 商户分类配置

参考 `configs/examples/` 中的 YAML 配置模板：
- `categories.yaml` — 交易分类
- `rules_uk.yaml` — 分类规则
- `merchants_uk.yaml` — 商户定义

## API 接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/transactions` | GET | 交易列表（分页） |
| `/api/transactions/{id}` | PATCH | 更新交易 |
| `/api/transactions/bulk-classify` | POST | 批量分类 |
| `/api/categories` | GET/POST | 分类管理 |
| `/api/rules` | GET/POST | 分类规则 |
| `/api/statements/upload` | POST | 上传银行账单 |
| `/api/imports/upload` | POST | 上传 AI 审核 |
| `/api/receipts/upload` | POST | 上传收据图片 |
| `/api/budgets` | GET/POST | 预算管理 |
| `/api/merchants` | GET/POST | 商户管理 |
| `/health` | GET | 健康检查 |

## 开发命令

```bash
make dev              # 后端 + 前端热重载开发
make test             # 运行 Go 测试
make lint             # Go 代码检查
make frontend-lint    # 前端 ESLint + 类型检查
make docker-build     # 构建 Docker 镜像
```

## 许可证

[MIT](LICENSE)

## 贡献

参见 [CONTRIBUTING.md](CONTRIBUTING.md)。
