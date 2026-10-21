# Dashboard Failures Implementation Plan

**Goal:** 概况页避免历史日志全量统计和 N+1 查询，提供最新失败可见性。
**Architecture:** Task 持久化最近日志摘要，TaskLogAdd 事务更新；概况 SQL 返回定长摘要和名称；站内摘要轮询。
**Tech Stack:** Go, Beego ORM, MySQL, Bootstrap 2, jQuery 1.7。
**Spec:** docs/superpowers/specs/2026-10-21-dashboard-failures-design.md

## Constraints
保留所有已有工作区改动；不发布、不提交、不连接现有数据库。

- [x] models/task.go、task_log.go、dashboard.go：字段、事务结果更新、概况摘要查询、失败摘要；install.sql 与独立迁移含回填和索引。
- [x] controllers/main.go、task.go，jobs/cron.go、job.go：替换概况数据来源、失败接口与过滤、排除系统 cron、超时邮件条件。
- [x] views/layout/layout.html、task/list.html、main/index.html，静态 CSS/JS：最新结果、失败筛选、全局提示及轮询异常状态。
- [x] 隔离 MySQL 验证迁移/回填、成功解除失败、错误摘要、并发顺序、性能样本；Go测试和编译；浏览器检查；独立代码审查。

## 验证记录
- `go test ./app/...`、编译、`git diff --check`通过；测试覆盖系统 cron 过滤、超时邮件、模板转义。
- `WEBCRON_TEST_DSN` 指向隔离 MySQL 8.0，`go test ./app/models -run TestDashboardMySQL -v` 通过：新建结构、摘要大小、当前失败、恢复、组合筛选、陈旧配置不覆盖结果、插入失败回滚、日志清理保留结果、旧结构迁移回填。
- 250任务/100000日志本地样本，旧概况数据查询五次耗时44.79/31.27/33.49/21.20/20.77ms；新摘要查询3.10/1.25/1.21/1.51/1.40ms。完整概况HTTP五次中位6.09ms。不是线上性能结果。
- Playwright 检查登录后的真实隔离应用：顶部3个失败、过滤3行、错误中的script按文本转义、成功执行后当前失败归零；历史失败仍保留。
- 独立代码审查未发现可操作缺陷。
