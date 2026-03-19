# camellia-oleifera-smart-harvest

油茶智能采摘辅助系统仓库，当前基线围绕三层能力组织：

- `recognition`：成熟果识别、成熟度判断、采摘建议基线
- `decision`：采摘路径与作业顺序决策骨架
- `operations`：地块、树木、进度与效率管理骨架

当前可运行链路保持为：

`clients/operator-console -> services/api-gateway -> services/recognition-api`

## 目录骨架

- `clients/operator-console/`：Nuxt Web + Tauri Desktop 控制台
- `services/recognition-api/`：FastAPI 识别服务
- `services/api-gateway/`：Go 网关，统一入口、鉴权、限流、代理
- `shared/contracts/`：对外契约，当前主文件为 `shared/contracts/openapi.yaml`
- `shared/constants/`：共享常量，当前保留成熟度颜色映射
- `shared/domain/`：共享领域词汇与分层元数据
- `mlops/training/`：训练与评估脚本
- `mlops/data/`：训练数据目录
- `mlops/artifacts/`：模型与指标产物
- `tooling/config/`：配置模板与本地配置
- `tooling/scripts/`：启动、训练、评估、校验脚本
- `tooling/docker/`：容器相关文件
- `tests/`：跨服务集成与性能测试

## 环境准备

- Python `>= 3.11`
- Go `1.25.6`
- Bun

安装依赖：

```bash
uv sync
bun install --cwd clients/operator-console
```

准备本地配置：

```bash
cp tooling/config/recognition.yaml.example tooling/config/recognition.yaml
cp tooling/config/service.yaml.example tooling/config/service.yaml
cp tooling/config/gateway.yaml.example tooling/config/gateway.yaml
```

Windows PowerShell:

```powershell
Copy-Item tooling/config/recognition.yaml.example tooling/config/recognition.yaml
Copy-Item tooling/config/service.yaml.example tooling/config/service.yaml
Copy-Item tooling/config/gateway.yaml.example tooling/config/gateway.yaml
```

## 启动

分服务启动：

```bash
uv run --directory services/recognition-api uvicorn main:app --reload --host 127.0.0.1 --port 8000
go run ./services/api-gateway/cmd/gateway --config tooling/config/gateway.yaml
bun run --cwd clients/operator-console dev -- --host 127.0.0.1 --port 3000
```

桌面端：

```bash
bun run --cwd clients/operator-console tauri:dev
```

脚本入口：

```bash
sh tooling/scripts/app.sh --host 127.0.0.1 --port 8000
sh tooling/scripts/gateway.sh --config tooling/config/gateway.yaml
sh tooling/scripts/frontend.sh --host 127.0.0.1 --port 3000
sh tooling/scripts/stack.sh --app-host 127.0.0.1 --app-port 8000 --gateway-config tooling/config/gateway.yaml --frontend-host 127.0.0.1 --frontend-port 3000
```

## 当前公开接口

当前契约只保留最小可运行基线：

- `GET /healthz`
- `GET /v1/health`
- `POST /v1/recognition/image`
- `GET /v1/recognition/stream`（WebSocket）

契约文件：`shared/contracts/openapi.yaml`

## 训练与评估

```bash
sh tooling/scripts/train.sh --data mlops/data/camellia-oleifera/data.yaml --name camellia_v1
sh tooling/scripts/eval.sh --data mlops/data/camellia-oleifera/data.yaml --exp camellia_v1
```

默认产物：

- 模型：`mlops/artifacts/models/`
- 指标：`mlops/artifacts/metrics/`

## 质量检查

```bash
uv run pytest -q
go test ./services/api-gateway/...
bun run --cwd clients/operator-console typecheck
bun run --cwd clients/operator-console test
bun run --cwd clients/operator-console generate
```

或执行：

```bash
sh tooling/scripts/verify.sh
```

## 当前状态说明

- 识别链路已迁到新目录和新路由。
- `decision` 与 `operations` 当前提供页面和网关目录骨架，后续迭代补充领域实现。
- 旧的 `batch / trace / dashboard` 语义已退出主线，不再作为现行接口和页面。
