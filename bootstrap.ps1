# chezmoi をこのリポジトリに向けて設定する。
#
#   .\bootstrap.ps1           設定を書いて差分を表示するだけ
#   .\bootstrap.ps1 -Apply    差分の適用まで行う
#
# chezmoi の source dir は既定で %USERPROFILE%\.local\share\chezmoi だが、このリポジトリは
# ghq の管理下に置きたいので sourceDir を設定ファイルに書いて上書きする。
# リポジトリルートの .chezmoiroot が home\ を指すので、実際の source dir は
# <repo>\home に解決される。
[CmdletBinding()]
param(
    [switch]$Apply
)

$ErrorActionPreference = 'Stop'

$repo = $PSScriptRoot
$configDir = Join-Path $env:USERPROFILE '.config\chezmoi'
$config = Join-Path $configDir 'chezmoi.toml'

if (-not (Get-Command chezmoi -ErrorAction SilentlyContinue)) {
    Write-Error @'
chezmoi が見つかりません。先に導入してください:
  winget install --id twpayne.chezmoi --exact
'@
}

if ((Test-Path $config) -and -not (Select-String -Path $config -SimpleMatch $repo -Quiet)) {
    Write-Error "既存の設定が別の場所を指しています: $config`n内容を確認してから手で消すか、書き換えてください"
}

# TOML の文字列内で \ をエスケープする
$sourceDir = $repo -replace '\\', '\\'

New-Item -ItemType Directory -Force -Path $configDir | Out-Null
@"
# bootstrap.ps1 が生成。chezmoi 自身の管理対象ではない。
sourceDir = "$sourceDir"

# .ps1 スクリプトには shebang が書けないのでインタプリタを明示する。
[interpreters.ps1]
command = "pwsh"
args = ["-NoLogo", "-NoProfile"]
"@ | Set-Content -Path $config -Encoding utf8
Write-Host "wrote: $config"
Write-Host ''

chezmoi doctor
Write-Host ''

if ($Apply) {
    chezmoi apply -v
} else {
    chezmoi diff
    Write-Host ''
    Write-Host '適用するには: chezmoi apply -v   (または .\bootstrap.ps1 -Apply)'
}
