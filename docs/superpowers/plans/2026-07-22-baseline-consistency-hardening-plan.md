# EchoChat Baseline Consistency Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让当前代码、API 文档、本机生命周期脚本和公网 Compose 配置保持一致，并能通过自动化检查防止再次漂移。

**Architecture:** 新增一个只读的仓库基线检查脚本，直接从 Go 路由、事件常量、media-server 路由和 Compose 声明验证关键事实。生命周期脚本通过统一的健康探测与失败累计返回真实退出码；公网部署通过显式环境变量映射和渲染后的 Compose 预检保证密码、运行模式与 Origin 配置真正进入容器。

**Tech Stack:** Bash 3.2+、Docker Compose v2、Go/Gin/Viper、Node.js/TypeScript、Markdown。

## Global Constraints

- 不读取或扫描 `frontend/node_modules`、`admin/node_modules`、`media-server/node_modules` 的内容。
- 不登录公网服务器，不部署、替换或修改已有 1Panel。
- 不执行 Cloudflare DNS、反向代理、Caddy 或生产前端部署。
- 不改变当前公开端口拓扑，不删除 Docker 数据卷。
- 历史文档保留当时口径，并补充当前实现状态，不把历史记录改写成当前事实。
- 所有行为修复遵循 RED → GREEN → REFACTOR；配置文件通过先失败的静态/渲染测试保护。

---

### Task 1: Repository Baseline Contract

**Files:**
- Create: `scripts/tests/baseline_contract_test.sh`
- Create: `scripts/verify-baseline.sh`

**Interfaces:**
- Consumes: Go router declarations, `backend/go-service/app/constants/meeting.go`, `media-server/src/routes/*.ts`, Markdown links, and `deploy/docker-compose.dev.yml`.
- Produces: `scripts/verify-baseline.sh`, a zero-on-success verifier used by developers and CI.

- [ ] **Step 1: Write the failing contract test**

```bash
#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

test -x "$ROOT_DIR/scripts/verify-baseline.sh"
"$ROOT_DIR/scripts/verify-baseline.sh"

rg -q 'stop_frontend' "$ROOT_DIR/scripts/stop.sh"
rg -q 'stop_admin' "$ROOT_DIR/scripts/stop.sh"
rg -q 'ECHOCHAT_DATABASE_PASSWORD' "$ROOT_DIR/deploy/docker-compose.dev.yml"
rg -q 'ECHOCHAT_REDIS_PASSWORD' "$ROOT_DIR/deploy/docker-compose.dev.yml"
rg -q 'ECHOCHAT_MINIO_SECRET_KEY' "$ROOT_DIR/deploy/docker-compose.dev.yml"
```

- [ ] **Step 2: Run the test to verify RED**

Run: `bash scripts/tests/baseline_contract_test.sh`

Expected: FAIL because `scripts/verify-baseline.sh` does not exist.

- [ ] **Step 3: Add the minimal verifier**

Implement `scripts/verify-baseline.sh` with these checks:

```bash
check_file "docs/api/frontend/group.md"
check_file "docs/api/admin/group.md"
check_markdown_links README.md docs
check_count "backend/go-service/app/im/router.go" 9
check_count "backend/go-service/app/group/router.go" 19
check_count "backend/go-service/app/meeting/router.go" 12
check_meeting_events "backend/go-service/app/constants/meeting.go" 9 8 16
check_media_routes "media-server/src" 10 1 2
check_compose_env_mapping deploy/docker-compose.dev.yml
```

The script must print every failure, keep scanning, and exit `1` when any failure exists. Local Markdown links with anchors are resolved after removing `#fragment`; external links and image data URLs are ignored.

- [ ] **Step 4: Run the test and confirm only known drift remains**

Run: `bash scripts/tests/baseline_contract_test.sh`

Expected: FAIL listing the two missing group documents, stale documented counts, and missing Compose environment mappings.

- [ ] **Step 5: Commit the executable test harness**

```bash
git add scripts/tests/baseline_contract_test.sh scripts/verify-baseline.sh
git commit -m "test: add EchoChat baseline contract"
```

### Task 2: Current API Documentation Baseline

**Files:**
- Create: `docs/api/frontend/group.md`
- Create: `docs/api/admin/group.md`
- Modify: `docs/api/README.md`
- Modify: `README.md`
- Modify: `media-server/README.md`
- Modify: `docs/progress/CURRENT_STATUS.md`
- Test: `scripts/tests/baseline_contract_test.sh`

