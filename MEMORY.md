# MEMORY

## 项目当前定义

本仓库当前唯一有效的项目目标是：构建一个面向油茶果采收场景的三层软件系统，项目可称为“油茶智能采摘辅助系统”。

仓库名 `camellia-oleifera-smart-harvest` 与该方向一致。仓库内现存的 `README.md`、`docs/prd.md`、`shared/schemas/openapi.yaml` 等文档中，仍有“荔枝识别”“区块链溯源”等旧表述，这些内容应视为待迁移的历史背景，而不是当前产品定义。

本次项目转向采用完全替换策略：旧的“荔枝识别 + 区块链溯源”方案不再是现行目标，新三层系统是后续设计、开发、文档更新和接口重构的唯一基线。

## 三层能力边界

### 第一层：成熟果智能识别

这是整个方案的核心创新层，负责围绕单果识别给出可执行的采摘判断。

当前对业务层承诺的主输出包括：

- 识别油茶果位置
- 判断成熟度
- 区分 `可采 / 暂不可采 / 遮挡不清`
- 给出采摘建议点位

第一层采用“两层输出”策略：

- 底层保留细粒度识别与成熟度建模空间，具体枚举后续设计。
- 业务层统一以 `可采 / 暂不可采 / 遮挡不清` 作为当前主标签输出，并附带建议点位。

支持与预留的接入形态包括：

- 手机端拍照识别
- 手持终端实时视频识别
- 后续预留给固定摄像头、机器人视觉或其他视觉终端调用

### 第二层：采摘路径与作业决策

这一层负责把识别结果转换为实际作业顺序与采摘策略，是“智能化采摘辅助”的核心体现。

核心能力包括：

- 判断哪棵树先采
- 判断树上哪些区域优先采
- 判断哪些果实值得采、哪些应跳过
- 输出对采摘人员更省时间的作业顺序

该层的目标不是再次识别，而是将识别层结果转化为时间、路线、优先级与跳过策略上的决策建议。

### 第三层：作业管理与数据平台

这一层负责形成作业管理闭环，为现场管理、统计分析与后续优化提供数据基础。

核心能力包括：

- 地块信息管理
- 树木档案管理
- 采摘进度统计
- 作业人员效率分析
- 预计产量与采收完成率

该层面向管理和复盘，不承担底层视觉识别，但消费前两层的结果并形成业务数据沉淀。

## 默认技术架构

在没有新的架构决策前，默认保留当前技术骨架 `frontend -> gateway -> app`，作为新项目迁移的实现基础。

- `frontend` 继续承担用户交互、识别结果展示、作业建议展示和管理平台页面。
- `gateway` 继续承担统一入口、鉴权、聚合编排与领域接口承接。
- `app` 继续承担视觉识别、推理与识别结果输出。

需要注意的是，保留的是技术链路与部署骨架，不保留旧的溯源业务语义。后续接口、页面、数据模型和流程应围绕“油茶智能采摘三层系统”重新定义。

## 已确认决策

- 新三层系统是本仓库当前唯一产品目标，不与旧荔枝溯源方案并行双轨维护。
- 第一层采用“两层输出”策略：底层保留细粒度识别空间，业务层主输出固定为 `可采 / 暂不可采 / 遮挡不清` 与采摘建议点位。
- 旧的 `batch / trace / 区块链 / 公开验真 / 溯源看板` 不再是当前产品目标。
- 旧 OpenAPI 和相关实现仍存在于仓库，但仅表示待迁移现状，不再作为新产品方向依据。
- `MEMORY.md` 是仓库级长期记忆与任务入口，不是一次性需求备忘。

## 当前任务

1. 统一项目命名与业务词汇，从“荔枝/溯源”迁移到“油茶/智能采摘”。
2. 重写产品文档基线，优先更新 `README.md`、`docs/prd.md`、`AGENTS.md` 中的项目目标、范围和术语。
3. 重新定义共享契约与数据模型，重点围绕以下三类新域展开：
   - 识别结果域：果实位置、成熟或可采判断、遮挡状态、建议点位。
   - 作业决策域：树优先级、树冠区域优先级、采摘顺序、跳过建议。
   - 作业管理域：地块、树木档案、进度、人员效率、预计产量、完成率。
4. 评估现有 `frontend -> gateway -> app` 链路如何映射到三层新能力，并在未有新决策前继续作为默认技术骨架。
5. 梳理旧接口、旧页面、旧测试中与 `batch / trace / dashboard` 强绑定的部分，标记为待迁移、待重命名或待下线。

