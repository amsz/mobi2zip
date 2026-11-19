# Releasing

## 一条命令

在仓库根目录：

```powershell
./release.ps1 0.1.4
```

脚本按顺序完成发布：

1. 确认在 `main` 上、工作区干净、本地和 GitHub 上都没有 `v0.1.4` 这个 tag
2. `git fetch` + `git pull --rebase origin main`
3. 本地跑 `go vet ./...` 和 `go test ./...`（CI 不跑测试，靠这一步保证）
4. 把 `mygo.json` 的 `version` 改成 `0.1.4`，只改这一行
5. 提交，标题就是版本号：`0.1.4`
6. 推 `main`，成功后再打 tag `v0.1.4` 并推送

`-DryRun` 只检查并打印 `mygo.json` 的改动，不改文件、不提交、不推送；`-SkipTests` 跳过第 3 步。任何一步失败脚本都会停下，并说明改动和 tag 停在哪一步。

先推 main、后打 tag 是故意的：两个一起推的话，main 被拒时 tag 仍会推上去，结果发布的是一个不在 main 上的提交。

## 之后

`.github/workflows/release.yml` 会做三件事：

- 校验 tag 与 `mygo.json` 的 `version` 一致
- 建一个 **draft** release，说明由 GitHub 按提交和 PR 自动生成
- 在 windows-latest 上构建便携 exe，重命名为 `mobi2zip-<version>-windows-amd64.exe` 后传到 draft

两个 job 都通过才能往下走；失败时不要发布，先修。

检查 draft 的附件和说明，然后发布：

```bash
gh release edit v0.1.4 --draft=false --latest
```

或者在 GitHub 的 Releases 页面点 **Publish release**。

## 如果推错了

如果 tag 推到了错误的提交上：先取消运行，再删 tag 和那个 draft，重新来：

```bash
git push origin :refs/tags/v0.1.4
gh release delete v0.1.4 --yes
```

## 为什么是 draft

构建产物先进入 draft，检查无误再点发布——避免一个失败的构建直接成为用户看到的“最新版本”。
