# Clash Verge Rev / Mihomo 订阅

## 修复的实际问题

旧版默认规则含有 `GEOIP,,DIRECT`，国家/地区字段为空。Mihomo 会报：

```
rules[13] [GEOIP,,DIRECT] error: missing subsequent parameters: GEOIP
```

本次不仅修正新安装的默认模板，也在生成订阅时删除旧配置里这一条明确无效的规则，因此已有 `config/template/template-clash-rule.yaml` 无需覆盖或重置。其他自定义规则不会因这个修复被删除。

## 接口与升级

- `/api/account/clashSubscribe/` 和 `/api/account/clashSubscribeForSb` 增加可选查询参数 `target=clash-verge`。后者仍使用原有 `id` 参数及权限检查。
- 新界面默认申请该目标，得到 `/api/auth/subscribe/<原token>?target=clash-verge`。
- 原 `/api/auth/subscribe/<原token>` 链接继续有效，保留原有路由策略，仅修复空 GEOIP 规则。账号、节点生成、密码、配额、到期信息和订阅鉴权不变。
- 节点仍使用 Mihomo YAML，并不存在独立的“Verge 节点编码格式”。新增的是经过验证的路由配置和迁移逻辑。
- 成功响应返回 `application/yaml`、原 `subscription-userinfo` 和 `profile-update-interval: 12`。无效 token、配置或空节点返回非 200 状态，避免把错误 JSON 当成成功订阅导入。
- 必须先部署配套后端才能获得新配置；仅改前端按钮名称不会改变旧服务器生成的内容。订阅 token 相当于节点凭证，请勿公开分享。

## 新目标的规则顺序

1. 静态 LAN 域名和 IPv4/IPv6 私有/本地地址直连（含 RFC1918、CGNAT、loopback、link-local、ULA）。即使规则集尚未下载，这些静态条目仍在。
2. 维护中的 private 域名规则集与不主动解析域名的 private IP 规则集。
3. 管理员已有的显式规则，保留原相对顺序。
4. CN 域名直连、解析后的 private IP 直连、CN IP 直连。后两条允许解析，覆盖“不在域名列表、但实际解析为大陆/局域网 IP”的情况。
5. 管理员原有的首个 `MATCH`/`FINAL` 及其后续内容；若没有则补 `MATCH,PROXY`。原来位于 catch-all 之后的不可达规则仍然不可达。

新目标设置 `mode: rule`。静态 LAN/private 规则优先于管理员规则；管理员显式规则优先于默认 CN 分流；管理员自定义的最终策略仍然生效，所以有自定义策略时不保证所有剩余流量均走代理。

若原模板的 `rules` 和 `rule-providers` 与旧版默认值相同（忽略空 GEOIP 规则、空白/注释），这两部分会在输出中替换为新默认值，避免继承旧模板对 Google、Apple 等业务的宽泛直连策略。模板附带的自定义 DNS 等其他字段保留。若规则或 providers 被修改，则保留自定义内容并合并；新 providers 避开名称/缓存路径冲突。配置不会回写管理员模板。

模板使用完整 YAML 映射合并，不经过仅含节点字段的结构体回写，因此 `dns`、`nameserver-policy`、自定义 `rule-providers`、`sub-rules` 等字段不会丢失。模板不能重复声明由账号生成的 `proxies`/`proxy-groups`；重复键和多文档 YAML 会明确报错，而非输出不可导入的拼接 YAML。

## 更新来源与实际同步方式

