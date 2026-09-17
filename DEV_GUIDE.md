# 开发指南

本仓库是 **空想云特别企业版**（`reverix_minimal_distribution`），基于 Sub2API 裁切。本文给协作开发用。

## 仓库

| 项 | 说明 |
|---|---|
| 公开仓库 | [GrooveSino/reverix_minimal_distribution](https://github.com/GrooveSino/reverix_minimal_distribution) |
| 上游 | [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api) |
| 默认分支 | `main`（本地开发分支名为 `req99-cut`） |
| 技术栈 | Go 后端（Ent + Gin）+ Vue 3 前端（pnpm） |
| 数据 | PostgreSQL + Redis |

## 环境

- Go **1.27.0**（`backend/go.mod` 指定，可用 `GOTOOLCHAIN=go1.27.0`）
- Node 24 + **pnpm**（不要用 npm 装前端）
- 交叉编译 Linux 二进制：`CGO_ENABLED=0 GOOS=linux GOARCH=amd64`

```bash
# 前端
pnpm --dir frontend install
pnpm --dir frontend exec vite build   # 产物在 backend/internal/web/dist

# 后端测试
cd backend
GOTOOLCHAIN=go1.27.0 go test -tags=unit ./...

# 嵌入前端后出 Linux 包
cd backend
GOTOOLCHAIN=go1.27.0 CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -tags=embed -trimpath -o ../dist/sub2api ./cmd/server
```

## 本发行版约定

1. **默认中文**，不要加回语言切换器。
2. **个人用户菜单**只保留：仪表盘、密钥、用量、渠道状态、模型广场、改密码。
3. **模型广场**只显示一列「价格」，不要「实付 / 官方」对照。
4. **不要把上游中转域名写进渠道名、分组名、Key 名或对用户可见的说明。**
5. **不要在用户账号里留下测试 Key。** 联调请用管理员自己的 Key。
6. 替换生产二进制时只换 `/opt/sub2api/sub2api`，不要重跑安装、不要动数据库和 `config.yaml`。

## 提交与推送

```bash
git checkout req99-cut
git add -A
git commit -m "说明这次改了什么"
git push github req99-cut:main
```

远程 `github` 指向本发行版仓库，`origin` 仍指向上游 Sub2API，请勿把定制提交推到上游。
