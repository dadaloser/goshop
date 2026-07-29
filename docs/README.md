# GoShop 文档索引

本文档目录按“先运行、后开发、再运维”的顺序组织。所有命令都应在仓库根目录执行。

## 新手入口

1. [新手使用教程](getting-started.md)：环境准备、配置、MySQL 初始化、服务启动、验证与维护。
2. [数据库初始化](database-initialization.md)：空 MySQL 实例的受控建库、迁移和 RBAC 种子导入。
3. [生产配置](production-configuration.md)：配置来源、密钥、TLS、可观测性和上线前检查。
4. [Proto 生成](proto-generation.md)：修改接口定义后的生成与校验流程。
5. [依赖韧性](dependency-resilience.md)：gRPC、Redis 与 MySQL 的超时、隔离、熔断和指标。

## 架构、研发与业务

- [架构与服务边界](architecture.md)
- [开发规范](development-standards.md)
- [安全与访问控制](security-and-access-control.md)
- [业务规则与可观测性](business-and-observability.md)
- [生产基线](production-baseline.md)

## 运维与发布

- [运维、发布与历史演练](operations-and-release.md)

## 已清理内容

已将架构草案、研发规则、运行手册和带日期的演练记录合并为上述主题文档。`gmicro-.md` 是内容截断的旧进度表，已移除。
