# webcron 部署指导

本文记录 webcron 在「线上 Linux 服务器无 Go 环境」时的部署方式。

## 场景说明

- 本地机器负责交叉编译 Linux amd64 版本。
- 线上服务器只负责替换文件和重启服务。
- 不需要在线上安装 Go。
- 不建议整包覆盖线上目录，避免覆盖 `conf/app.conf` 中的线上配置。

---

## 本次发布：MFA + 操作审计 + 账号管理（2026-06-23）

### 涉及改动

1. **MFA 多因素认证**：登录强制绑定 TOTP（Google Authenticator / 2FAS / Aegis 扫码），扫码绑定，输入 6 位验证码（满 6 位自动提交）。
2. **操作审计**：记录登录、MFA、任务、分组、账号等所有关键操作的审计日志，仅 admin 可见。
3. **账号管理**：新增账号管理功能，仅 admin 可操作。添加账号时密码由服务端随机生成，仅创建成功后一次性弹窗展示。
4. **权限控制**：除 admin 外，普通用户看不到也无法访问「账号管理」「操作审计」。
5. **本地模式**：新增 `jobs.local` 开关，开启后不自动执行任何定时任务（仅本地调试用，生产环境必须为 `false`）。

### 数据库

本次使用**全新数据库**，直接用 `install.sql` 建表即可，无需执行增量迁移脚本。

- `install.sql`：全量建表脚本（含 `t_user` 的 MFA 字段、`create_time` 字段，及 `t_operation_log` 表）。
- `migrations/2026-06-23-security-account-mfa-audit.sql`：增量迁移脚本，仅当在已有旧库上升级时使用（本次不需要）。

> 注：本次发布前已在新库（`cm_cron`）执行过 `install.sql`，表结构已就绪。如服务器连同一库则无需再建表。

---

## 本地打包

### 方式一：使用打包脚本（需 bee）

```bash
cd /Users/wind/wwwroot/webcron
./pack.sh 20260623-mfa-audit
```

生成 `webcron-20260623-mfa-audit.tar.gz`。

### 方式二：手动交叉编译（本次实际使用）

不依赖 bee，直接 `go build`：

```bash
cd /Users/wind/wwwroot/webcron

# 1. 交叉编译 Linux amd64 二进制
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/webcron-pkg/webcron .

# 2. 准备生产配置（关键：jobs.local = false）
#    将本地 conf/app.conf 的 jobs.local 改为 false 后复制
cp conf/app.conf /tmp/webcron-pkg/conf/app.conf  # 注意先改 jobs.local=false

# 3. 复制运行时所需目录和文件
cp -R views /tmp/webcron-pkg/views
cp -R static /tmp/webcron-pkg/static
cp run.sh /tmp/webcron-pkg/run.sh
cp install.sql /tmp/webcron-pkg/install.sql
cp -R migrations /tmp/webcron-pkg/migrations

# 4. 清理 .DS_Store 并打包
find /tmp/webcron-pkg -name '.DS_Store' -delete
tar -czf /tmp/webcron-20260623-mfa-audit.tar.gz -C /tmp/webcron-pkg .
```

### 校验打包结果

```bash
# 确认二进制是 Linux amd64
tar -xzf /tmp/webcron-20260623-mfa-audit.tar.gz -C /tmp/verify webcron
file /tmp/verify/webcron
# 期望输出: ELF 64-bit LSB executable, x86-64, statically linked
```

本次打包结果：

- 文件：`/tmp/webcron-20260623-mfa-audit.tar.gz`
- 大小：11M
- 内容：二进制 + conf + views + static + run.sh + install.sql + migrations

---

## 线上部署

### 前置准备

1. 确认服务器有解压工具（`tar`）。
2. 确认服务器到 RDS（`pub-2zejby37y6q61sktn.rwlb.rds.aliyuncs.com:3306`）网络可达。
3. 数据库表结构已就绪（本次发布前已建好；若用全新库则执行 `install.sql`）。

### 部署步骤

将打包文件上传到服务器后，执行：

