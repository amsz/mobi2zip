# Releasing

1. 工作区干净、main 与远端同步，本地跑一遍测试（CI 不再跑测试，靠这一步保证）：

   ```bash
   git pull --rebase origin main
   go vet ./...
   go test ./...
   ```

2. 改 `mygo.json` 的 `version`，并在 `CHANGELOG.md` 顶部加一段该版本的 `## 0.1.1`，写清楚用户能看到的变化（`git log v0.1.0..HEAD` 可参考）。两者合成一个 commit，标题就用版本号，例如 `0.1.1`。

3. 先推 main，等它成功后再打 tag。两个一起推的话，main 被拒时 tag 仍会推上去，结果发布的是一个不在 main 上的提交：

   ```bash
   git push origin main
   git tag v0.1.1
   git push origin v0.1.1
   ```

4. `.github/workflows/release.yml` 会做四件事：

   - 校验 tag 与 `mygo.json` 的 `version` 一致
   - 从 `CHANGELOG.md` 抽出该版本的段落，作为 release notes
   - 建一个 **draft** release
   - 在 windows-latest 上构建便携 exe，重命名为 `mobi2zip-<version>-windows-amd64.exe` 后传到 draft

   两个 job 都通过才能往下走；失败时不要发布，先修。

5. 检查 draft 的附件和说明，然后发布：

   ```bash
   gh release edit v0.1.1 --draft=false --latest
   ```

6. 如果 tag 推到了错误的提交上：先取消运行，再删 tag 和那个 draft，重新来：

   ```bash
   git push origin :refs/tags/v0.1.1
   gh release delete v0.1.1 --yes
   ```

## 为什么是 draft

构建产物先进入 draft，检查无误再点发布——避免一个失败的构建直接成为用户看到的“最新版本”。参考项目 [mygo-cloudmusic](https://github.com/egoist/mygo-cloudmusic) 的 `RELEASING.md` 也是这个流程。
