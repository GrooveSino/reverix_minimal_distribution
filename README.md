<div align="center">

<img src="assets/logo.svg" alt="空想云" width="128" />

# 空想云（特别企业版）

**Reverix Minimal Distribution**

空想云（广州）计算机技术有限公司内部发行版。默认中文，个人用户界面已裁切，只保留实际使用所需能力。

</div>

## 这是什么

本仓库是基于开源项目 [Sub2API](https://github.com/Wei-Shaw/sub2api) 的定制发行版，用于企业内部 API 网关。

个人用户登录后可以看到：

- 仪表盘
- API 密钥
- 使用记录
- 渠道状态
- 模型广场（只展示一列「价格」）
- 修改密码

个人用户看不到兑换、订阅、支付、推广、资料编辑、语言切换等自助运营能力。管理员后台保持完整。

## 团队公池

管理员配置团队累计总额度，所有成员共享公池，同时保留个人余额与独立消费记录。公池耗尽后暂停全员新调用，追加额度后恢复。首次升级需导入历史消费并设置总额度，详见 [公池使用与升级说明](docs/team-balance-pool.md)。

## 品牌

| 项 | 内容 |
|---|---|
| 产品名 | 空想云 |
| 副标题 | 特别企业版 |
| 公司 | 空想云（广州）计算机技术有限公司 |
| 仓库 | [GrooveSino/reverix_minimal_distribution](https://github.com/GrooveSino/reverix_minimal_distribution) |

## 技术栈

- 后端：Go 1.27、Gin、Ent、PostgreSQL、Redis
- 前端：Vue 3、Vite、pnpm、Tailwind
- 部署：Linux amd64 静态二进制 + Caddy / systemd

## 本地开发

```bash
# 前端
pnpm --dir frontend install
pnpm --dir frontend exec vite build

# 后端（嵌入前端）
cd backend
GOTOOLCHAIN=go1.27.0 CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -tags=embed -trimpath -o ../dist/sub2api ./cmd/server
```

更细的环境说明见 [DEV_GUIDE.md](DEV_GUIDE.md)，部署文件见 [deploy/README.md](deploy/README.md)。

## 许可证

上游 Sub2API 使用 LGPL-3.0，见 [LICENSE](LICENSE)。本发行版在其基础上修改，贡献请阅读 [CLA.md](CLA.md)。

使用本项目可能违反上游模型服务商的用户协议，请自行评估合规风险。本仓库按现状提供，空想云（广州）计算机技术有限公司不对使用后果承担责任。
