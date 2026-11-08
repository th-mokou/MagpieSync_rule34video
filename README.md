# Rule 34 Video 插件

Rule 34 Video 是 MagpieSync 的站点插件，用于读取 Rule 34 Video 账户中的影片列表并将影片加入 MagpieSync 下载队列。

## 功能

- 支持邮箱/用户名与密码登录，也支持 Cookie 登录。
- 同步喜欢的影片、稍后再看和播放列表。
- 解析影片页面的可用直链，支持 1080P、720P、480P 和 360P。
- 下载图片时使用 Rule 34 Video 页面作为 Referer。

## 登录

在 MagpieSync 添加账户时，可选择账号密码登录，或粘贴从已登录浏览器复制的完整 Cookie。Cookie 登录至少需要 `PHPSESSID` 或 `__Host-kt_member_v2`。如果站点触发 DDoS-Guard 或 Cloudflare 验证，可能还需要同一网络出口下取得的验证 Cookie。

Cookie 登录时需提供与生成 Cookie 的浏览器一致的 User-Agent。建议为 MagpieSync 配置可访问站点的代理，尤其是在遇到站点验证或限流时。Cookie、密码及账户凭据属于敏感信息，请勿将其提交到代码仓库或公开日志。Cookie 失效后，请重新登录或更新 Cookie。

## 在 MagpieSync 中启用

本目录是 MagpieSync 的编译期插件，不是独立运行的程序。将插件放入主项目的 `internal/sites/plugins/rule34video` 后，在 MagpieSync 仓库根目录执行：

```sh
go generate ./internal/sites/all
go test ./internal/sites/plugins/rule34video/...
```

默认构建会包含该插件。使用站点白名单构建时，可执行：

```sh
make build SITES="rule34video"
```

## 项目结构

- `register.go`、`login.go`、`source.go`：向 MagpieSync 注册站点能力、处理登录并创建同步与下载解析器。
- `core/`：Rule 34 Video 客户端、页面解析、列表读取和下载链接解析实现。
- `assets/`：插件使用的站点图标。

## 开发测试

在 MagpieSync 主项目根目录运行插件测试：

```sh
go test ./internal/sites/plugins/rule34video/...
```

插件通过 `!no_rule34video && (!explicit_sites || site_rule34video)` 构建约束控制是否编译。主项目的站点清单由 `go generate ./internal/sites/all` 生成，请勿手动编辑生成文件。
