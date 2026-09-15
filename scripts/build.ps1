# 交叉编译单执行产物：windows / linux / darwin × amd64 / arm64。
#
# 前置：先 `pnpm -C web build` 产出 web/dist/（go:embed 读取它，产物本身不入库）。
# 用法：pwsh -NoProfile -File scripts/build.ps1 [-Version v1.0.0] [-OutputDir bin]
param(
    [string]$Version = '',
    [string]$OutputDir = 'bin'
)

$ErrorActionPreference = 'Stop'
Set-Location (Join-Path $PSScriptRoot '..')

if (-not (Test-Path 'web/dist/index.html')) {
    throw 'web/dist/index.html 不存在：请先执行 pnpm -C web build，再交叉编译。'
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
