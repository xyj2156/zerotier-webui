# 交叉编译单执行产物：windows / linux / darwin × amd64 / arm64。
#
# 前置：需装 pnpm（corepack enable 或 npm i -g pnpm）与 Go；脚本会自动构建前端再交叉编译。
# 用法：pwsh -NoProfile -File scripts/build.ps1 [-Version v1.0.0] [-OutputDir bin] [-SkipWeb]
#       -SkipWeb：dist 已是最新时跳过前端构建，只做交叉编译。
param(
    [string]$Version = '',
    [string]$OutputDir = 'bin',
    [switch]$SkipWeb
)

$ErrorActionPreference = 'Stop'
Set-Location (Join-Path $PSScriptRoot '..')

if ($SkipWeb) {
    Write-Host '跳过前端构建（-SkipWeb）'
} else {
    Push-Location web
    try {
        $installArgs = @('install')
        if ($env:CI) { $installArgs += '--frozen-lockfile' }
        & pnpm @installArgs
        if ($LASTEXITCODE -ne 0) { throw "pnpm $($installArgs -join ' ') 失败" }
        & pnpm build
        if ($LASTEXITCODE -ne 0) { throw 'pnpm build 失败' }
    } finally {
        Pop-Location
    }
}

if (-not (Test-Path 'web/dist/index.html')) {
    throw 'web/dist/index.html 不存在且未跳过前端构建：请查看上面的 pnpm 输出。'
}

$module = 'github.com/xyj2156/zerotier-webui'

if (-not $Version) {
    $Version = (& git describe --tags --always --dirty) 2>$null
    if (-not $Version) { $Version = 'dev' }
}
$commit = (& git rev-parse --short HEAD) 2>$null
if (-not $commit) { $commit = 'none' }
$built = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')

$ldflags = "-s -w " +
    "-X $module/internal/buildinfo.Version=$Version " +
    "-X $module/internal/buildinfo.Commit=$commit " +
    "-X $module/internal/buildinfo.Date=$built"

$buildArgs = @('build', '-trimpath', '-ldflags', $ldflags)

$env:CGO_ENABLED = '0'
$targets = @(
    @{ GOOS = 'windows'; GOARCH = 'amd64' },
    @{ GOOS = 'windows'; GOARCH = 'arm64' },
    @{ GOOS = 'linux'; GOARCH = 'amd64' },
    @{ GOOS = 'linux'; GOARCH = 'arm64' },
    @{ GOOS = 'darwin'; GOARCH = 'amd64' },
    @{ GOOS = 'darwin'; GOARCH = 'arm64' }
)

New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null

foreach ($target in $targets) {
    $ext = if ($target.GOOS -eq 'windows') { '.exe' } else { '' }
    $name = "zerotier-webui-$($target.GOOS)-$($target.GOARCH)$ext"
    $env:GOOS = $target.GOOS
    $env:GOARCH = $target.GOARCH
    & go ($buildArgs + @('-o', (Join-Path $OutputDir $name), './cmd/zerotier-webui'))
    if ($LASTEXITCODE -ne 0) { throw "编译 $name 失败" }
    $sizeMB = [math]::Round((Get-Item (Join-Path $OutputDir $name)).Length / 1MB, 1)
    Write-Host ("{0,-40} {1,7} MB" -f $name, $sizeMB)
}

Remove-Item Env:GOOS, Env:GOARCH -ErrorAction SilentlyContinue
Write-Host "`n版本 $Version（commit $commit，构建于 $built）"
