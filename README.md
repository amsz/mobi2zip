# 漫画解包

把漫画电子书里的图片抽出来，重新打包成同名 `.zip` 的 Windows 桌面工具。

## 依赖

| 依赖 | 说明 |
|---|---|
| [calibre](https://calibre-ebook.com/) | 处理 `.mobi` 和 `.pdf` 需要；只转 `.epub` 不需要。设置里可验证路径 |

## 构建

需要 Go 1.27+：

```bash
go tool mygo dev     # 开发运行
go tool mygo build   # 打包到 build/windows-amd64（mobi2zip.exe + NSIS 安装包）
go test ./...
```

产物文件名是英文的 `mobi2zip.exe`；Windows 上显示的名称仍为「漫画解包」（`mygo.json` 的 `windows.displayName`）。

首次构建会拉取 MyGo 依赖；若 `proxy.golang.org` 不通：

```bash
GOPROXY=https://goproxy.cn,direct go tool mygo build
```

## 配置

设置保存在用户数据目录：

```
%APPDATA%\mobi2zip\settings.json
```

字段：

```json
{
  "archive": "D:\\归档",
  "calibrePath": "C:\\Program Files\\Calibre2\\ebook-convert.exe",
  "notify": true
}
```

## 目录结构

```
.
├─ main.go                 # 入口
├─ mygo.json               # 应用元信息（名称、标识、版本）
├─ resources/icon.png      # 应用图标（1024×1024，构建时生成 .ico/.icns）
├─ icon/make_icons.py      # 图标生成脚本（Pillow）
├─ internal/
│  ├─ app/                 # 界面与状态
│  │  ├─ app.go            # 应用状态、post、窗口
│  │  ├─ ui_main.go        # 主窗口：拖放区、列表、底栏
│  │  ├─ ui_settings.go    # 设置窗口
│  │  ├─ ui_kit.go         # 配色与小组件
│  │  ├─ ui_icons.go       # SVG 图标
│  │  ├─ model.go          # 分组/文件模型与扫描
│  │  ├─ convert.go        # 转换队列（2 并发）
│  │  ├─ archive.go        # 归档移动
│  │  ├─ settings.go       # 设置读写
│  │  └─ dev.go            # 调试用的环境变量注入
│  └─ convert/             # 转换内核
│     ├─ convert.go        # 管线与 zip 打包
│     └─ epub.go           # EPUB 解析（spine 页序取图）
└─ docs/screenshot.png
```

## 开发辅助

设置环境变量后启动，可以不用手动拖拽就检查界面：

| 变量 | 作用 |
|---|---|
| `MOBI2ZIP_DEMO=<目录>` | 启动时把该目录加入列表 |
| `MOBI2ZIP_DEMO_FAKE=1` | 把列表里的文件标记成各种结束状态，检查图标 |
| `MOBI2ZIP_DEMO_AUTOSTART=1` | 加入列表后自动开始转换 |

## 发布

改 `mygo.json` 里的 `version`，提交，然后打同名 tag 推上去：

```bash
git tag v0.1.0
git push origin v0.1.0
```

GitHub Actions（[release.yml](.github/workflows/release.yml)）会按顺序：

1. `go vet` + `go test`（不通过就停在这一步，不会发版）
2. 校验 tag 与 `mygo.json` 的 `version` 一致
3. `go tool mygo build` 构建（生成图标、exe、NSIS 安装包）
4. 打一个绿色版 zip
5. 创建 GitHub Release，附上：
   - `mobi2zip-setup-<version>-amd64.exe` —— 安装包
   - `mobi2zip-<version>-windows-amd64.zip` —— 绿色版（解压即用）

> 也可以在 Actions 页面手动触发（`workflow_dispatch`），只构建和上传 artifact，不发版。

