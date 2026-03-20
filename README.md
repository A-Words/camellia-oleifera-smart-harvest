# camellia-oleifera-smart-harvest

油茶智能采摘辅助系统仓库，当前基线围绕三层能力组织：

- `recognition`：油茶果目标检测 + 多模态成熟度识别基线
- `decision`：地块级采摘路径、树内作业顺序与人工调整决策中心
- `operations`：地块/树木建档与树级作业单执行闭环

当前可运行链路保持为：

`clients/operator-console -> services/api-gateway -> services/recognition-api`

## 目录骨架

- `clients/operator-console/`：Nuxt Web + Tauri Desktop 控制台
- `services/recognition-api/`：FastAPI 识别服务
- `services/api-gateway/`：Go 网关，统一入口、鉴权、限流、代理
- `shared/contracts/`：对外契约，当前主文件为 `shared/contracts/openapi.yaml`
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

说明：

- `tooling/config/recognition.yaml` 中的相对 `model_path` 现在会优先按配置文件目录解析，再回退到仓库根目录，因此从仓库根目录或 `services/recognition-api` 目录启动都能正确找到模型。
- 若启用多模态成熟度识别，还需提供环境变量 `CAMELLIA_VLM_API_KEY`。

## 启动

分服务启动：

```bash
uv run --directory services/recognition-api uvicorn main:app --reload --host 127.0.0.1 --port 8000
go run ./services/api-gateway/cmd/gateway --config tooling/config/gateway.yaml
bun run --cwd clients/operator-console dev -- --host 127.0.0.1 --port 3000
```

也支持在 `services/recognition-api` 目录内直接运行：

```bash
uv run uvicorn main:app --reload --host 127.0.0.1 --port 8000
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

当前契约公开接口基线：

- `GET /healthz`
- `GET /v1/health`
- `POST /v1/recognition/image`
- `GET /v1/recognition/stream`（WebSocket）
- `POST /v1/decision/recommendation`
- `GET /v1/decision/history`
- `POST /v1/decision/observations`
- `GET /v1/decision/observations`
- `POST /v1/decision/plans`
- `GET /v1/decision/plans`
- `PATCH /v1/decision/plans/{plan_id}`
- `POST /v1/operations/plots`
- `GET /v1/operations/plots`
- `POST /v1/operations/trees`
- `GET /v1/operations/plots/{plot_id}/trees`
- `PATCH /v1/operations/trees/{tree_id}`
- `POST /v1/operations/work-orders`
- `GET /v1/operations/work-orders`
- `PATCH /v1/operations/work-orders/{work_order_id}`

当前 `Detection` 结果在保留目标框和数量字段的同时，支持返回成熟度三态 `ripeness`：

- `harvestable`
- `not_ready`
- `occluded_unclear`

契约文件：`shared/contracts/openapi.yaml`

## 训练与评估

```bash
sh tooling/scripts/train.sh --data mlops/data/camellia-oleifera/data.yaml --name camellia_detection_v1
sh tooling/scripts/eval.sh --data mlops/data/camellia-oleifera/data.yaml --exp camellia_detection_v1
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
- 当前活跃识别契约基于单类 `camellia_oleifera_fruit` 检测，并支持通过外部多模态 VLM 对检测框补全成熟度三态。
- `/recognition` 实时页当前会将流式识别返回的目标框叠加到摄像头画面上，并优先展示 `可采 / 暂不可采 / 遮挡不清`；异步判定尚未完成时显示 `判定中`。
- `/recognition` 当前要求先绑定地块和树木，最新识别快照会以树级 `TreeObservation` 归档到网关数据库。
- `/decision` 当前以地块为上下文生成整块地计划，展示树间路线、树内区域优先级、采摘顺序、跳过建议，并支持人工调整后生成树级作业单。
- 原始训练数据已落位到 `mlops/data/raw/camellia-oleifera-fruit-yolo/`，当前训练基线数据集位于 `mlops/data/camellia-oleifera/`。
- 网关当前会自动初始化 `plots`、`trees`、`tree_observations`、`decision_plans`、`decision_plan_trees`、`work_orders`、`decision_history` 等表，并保留旧树级即时推荐接口作为兼容入口。
- `/operations` 当前已提供地块/树木建档、树状态切换、作业单筛选和状态推进页面。
- 旧的 `batch / trace / dashboard` 语义已退出主线，不再作为现行接口和页面。
