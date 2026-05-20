# Frontend

Flutter 前端目录。当前服务已先提供一个不依赖 Flutter 工具链的内置 Web 控制台，启动 Go 后端后访问：

```text
http://127.0.0.1:8080/
```

该控制台用于在产品迭代中直接查看已实现能力、数据概况、近期任务和下载目录监听状态，也可直接创建媒体库与下载目录；后续 Flutter Web 初始化后可复用 `/api/dashboard` 作为首页汇总数据源。

计划支持：

- Web
- Desktop
- Mobile

建议初始化命令：

```bash
flutter create .
```

初始化后按 `docs/frontend-design.md` 中的页面结构组织 feature 模块。
