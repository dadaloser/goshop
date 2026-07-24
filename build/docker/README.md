# goshop Docker Build

所有服务统一使用 `build/docker/Dockerfile` 构建，通过 build args 指定入口、配置和端口。
Jenkins 可以使用 `build/docker/Jenkinsfile`，通过 `SERVICE` 参数选择要构建的服务。

示例：

```bash
docker build \
  -f build/docker/Dockerfile \
  --build-arg SERVICE_NAME=goshop-user-srv \
  --build-arg CMD_PATH=./cmd/user \
  --build-arg CONFIG_PATH=configs/user/srv.yaml \
  --build-arg GRPC_PORT=8014 \
  --build-arg HTTP_PORT=8054 \
  --build-arg VERSION=local \
  --build-arg VCS_REF=$(git rev-parse --short=12 HEAD) \
  -t goshop-user-srv:local .
```

服务参数：

| Service | CMD_PATH | CONFIG_PATH | GRPC_PORT | HTTP_PORT |
| --- | --- | --- | --- | --- |
| admin | `./cmd/admin` | `configs/admin/admin.yaml` | `8010` | `8050` |
| api | `./cmd/api` | `configs/api/api.yaml` | `8009` | `8049` |
| goods | `./cmd/goods` | `configs/goods/srv.yaml` | `8010` | `8051` |
| inventory | `./cmd/inventory` | `configs/inventory/srv.yaml` | `8012` | `8052` |
| order | `./cmd/order` | `configs/order/srv.yaml` | `8013` | `8053` |
| review | `./cmd/review` | `configs/review/srv.yaml` | `8015` | `8055` |
| user | `./cmd/user` | `configs/user/srv.yaml` | `8014` | `8054` |

运行时可通过 `APP_CONFIG` 覆盖配置文件路径。

Jenkins 参数：

| 参数 | 说明 |
| --- | --- |
| `SERVICE` | `admin`、`api`、`goods`、`inventory`、`order`、`review`、`user` |
| `version` | 镜像 tag 前缀，例如 `v1.0.0` |
| `branch` | 镜像 tag 后缀，例如 `main`、`dev`、提交短 SHA |
| `RUN_TESTS` | 是否在构建镜像前运行对应服务测试 |
| `PUSH_IMAGE` | 是否推送镜像；本地验证 Jenkins 构建时可关闭 |

如果你习惯一个服务一个 Jenkins Job，也可以继续使用 `build/docker/<service>/Jenkinsfile`。

所有 Jenkins agent 必须预装固定版本的 `govulncheck`、`gitleaks`、`syft`、`trivy` 和 `cosign`。流水线在推送前执行格式、vet、lint、漏洞、secret、migration、配置及 protobuf 门禁，并对本地镜像生成 SPDX SBOM、阻断未修复的 HIGH/CRITICAL 漏洞。推送后以 registry digest 签名并附加 SBOM attestation；制品归档包含 SBOM 和 digest。

Jenkins credentials：`harbor-goshop`、Secret file `cosign-goshop-key`、Secret text `cosign-goshop-password`。生产部署只接受 `IMAGE@sha256:DIGEST`，并在部署前执行 Cosign 验证。
