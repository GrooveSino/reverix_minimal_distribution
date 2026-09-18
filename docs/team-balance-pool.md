# 团队公池余额

所有用户（包括管理员、配额管理员和新增用户）共享一个公池，同时保留各自的个人余额。

`公池剩余余额 = 管理员设置的累计总额度 − 团队累计实际消费`

例如总额度为 $10,000，历史消费 $1,000，则公池剩余 $9,000。某个用户再消费 $100，其个人余额和公池各减少 $100，公池变为 $8,900。给某个人充值或调整个人余额不会改变公池。

## 使用

- 用户仪表盘展示团队剩余余额、累计总额度、累计消费及个人余额；公池每 10 秒刷新，也可手动刷新。
- 管理员仪表盘提供“设置累计总额度”和“追加额度”。设置 $10,000 的含义是累计预算为 $10,000，**不是把剩余余额重置为 $10,000**。
- 只有 `admin` 可以修改；`quota` 和普通用户只能读取。调整记录写入 `team_balance_adjustments`，同时沿用管理端审计中间件。
- 修改请求必须提交读取时的 `revision`。并发修改或重放返回 HTTP 409，不会重复追加；出现网络错误后应刷新核对，不自动重发。
- 公池剩余 ≤ 0，所有新调用返回余额不足；个人余额不足只影响该成员。数据库读取失败返回 503，不放行。
- 已开始的请求继续结算，允许公池变成负数。追加额度后需使剩余余额大于零才能恢复。
- 余额查询、已完成异步图片结果查询仍可使用。订阅配额继续统计，但订阅和简易模式不能绕过本企业版的个人余额与公池扣费。

## 首次升级

1. 备份数据库，停止接收新网关请求，等待已有请求、用量写入队列和异步任务完成。停止所有旧版本实例后再升级，**不要混跑新旧扣费程序**。
2. 启动新版本，执行 `239_team_balance_pool.sql`。迁移按 `usage_logs.actual_cost`（实际收费，含倍率）导入现存的全体历史消费，不按用户是否已删除进行过滤。只导入一次。
3. 初始总额度为零，网关会暂停新调用；管理员登录仪表盘设置真实的累计总额度后恢复。示例中的 $10,000 不会自动作为生产初始额度。
4. 核对导入消费和剩余余额后恢复流量。

历史导入以仍在数据库中的使用记录为依据。如果旧记录曾被清理或写入失败，已丢失的消费无法自动重建；升级前需从备份恢复或核对账本。升级后的消费保存在独立累计计数器中，删除用户或清理用量记录不会释放已消费额度。

个人扣费、公池扣费和请求去重在同一个 PostgreSQL 事务中提交。批量图片在最终结算时扣公池，冻结/释放个人余额不扣公池。公池准入直接读取主数据库，无跨实例余额缓存延迟。页面轮询会有最多约 10 秒显示延迟，不影响后端即时拦截。

## 接口

- `GET /api/v1/user/team-balance`：登录用户读取。
- `GET /api/v1/admin/team-balance`：管理员读取。
- `POST /api/v1/admin/team-balance`：`{"operation":"add","amount":5000,"revision":1}`；`operation` 也支持 `set`，金额采用 USD、最多 8 位小数。

## 验证

```bash
cd backend
go test -tags=unit ./internal/service ./internal/repository ./internal/handler ./internal/server/middleware -run 'TestTeamBalance'
# 可选真实数据库验证：测试会创建并删除独立 schema，不修改现有业务表。
TEAM_BALANCE_TEST_DSN='postgres://user@127.0.0.1:5432/testdb?sslmode=disable' \
  go test -tags=unit ./internal/repository -run TestTeamBalancePostgres -count=1

cd ../frontend
pnpm exec vitest run src/components/common/__tests__/TeamBalanceCard.spec.ts
```