## 过时内容提醒

仓库中现有与以下主题相关的内容，默认视为历史实现或待迁移内容：

- 荔枝目标检测项目定位
- 区块链溯源闭环
- 采摘批次 `batch`
- 溯源码 `trace_code`
- 公开验真页面与相关看板
- 围绕旧领域模型编写的 OpenAPI、页面、测试和文档

这些内容可以作为了解现状和复用技术实现的参考，但不能再作为项目范围、命名、接口设计或验收标准的依据。

## 2026-03-19 目录重构落地记录

本次已完成一轮“结构 + 文档”迁移，仓库当前正式基线不再是旧的 `frontend / gateway / app / training / configs / scripts / docker / artifacts` 顶层布局，而是以下结构：

- `clients/operator-console`
- `services/recognition-api`
- `services/api-gateway`
- `shared/contracts`
- `shared/constants`
- `shared/domain`
- `mlops/training`
- `mlops/data`
- `mlops/artifacts`
- `tooling/config`
- `tooling/scripts`
- `tooling/docker`

### 当前有效链路

当前最小可运行链路为：

- `clients/operator-console -> services/api-gateway -> services/recognition-api`

其中：

- 前端默认仍必须走网关，不应直连识别服务。
- 识别服务当前只保留 `system + recognition` 能力。
- 网关当前只保留统一入口、鉴权、限流、日志、识别代理和三层骨架。
- `decision` 与 `operations` 当前允许为骨架页、骨架目录和占位类型，不代表业务已实现。

### 当前路由与接口基线

当前 Web 路由基线固定为：

- `/recognition`
- `/decision`
- `/operations`

首页 `/` 默认跳转到 `/recognition`。

以下旧路由已退出主线，不应再恢复为兼容入口：

- `/batch/create`
- `/trace/*`
- `/dashboard`

当前 HTTP API 基线固定为：

- `GET /healthz`
- `GET /v1/health`
- `POST /v1/recognition/image`
- `GET /v1/recognition/stream`

共享契约当前以 `shared/contracts/openapi.yaml` 为准，不再以 `shared/schemas/openapi.yaml` 为准。

### 当前配置与环境变量基线

配置文件当前固定为：

- `tooling/config/recognition.yaml`
- `tooling/config/recognition.yaml.example`
- `tooling/config/service.yaml`
- `tooling/config/service.yaml.example`
- `tooling/config/gateway.yaml`
- `tooling/config/gateway.yaml.example`

当前环境变量前缀统一为 `CAMELLIA_`，已切换的关键变量包括：

- `CAMELLIA_RECOGNITION_CONFIG`
- `CAMELLIA_SERVICE_CONFIG`
- `CAMELLIA_GATEWAY_CONFIG`

不再保留 `LYCHEE_` 前缀兼容约定。

### 识别与训练相关记忆

- 识别服务内部推理代码已从旧 `inference` 语义收敛到 `services/recognition-api/core/recognition`。
- 共享 OpenAPI 已迁到 `shared/contracts/openapi.yaml`。
- 训练数据示例路径当前以 `mlops/data/camellia-oleifera/data.yaml` 为准。
- 训练与评估默认输出目录当前以 `mlops/artifacts/models` 和 `mlops/artifacts/metrics` 为准。

### 文档与协作约定补充

- `AGENTS.md` 已不再保留“禁止改顶层目录”的旧约束，当前应遵守新三层骨架约束。
- 旧的 `batch / trace / dashboard / reconcile / 区块链溯源` 相关实现已从活跃链路移除，后续如需参考应查 Git 历史，而不是在主线恢复兼容目录。
- `MEMORY.md` 中本节之前的旧结构描述、旧路径描述，应理解为项目转向前的历史上下文；当前执行任务时应以上述新目录和新接口基线为准。

### 本次已完成验证

本次目录重构落地后已完成以下校验：

- `uv run pytest -q`
- `go test ./services/api-gateway/...`
- `bun run --cwd clients/operator-console typecheck`
- `bun run --cwd clients/operator-console test`
- `bun run --cwd clients/operator-console generate`

补充说明：

- `generate` 已成功完成并产出静态结果。
- 生成过程中出现过 Google Fonts 元数据拉取超时告警，以及前端 chunk 偏大告警，但不影响当前产物生成和本轮结构迁移验收。