```bash
# 1. 选择部署目录（假设为 /www/wwwroot/webcron）
cd /www/wwwroot/webcron

# 2. 若为首次部署，创建目录
mkdir -p /www/wwwroot/webcron && cd /www/wwwroot/webcron

# 3. 上传包后解压（整包解压到部署目录）
tar -xzvf /上传路径/webcron-20260623-mfa-audit.tar.gz -C /www/wwwroot/webcron

# 4. 确认配置正确
cat conf/app.conf
# 重点检查：
#   - jobs.local = false      （生产必须为 false）
#   - runmode = prod           （生产模式）
#   - db.host / db.password   （指向正确的库）

# 5. 赋予执行权限
chmod +x webcron run.sh

# 6. 停止旧进程（二选一）
#    方式一：用 run.sh（内部 killall webcron，适合进程名就是 webcron）
./run.sh stop
#    方式二：手动查杀（旧进程改名/多实例/路径不同时更稳）
ps -ef | grep webcron | grep -v grep
#    记下 PID 后 kill，例如：
#    kill 12345
#    若进程不退出可强制：kill -9 12345
#    确认已无残留：
ps -ef | grep webcron | grep -v grep

# 7. 启动服务
./run.sh start
```

> 停旧进程是必做步骤：不先停旧进程直接解压覆盖二进制，正在运行的旧进程仍占着端口和内存，新启动会因端口占用失败。

### 验证

```bash
# 检查进程
ps -ef | grep webcron

# 检查端口（默认 8000）
netstat -tlnp | grep 8000

# 查看启动日志
tail -50 info.log
```

访问 `http://服务器地址:8000`，应跳转到登录页。

### 首次登录 admin

1. 用 admin 默认账号登录（`install.sql` 中的初始密码，或已修改的密码）。
2. 登录后会引导到 **MFA 绑定页**，显示二维码。
3. 用 Google Authenticator / 2FAS / Aegis 扫码，输入 6 位验证码（输满自动提交）完成绑定。
4. 后续登录需输入密码 + MFA 验证码。

### 创建普通用户

1. admin 登录后，进入「账号管理 → 新建账号」。
2. 填写用户名（无需邮箱、无需密码，密码由系统随机生成 16 位强密码）。
3. 创建成功后弹出一次性窗口，展示账号 + 密码，点「复制」按钮复制后转交对应人员。
4. **密码仅展示这一次**，关闭窗口后无法再次查看。

---

## 服务管理

使用 `run.sh` 管理服务：

```bash
./run.sh start     # 启动
./run.sh stop      # 停止
./run.sh restart    # 重启
```

若 `run.sh` 不可用，手动重启：

```bash
cd /www/wwwroot/webcron
killall webcron
chmod +x webcron
nohup ./webcron >> info.log 2>&1 &
```

---

## 后续更新（增量部署）

后续版本如仅改动代码，可只替换二进制和变更的模板/静态文件，不覆盖 `conf/app.conf`：

```bash
cd /www/wwwroot/webcron

# 备份
cp webcron webcron.bak.$(date +%Y%m%d%H%M%S)

# 解压并仅覆盖二进制和指定文件
tar -xzf /上传路径/webcron-版本号.tar.gz webcron views/xxx.html

chmod +x webcron
./run.sh restart
```

若涉及数据库结构变更，先执行 `migrations/` 下的增量脚本再重启。

---

## 注意事项

- **`conf/app.conf` 含数据库密码**，包文件和编译产物已在 `.gitignore` 中忽略，不要提交到 Git。
- **生产环境 `jobs.local` 必须为 `false`**，否则定时任务不会自动执行。
- **不要覆盖线上 `conf/app.conf`**，除非确认配置无误；增量部署时排除该文件。
- 每次部署前确认二进制架构为 `ELF 64-bit LSB executable, x86-64`。
- admin 的 MFA 密钥明文存储于 `t_user.mfa_secret`，这是 TOTP 协议要求（服务端需用明文 secret 校验），属行业惯例。
