# AGENTS.md

## 1. 文档定位

- 本文档面向仓库内执行任务的自动化 Agent。
- 目标是先对齐当前产品基线，再执行改动，再完成校验。
- 当前长期记忆与迁移方向以 `MEMORY.md` 为准。

## 2. 当前项目基线

- 仓库当前唯一有效目标：油茶智能采摘辅助系统。
- 三层能力基线：
  - `recognition`：单果检测 + 多模态成熟度识别基线，当前主线输出油茶果目标框、数量与成熟度三态
  - `decision`：采摘路径与作业顺序决策，当前已落地 plot/tree observation、plot 级计划与人工调整
  - `operations`：作业管理与数据平台，当前已落地 plot/tree 档案与树级作业单状态流转
- 默认调用链：
  - `clients/operator-console -> services/api-gateway -> services/recognition-api`

## 3. 目录约定

- 顶层骨架固定为：
  - `clients/`
  - `services/`
  - `shared/`
  - `mlops/`
  - `tooling/`
  - `docs/`
  - `tests/`
- 当前关键目录：
  - `clients/operator-console/`：Nuxt Web + Tauri Desktop
  - `services/recognition-api/`：FastAPI 识别服务
  - `services/api-gateway/`：Go 网关
  - `shared/contracts/openapi.yaml`：对外契约
  - `shared/domain/`：共享领域词汇
  - `tooling/config/`：配置模板与本地配置
  - `tooling/scripts/`：脚本入口
  - `mlops/training/`：训练与评估脚本
  - `mlops/artifacts/`：模型与指标产物

## 4. 执行工作流

### 4.1 安装依赖

- `uv sync`
- `bun install --cwd clients/operator-console`

### 4.2 准备配置

- `tooling/config/recognition.yaml.example -> tooling/config/recognition.yaml`
- `tooling/config/service.yaml.example -> tooling/config/service.yaml`
- `tooling/config/gateway.yaml.example -> tooling/config/gateway.yaml`
- `tooling/config/recognition.yaml` 中相对 `model_path` 的解析顺序固定为：
  - 绝对路径原样使用
  - 相对配置文件目录
  - 相对仓库根目录
- 若启用多模态成熟度识别，还需提供：
  - 环境变量 `CAMELLIA_VLM_API_KEY`

### 4.3 启动服务

- recognition-api:
  - `uv run --directory services/recognition-api uvicorn main:app --reload --host 127.0.0.1 --port 8000`
  - 或在 `services/recognition-api` 目录中执行：`uv run uvicorn main:app --reload --host 127.0.0.1 --port 8000`
- api-gateway:
  - `go run ./services/api-gateway/cmd/gateway --config tooling/config/gateway.yaml`
- operator-console:
  - `bun run --cwd clients/operator-console dev -- --host 127.0.0.1 --port 3000`
- desktop:
  - `bun run --cwd clients/operator-console tauri:dev`

### 4.4 训练与评估

- 训练：
  - `uv run python mlops/training/train.py --data mlops/data/camellia-oleifera/data.yaml --model yolo26n.pt --project mlops/artifacts/models --name camellia_detection_v1`
- 评估：
  - `uv run python mlops/training/eval.py --model mlops/artifacts/models/camellia_detection_v1/weights/best.pt --data mlops/data/camellia-oleifera/data.yaml --output mlops/artifacts/metrics/camellia_detection_v1-eval_metrics.json`

### 4.5 校验与测试

- Python：
  - `uv run pytest -q`
- Go：
  - `go test ./services/api-gateway/...`
- Frontend：
  - `bun run --cwd clients/operator-console typecheck`
  - `bun run --cwd clients/operator-console test`
  - `bun run --cwd clients/operator-console generate`

## 5. 改动规则

- 优先最小改动，只改与当前任务直接相关文件。
- 顶层骨架与服务名称默认保持：
  - `clients/operator-console`
  - `services/recognition-api`
  - `services/api-gateway`
- 当前版本不再保留旧的 `batch / trace / dashboard` 作为现行业务命名。
- 修改路径、命令、配置或接口时，必须同步检查：
  - `README.md`
  - `AGENTS.md`
  - `docs/prd.md`
  - `tooling/config/*.yaml.example`
  - `shared/contracts/openapi.yaml`
- 每次完成一次实际操作或一轮有效改动后，都应将新的长期记忆追加到 `MEMORY.md` 末尾。
- `MEMORY.md` 采用追加式维护：
  - 只允许新增记忆，不允许修改、覆盖或删除旧有记忆。
  - 如旧记忆已过时，只能在末尾追加“新决策 / 新基线 / 覆盖说明”，不能回改历史内容。
- 前端禁止直连 `services/recognition-api`，默认必须经过 `services/api-gateway`。
- 新增共享字段或共享枚举时，优先放入 `shared/domain/` 或 `shared/constants/`，再同步到各服务。
- 任何行为改动，至少运行一次相关测试；若未执行，必须说明原因与风险。

## 6. 当前接口基线

- 当前对外最小基线只要求以下接口可运行：
  - `GET /healthz`
  - `GET /v1/health`
  - `POST /v1/recognition/image`
  - `GET /v1/recognition/stream`
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
- 当前识别结果字段主线为：
  - `Detection`: `bbox`、`class_name`、`confidence`、`track_id`、`ripeness`
- 当前决策结果字段主线为：
  - `DecisionSnapshotRequest`: `frame_index`、`timestamp_ms`、`frame_width`、`frame_height`、`detections`
  - `DecisionRecommendationResponse`: `decision_id`、`created_at`、`summary`、`zone_priorities`、`pick_sequence`、`skip_items`
- 当前完整作业闭环字段主线还包括：
  - `TreeObservation`: `observation_id`、`tree_id`、`captured_at`、`frame_index`、`frame_width`、`frame_height`、`detections`
  - `DecisionPlan`: `plan_id`、`plot_id`、`generated_at`、`tree_sequence`、`tree_recommendations`、`manual_override`
  - `WorkOrder`: `work_order_id`、`plot_id`、`tree_id`、`plan_id`、`status`、`zone_priorities`、`pick_sequence`、`skip_items`

## 7. 提交前检查

- 命令与路径示例可运行，且参数与脚本实现一致。
- 未引入硬编码绝对路径。
- `tooling/config/*.yaml.example` 仍可直接复制使用。
- `shared/contracts/openapi.yaml` 与 `services/api-gateway`、`services/recognition-api`、`clients/operator-console` 字段一致。
- 前端调用链未偏离：`operator-console -> api-gateway -> recognition-api`
- `/recognition` 必须先绑定 plot/tree，再归档 observation。
- `/decision` 必须以 plot 为上下文生成计划，不应绕过网关重复直连识别服务。
- `/operations` 必须能看到 plot/tree 档案与 work order 状态流。
- 前端质量门禁通过：`typecheck`、`test`、`generate`
- 测试执行情况已记录。

## 8. 默认策略

- 不确定时优先保持识别链路稳定。
- 涉及结构性改动时，先说明影响范围，再实施迁移。
- 若发现旧荔枝/溯源语义残留，应按当前三层系统基线清理或重命名，而不是继续扩展旧模型。