使用 [MetaCubeX/meta-rules-dat](https://github.com/MetaCubeX/meta-rules-dat) 的维护分支：

| 内容 | 路径 | behavior |
| --- | --- | --- |
| LAN/private 域名 | `meta/geo/geosite/private.mrs` | `domain` |
| 大陆域名 | `meta/geo/geosite/cn.mrs` | `domain` |
| LAN/private IP | `meta/geo/geoip/private.mrs` | `ipcidr` |
| 大陆 IP（含 IPv6） | `meta/geo/geoip/cn.mrs` | `ipcidr` |

完整 URL 前缀为 `https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/`。全部使用 `type: http`、`format: mrs`、`interval: 86400` 和独立缓存路径；通过 `PROXY` 组中选定的节点下载，避免依赖大陆直连 GitHub。客户端必须有可用节点才能首次获取这些远程规则。

这是客户端每 24 小时检查更新，上游通常每日构建；不是实时推送。客户端关闭时不会持续更新。订阅节点信息的 12 小时刷新建议与规则集自己的更新周期相互独立，Clash Verge Rev 的本地设置也可能覆盖订阅建议。

已缓存的规则可以继续使用；首次无网络、无法连接代理、上游不可达或规则下载失败时，不能保证完整的 CN 分类。规则集的分类也不是地理或网络质量的绝对保证。

## DNS / TUN / IPv6 边界

不自动启用 TUN、系统代理、DNS 劫持或 IPv6，也不覆盖管理员 DNS 设置。规则只管理实际进入 Mihomo 的流量；Global 模式或客户端覆写可绕过规则模式。规则中覆盖 IPv6 地址不代表强制启用 IPv6。

本地域名仍需要系统/LAN DNS 或应用 mDNS 能正确解析。`.local` 的直连规则不等同于提供 mDNS 解析。用户现有 fake-IP、DNS 和代理节点 DNS 设置会影响真实行为。本测试没有证明“无 DNS 泄漏”，也没有替用户测试真实节点连通性。

## 验证与复现

2026-10-05 使用官方 Mihomo **v1.19.32**（GitHub 最新稳定版，2026-09-30 发布），下载资产 `mihomo-linux-amd64-compatible-v1.19.32.gz` 并核对 SHA256：

```
ba3ce607747a07f948fc35780e108a4a7c7f552a38b9bd4d115f313ebcb89c20
```

- 原始模板复现上面的 `rules[13]` 错误。
- 修正后的旧目标和新目标均通过 `mihomo -t`；旧目标依赖现有 GeoIP 数据库，测试时从官方源准备该数据库。
- 配置 fixture 覆盖 Trojan、VLESS、VMess/WS、Shadowsocks、SOCKS5、Trojan-Go/WS、Hysteria、Hysteria2，使用假凭证。
- Go 单元测试覆盖旧存量模板迁移、自定义 DNS/规则/最终策略保持、provider 类型/路径/刷新周期、名称冲突、IPv4/IPv6 静态兜底、重复键/错误类型/多文档/空节点拒绝。
- Mihomo 运行时验证 20 条路由：IPv4/IPv6 LAN、LAN 域名、CN 域名与 IPv4/IPv6、未列出的域名解析到 LAN/CN IP，以及公网地址/域名的最终代理策略。
- 运行时测试使用本机 HTTP CONNECT 捕获节点和本机 DNS，不向样例目的地址发送外部流量。真实官方 MRS 数据由本机 HTTP 提供，刷新间隔在测试中缩短为 1 秒，验证首次下载和后续轮询。正式输出仍为 86400 秒。

基础检查：

```sh
go test ./util
go build ./...
go vet ./...
go run ./cmd/verify-clash-profile > /tmp/verge.yaml
mihomo -t -d /tmp/mihomo-check -f /tmp/verge.yaml
```

下载 `/tmp/verge.yaml` 中四个 provider 的 URL 到对应 `path` 后：

```sh
python scripts/test-clash-verge-routing.py /path/to/mihomo /tmp/verge.yaml /tmp/mihomo-check
```

Python 脚本需要 PyYAML。CI 会自动准备依赖、核对 Mihomo 下载校验和并运行以上测试。没有对线上数据库、真实用户订阅或用户设备做部署/连接测试。尝试完整 `go test ./...` 时，仓库原有 `testing/service` 被 `core.init()` 过早执行 `flag.Parse()` 阻塞，报 `flag provided but not defined: -test.testlogfile`；该旧套件还依赖外部 gRPC 服务。故不把全部测试宣称为通过。离线回归测试 `go test ./util ./testing/util`、完整编译和 `go vet ./...` 均通过（Go 1.20.14 / 1.26.8）。

官方配置依据：[rule-providers](https://wiki.metacubex.one/config/rule-providers/)、[规则与 no-resolve](https://wiki.metacubex.one/config/rules/)、[Mihomo 稳定版](https://github.com/MetaCubeX/mihomo/releases/tag/v1.19.32)。
