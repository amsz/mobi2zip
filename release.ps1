#Requires -Version 5.1
<#
.SYNOPSIS
    发布一个新版本：改 mygo.json 的 version，提交，推 main，再打 tag 并推送。

.DESCRIPTION
    按 RELEASING.md 的顺序执行。先推 main，等它成功后才创建并推送 tag：
    两个一起推的话，main 被拒时 tag 仍会推上去，发布的就是一个不在 main 上的提交。

    脚本开始前会检查：当前分支是 main、工作区干净、本地没有同名 tag，
    并 fetch/pull 到最新；除非加 -SkipTests，还会跑 go vet 和 go test。
    任何一步失败都会停下，不会把后面的步骤继续做完。

.PARAMETER Version
    新版本号，如 0.1.4；写成 v0.1.4 也可以。

.PARAMETER SkipTests
    跳过本地的 go vet 和 go test。CI 不跑测试，平时不要跳过。

.PARAMETER DryRun
    只做检查并打印 mygo.json 将要发生的改动，不改文件、不提交、不推送。

.EXAMPLE
    ./release.ps1 0.1.4

.EXAMPLE
    ./release.ps1 v0.1.4 -DryRun

.EXAMPLE
    ./release.ps1 0.1.4 -SkipTests
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true, Position = 0)]
    [string]$Version,

    [switch]$SkipTests,

    [switch]$DryRun
)

$ErrorActionPreference = 'Stop'

function Stop-Release {
    param([string]$Message)
    Write-Host ''
    Write-Host "错误：$Message" -ForegroundColor Red
    exit 1
}

# Invoke-Checked runs one step and stops the release when it fails, so a
# failed push never lets the tag go out on its own.
function Invoke-Checked {
    param([string]$What, [scriptblock]$Command)
    & $Command
    if ($LASTEXITCODE -ne 0) {
        Stop-Release "$What 失败（退出码 $LASTEXITCODE）。$AfterFailureHint"
    }
}

# 1. 脚本就放在仓库根目录，后面的相对路径都以它为准。
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location -LiteralPath $root

if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Stop-Release 'PATH 里找不到 git。'
}
if (-not (Test-Path -LiteralPath (Join-Path $root '.git'))) {
    Stop-Release "找不到 $root\.git，release.ps1 要放在仓库根目录。"
}

$jsonPath = Join-Path $root 'mygo.json'
if (-not (Test-Path -LiteralPath $jsonPath -PathType Leaf)) {
    Stop-Release "找不到 $jsonPath。"
}