**Interfaces:**
- Consumes: 19 user group routes, 3 admin group routes, 12 meeting REST routes, 16 unique meeting WS events, and 10 media lifecycle routes.
- Produces: a current-source API index while retaining historical Task/Phase records.

- [ ] **Step 1: Tighten the failing documentation assertions**

Add exact assertions to `scripts/verify-baseline.sh`:

```bash
check_text docs/api/README.md '即时通讯（9 个 API）'
check_text docs/api/README.md '群聊管理（19 个 API）'
check_text docs/api/README.md '会议（12 REST + 16 WS 事件）'
check_text README.md '12 个 REST 接口 + 16 个 WebSocket 事件'
check_text media-server/README.md '当前实现：10 个 `/internal/v1` 生命周期接口'
```

- [ ] **Step 2: Run the verifier to verify RED**

Run: `bash scripts/verify-baseline.sh`

Expected: FAIL on missing group files and all stale count assertions.

- [ ] **Step 3: Write current group API documents**

`docs/api/frontend/group.md` must list each method and path from `app/group/router.go`, grouped as group management (5), members (7), owner/global mute (2), join requests (3), and invitation decisions (2). Request bodies must use the exact DTO fields: `name`, `member_ids`, `avatar`, `notice`, `is_searchable`, `user_ids`, `nickname`, `role`, `is_muted`, `new_owner_id`, `message`, and `action`.

`docs/api/admin/group.md` must document:

```text
GET    /api/v1/admin/groups?page=1&page_size=20&keyword=
GET    /api/v1/admin/groups/:id
DELETE /api/v1/admin/groups/:id
```

- [ ] **Step 4: Correct current summaries without rewriting history**

Update current navigation/count statements to IM 9, user group 19, meeting 12 REST plus 16 unique WS events (9 C→S, 8 S→C, with `meeting.member.state.changed` overlapping), and media 10 lifecycle plus `/internal/info`, `/healthz`, `/readyz`. Historical nine-route Task records remain and gain a dated current-status note.

- [ ] **Step 5: Run documentation verification**

Run: `bash scripts/verify-baseline.sh`

Expected: document and link checks PASS; lifecycle and Compose assertions may still fail.

- [ ] **Step 6: Commit documentation consistency**

```bash
git add README.md docs/api docs/progress/CURRENT_STATUS.md media-server/README.md scripts/verify-baseline.sh
git commit -m "docs: align API baseline with current routes"
```

### Task 3: Symmetric Start and Stop Lifecycle

**Files:**
- Create: `scripts/tests/lifecycle_contract_test.sh`
- Modify: `scripts/start.sh`
- Modify: `scripts/stop.sh`
- Modify: `docs/guides/startup-and-deployment.md`

**Interfaces:**
- Consumes: local PID files in `.run`, ports 8085/3300/5173/3100, and Compose profile `public`.
- Produces: symmetric `app`, `full`, and `--all` modes with nonzero exit on partial startup failure.

- [ ] **Step 1: Write failing lifecycle tests**

The test extracts the `main()` case bodies and asserts:

```bash
assert_case_contains full 'stop_frontend'
assert_case_contains full 'stop_admin'
assert_case_contains full 'stop_full'
assert_case_contains --all 'stop_full'
assert_case_not_contains --all 'stop_docker'
assert_start_has_failure_counter
assert_start_does_not_swallow_failures
```

- [ ] **Step 2: Verify RED**

Run: `bash scripts/tests/lifecycle_contract_test.sh`

Expected: FAIL because `full` leaves local front/admin running, `--all` leaves full-stack containers running, and startup failures are swallowed.

- [ ] **Step 3: Implement symmetric stop semantics**

Use these exact mode semantics:

```text
app/default: local backend + media + frontend + admin; keep middleware containers
full: local frontend + local admin + all Compose services; preserve volumes
--all: every local app + every Compose service including public profile; preserve volumes
docker: middleware containers only
```

Every stop function returns a status while later stops continue. `full` invokes `stop_frontend`, `stop_admin`, then `stop_full`; `--all` invokes all four local stops and `stop_full`.

- [ ] **Step 4: Make startup failure explicit**

Introduce a helper with this contract:

```bash
run_start_step() {
  local label="$1"; shift
  if "$@"; then return 0; fi
  log_err "$label 启动失败"
  START_FAILURES=$((START_FAILURES + 1))
  return 0
}
```

