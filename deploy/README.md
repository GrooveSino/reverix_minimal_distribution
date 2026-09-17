# 部署说明

本目录用于在 Linux 服务器或 Apple Silicon Mac 上部署空想云特别企业版。

## 方式

| 方式 | 适用 | 安装向导 |
|---|---|---|
| Docker Compose | 快速拉起全套 | 一般不需要（自动配置） |
| Apple container | macOS 26 本机 | 一般不需要 |
| 二进制 + systemd | 生产机 | Web 向导；本发行版现网是直接替换二进制 |

## 文件

| 文件 | 说明 |
|---|---|
| `docker-compose.yml` | Docker Compose（命名卷） |
| `docker-compose.local.yml` | Docker Compose（本地目录，方便迁移） |
| `docker-deploy.sh` | 一键 Docker 部署 |
| `apple-container.sh` | Apple `container` 生命周期脚本 |
| `APPLE_CONTAINER.md` | Apple container 运维说明 |
| `.env.example` | 容器环境变量模板 |
| `DOCKER.md` | Docker 镜像说明 |
| `install.sh` | 二进制一键安装 |
| `install-datamanagementd.sh` | datamanagementd 安装 |
| `sub2api.service` | systemd 单元 |
| `sub2api-datamanagementd.service` | datamanagementd 单元 |
| `DATAMANAGEMENTD_CN.md` | datamanagementd 联动说明 |
| `config.example.yaml` | 配置示例 |
| `EDGE_SECURITY.md` | 反代、CDN/WAF、可信代理与入口加固 |

## 生产替换（本发行版常用）

不要重跑 `install.sh`，也不要动 PostgreSQL、Redis、Caddy、`config.yaml`。只替换二进制：

```bash
systemctl stop sub2api
install -o sub2api -g sub2api -m 755 ./sub2api /opt/sub2api/sub2api
systemctl start sub2api
```

替换前建议先备份旧文件：`cp -a /opt/sub2api/sub2api /opt/sub2api/sub2api.bak`

## Docker

```bash
./docker-deploy.sh
```

详见 [DOCKER.md](./DOCKER.md)。

## Apple container

```bash
./apple-container.sh init
./apple-container.sh up
./apple-container.sh status
```

详见 [APPLE_CONTAINER.md](./APPLE_CONTAINER.md)。