# 2. 规范版本号：接受 0.1.4 和 v0.1.4，tag 一律写成 v0.1.4。
$Version = $Version.Trim()
if ($Version.StartsWith('v') -or $Version.StartsWith('V')) {
    $Version = $Version.Substring(1)
}
if ($Version -notmatch '^\d+\.\d+\.\d+(?:-[0-9A-Za-z][0-9A-Za-z.-]*)?$') {
    Stop-Release "版本号要形如 0.1.4（可带 -beta.1 这样的后缀），收到 `"$Version`"。"
}
$tag = "v$Version"

# 3. 读现在的版本，新的必须比它大：发布不往回走。
$json = Get-Content -LiteralPath $jsonPath -Raw -Encoding UTF8
if ($json -notmatch '"version"\s*:\s*"([^"]+)"') {
    Stop-Release 'mygo.json 里找不到 "version" 字段。'
}
$current = $Matches[1]

$currentBase = [version](($current -split '-')[0])
$newBase = [version](($Version -split '-')[0])
if ($newBase -lt $currentBase -or ($newBase -eq $currentBase -and $current -notmatch '-')) {
    Stop-Release "mygo.json 现在是 $current，新版本 $Version 必须比它大。"
}

# 4. 本地已有同名 tag 说明这个版本发过了，或者上次发到一半。
$existing = git rev-parse -q --verify "refs/tags/$tag" 2>$null
if ($existing) {
    Stop-Release "本地已经有 tag $tag。确认它没发出去的话，先 git tag -d $tag 再重试。"
}

# 5. 工作区必须干净，否则提交时会把无关改动一起带进版本提交。
$dirty = @(git status --porcelain)
if ($LASTEXITCODE -ne 0) {
    Stop-Release 'git status 失败，先确认仓库状态。'
}
if ($dirty.Count -gt 0) {
    $list = $dirty -join [Environment]::NewLine
    Stop-Release "工作区不干净，先提交或清理这些改动：$([Environment]::NewLine)$list"
}

# 6. 只从 main 发布。
$branch = (git rev-parse --abbrev-ref HEAD).Trim()
if ($branch -ne 'main') {
    Stop-Release "当前在 $branch 分支，发布要在 main 上。"
}

# 7. 和远端对齐，并确认 GitHub 上没有同名 tag。
$AfterFailureHint = '改动和本地提交都还在，没有创建 tag。'
Invoke-Checked 'git fetch' { git fetch --prune --tags origin }
Invoke-Checked 'git pull --rebase' { git pull --rebase origin main }

$AfterFailureHint = '没有提交、没有创建 tag。'
$remoteTag = git ls-remote --tags --refs origin "refs/tags/$tag"
if ($LASTEXITCODE -ne 0) {
    Stop-Release "读不到远端的 tag 列表（可能是没登录 GitHub）。$AfterFailureHint"
}
if ($remoteTag) {
    Stop-Release "GitHub 上已经有 tag $tag，同一个版本不能发两次。"
}

# 8. CI 不跑测试，所以本地跑；跳过时至少提醒一次。
if (-not $SkipTests) {
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Stop-Release 'PATH 里找不到 go，无法跑测试；确实要跳过就加 -SkipTests。'
    }
    Write-Host "==> go vet ./..." -ForegroundColor Cyan
    Invoke-Checked 'go vet ./...' { go vet ./... }
    Write-Host "==> go test ./..." -ForegroundColor Cyan
    Invoke-Checked 'go test ./...' { go test ./... }
} else {
    Write-Host '警告：已跳过 go vet 和 go test。' -ForegroundColor Yellow
}

# 9. 只改 version 那一行的值，其余排版、转义原样保留。
$pattern = [regex]::new('("version"\s*:\s*")[^"]*(")')
$evaluator = [System.Text.RegularExpressions.MatchEvaluator] {
    param($m)
    $m.Groups[1].Value + $Version + $m.Groups[2].Value
}
$updated = $pattern.Replace($json, $evaluator, 1)
if ($updated -eq $json) {
    Stop-Release 'mygo.json 的 version 没有被改动，检查字段写法。'
}
$utf8NoBom = [System.Text.UTF8Encoding]::new($false)

if ($DryRun) {
    # 同目录的临时文件，diff 里才显示干净的两边文件名。
    $tmp = Join-Path $root 'mygo.json.dryrun'
    try {
        [IO.File]::WriteAllText($tmp, $updated, $utf8NoBom)
        Write-Host "预演：mygo.json 的 version 会从 $current 改成 $Version，tag 会是 $tag。" -ForegroundColor Cyan
        git --no-pager diff --no-index -- mygo.json mygo.json.dryrun
    } finally {
        Remove-Item -LiteralPath $tmp -Force -ErrorAction SilentlyContinue
    }
    Write-Host '预演结束：没有改文件、没有提交、没有推送。' -ForegroundColor Cyan
    exit 0
}

[IO.File]::WriteAllText($jsonPath, $updated, $utf8NoBom)
git --no-pager diff -- mygo.json

# 10. 提交、推 main。main 推送失败时提交留在本地，但 tag 还没建。
$AfterFailureHint = "提交已在本地但没推上去，tag 还没创建；修好后执行 git push origin main，再执行 git tag $tag 和 git push origin $tag。"
Invoke-Checked 'git add mygo.json' { git add -- mygo.json }
Invoke-Checked "git commit $Version" { git commit -m $Version }
Invoke-Checked 'git push origin main' { git push origin main }

# 11. main 已经在远端了，才让 tag 出去。
$AfterFailureHint = "main 已推送，但 tag $tag 只在本地；修好后执行 git push origin $tag。"
Invoke-Checked "git tag $tag" { git tag $tag }
Invoke-Checked "git push origin $tag" { git push origin $tag }

Write-Host ''
Write-Host "已推送 $Version（$tag）。" -ForegroundColor Green
Write-Host '接下来：'
Write-Host '  1. 等 GitHub Actions 的 Release 跑完，检查 draft 的附件和说明。'
Write-Host "  2. 发布：gh release edit $tag --draft=false --latest"
Write-Host '     或者在 Releases 页面点 Publish release。'
exit 0