After all requested services are attempted, print the summary only when `START_FAILURES == 0`; otherwise print the number of failures and return `1`.

- [ ] **Step 5: Run lifecycle tests**

Run: `bash scripts/tests/lifecycle_contract_test.sh && bash -n scripts/start.sh scripts/stop.sh`

Expected: PASS.

- [ ] **Step 6: Update the startup guide and commit**

```bash
git add scripts/start.sh scripts/stop.sh scripts/tests/lifecycle_contract_test.sh docs/guides/startup-and-deployment.md
git commit -m "fix: make local lifecycle modes symmetric"
```

### Task 4: Readiness and Status Reporting

**Files:**
- Create: `scripts/tests/health_contract_test.sh`
- Modify: `scripts/start.sh`
- Modify: `scripts/status.sh`

**Interfaces:**
- Consumes: HTTP URLs, PID files, listening ports, and Docker `.State.Health.Status`.
- Produces: `wait_for_http(name, url, pid, timeout)` and status output distinguishing process, HTTP readiness, container health, and stale PID files.

- [ ] **Step 1: Write failing health contract tests**

Assert the scripts contain and use:

```text
wait_for_http
http://127.0.0.1:8085/health
http://127.0.0.1:3300/readyz
https://127.0.0.1:5173
http://127.0.0.1:3100
.State.Health.Status
STALE-PID
NOT-READY
```

The test must use a temporary Python HTTP server to prove a reachable URL passes and an unused port times out with nonzero status.

- [ ] **Step 2: Verify RED**

Run: `bash scripts/tests/health_contract_test.sh`

Expected: FAIL because the readiness helper and health-aware status states do not exist.

- [ ] **Step 3: Add bounded HTTP readiness**

Implement `wait_for_http()` using `curl --insecure --fail --silent --show-error --max-time 2`, polling once per second until timeout. It must also fail early if the tracked PID exits. `spawn_bg()` accepts `health_url` and `timeout`, and prints the last 20 log lines on failure.

- [ ] **Step 4: Report truthful status**

For local apps, report these independent states:

```text
RUNNING      listener and HTTP ready
NOT-READY    listener exists but HTTP check fails
STALE-PID    pidfile exists but PID is dead and no listener is present
STOPPED      no live PID/listener
```

For containers, report `RUNNING/healthy`, `RUNNING/starting`, `RUNNING/no-healthcheck`, `unhealthy`, stopped state, or `NOT-CREATED`.

- [ ] **Step 5: Run health tests and syntax checks**

Run: `bash scripts/tests/health_contract_test.sh && bash -n scripts/start.sh scripts/status.sh`

Expected: PASS.

- [ ] **Step 6: Commit readiness reporting**

```bash
git add scripts/start.sh scripts/status.sh scripts/tests/health_contract_test.sh
git commit -m "fix: verify service readiness and health"
```

### Task 5: Public Compose Configuration Propagation

**Files:**
- Create: `scripts/tests/public_compose_contract_test.sh`
- Modify: `deploy/docker-compose.dev.yml`
- Modify: `deploy/.env.public.example`
- Modify: `scripts/deploy-public.sh`
- Modify: `docs/guides/startup-and-deployment.md`

**Interfaces:**
- Consumes: `POSTGRES_*`, `REDIS_PASSWORD`, `MINIO_*`, `JWT_SECRET`, `MEDIA_INTERNAL_TOKEN`, `GO_SERVER_MODE`, and `WS_ALLOWED_ORIGINS`.
- Produces: explicit `ECHOCHAT_*` environment values in `go-service`, optional Redis authentication locally, mandatory strong Redis authentication publicly, and a rendered Compose preflight.

- [ ] **Step 1: Write a failing Compose contract test**

Create a temporary env file with sentinel values and run:

```bash
docker compose --env-file "$env_file" -f deploy/docker-compose.dev.yml config >"$rendered"
```

Assert the rendered `go-service.environment` includes the sentinel values for database user/password/dbname, Redis password, MinIO access/secret/bucket, server mode `release`, WS origins, JWT secret, and media token. Assert the Redis command and healthcheck authenticate when a password exists.

- [ ] **Step 2: Verify RED**

Run: `bash scripts/tests/public_compose_contract_test.sh`

Expected: FAIL because current Compose only forwards JWT and media token.

- [ ] **Step 3: Map public values into Go and Redis**

Add these `go-service.environment` keys:

```yaml
ECHOCHAT_SERVER_MODE: ${GO_SERVER_MODE:-debug}
ECHOCHAT_SERVER_WS_ALLOWED_ORIGINS: ${WS_ALLOWED_ORIGINS:-}
ECHOCHAT_DATABASE_USER: ${POSTGRES_USER:-echochat}
ECHOCHAT_DATABASE_PASSWORD: ${POSTGRES_PASSWORD:-echochat_dev_2026}
ECHOCHAT_DATABASE_DBNAME: ${POSTGRES_DB:-echochat}
ECHOCHAT_REDIS_PASSWORD: ${REDIS_PASSWORD:-}
ECHOCHAT_MINIO_ACCESS_KEY: ${MINIO_ROOT_USER:-echochat}
ECHOCHAT_MINIO_SECRET_KEY: ${MINIO_ROOT_PASSWORD:-echochat123456}
ECHOCHAT_MINIO_BUCKET: ${MINIO_BUCKET:-echochat}
```

Pass `REDIS_PASSWORD` to the Redis container. Its shell command conditionally adds `--requirepass`, and its healthcheck uses `REDISCLI_AUTH` only when non-empty.

- [ ] **Step 4: Complete the public environment template**

Add:

```dotenv
GO_SERVER_MODE=release
WS_ALLOWED_ORIGINS=_REPLACE_WITH_HTTPS_ORIGINS_
MINIO_BUCKET=echochat
```

The deploy validator treats Redis password and HTTPS origins as required in `DEPLOY_MODE=public`, rejects every `_REPLACE_WITH_*_` value, and never prints secret values.

- [ ] **Step 5: Preflight the rendered Compose model**

Before build/up, run:

```bash
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" --profile public config --quiet
```

If this fails, abort before any container mutation. Continue using the existing 1Panel/Docker installation; do not install Docker or 1Panel.

- [ ] **Step 6: Run Compose and baseline tests**

Run: `bash scripts/tests/public_compose_contract_test.sh && bash scripts/verify-baseline.sh && bash -n scripts/deploy-public.sh`

Expected: PASS. If Docker is unavailable, the test prints `SKIP` only for the rendered-runtime assertion; static mapping assertions still run and must pass.

- [ ] **Step 7: Commit configuration propagation**

```bash
git add deploy/docker-compose.dev.yml deploy/.env.public.example scripts/deploy-public.sh scripts/tests/public_compose_contract_test.sh scripts/verify-baseline.sh docs/guides/startup-and-deployment.md
git commit -m "fix: propagate public configuration into compose"
```

### Task 6: Full Regression and Handoff

**Files:**
- Modify: `docs/progress/CURRENT_STATUS.md`

**Interfaces:**
- Consumes: every artifact from Tasks 1-5.
- Produces: a verified branch handoff with explicit environment limitations.

- [ ] **Step 1: Run shell and contract verification**

```bash
bash -n scripts/*.sh scripts/tests/*.sh
bash scripts/tests/baseline_contract_test.sh
bash scripts/tests/lifecycle_contract_test.sh
bash scripts/tests/health_contract_test.sh
bash scripts/tests/public_compose_contract_test.sh
git diff --check
```

Expected: PASS with no whitespace errors.

- [ ] **Step 2: Run application baselines without scanning generated folders**

```bash
(cd backend/go-service && GOCACHE="$(pwd)/.gocache" go test ./... && GOCACHE="$(pwd)/.gocache" go vet ./...)
(cd media-server && npm test -- --run && npm run build)
(cd frontend && npm run build:h5)
(cd admin && npm run build)
```

Expected: all available dependency-backed checks PASS. Missing ignored `node_modules` in this worktree is reported as an environment limitation, not a product regression; do not traverse it.

- [ ] **Step 3: Update current status with evidence**

Add a dated 2026-07-22 baseline section containing exact commands, pass counts, skipped checks, and the boundary that public server/1Panel/Cloudflare were not changed.

- [ ] **Step 4: Inspect final scope**

Run: `git status --short && git log --oneline --decorate -8`

Expected: only intentional changes are present on `codex/baseline-consistency-hardening`.

- [ ] **Step 5: Commit the verified status record**

```bash
git add docs/progress/CURRENT_STATUS.md
git commit -m "docs: record verified consistency baseline"
```
