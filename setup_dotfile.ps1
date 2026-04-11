# 実行例: 
#  .\setup_dotfiles.ps1 -Install -Link  (両方実行)
#  .\setup_dotfiles.ps1 -Install        (インストールのみ)
#  .\setup_dotfiles.ps1 -Link           (リンク作成のみ)

param (
    [Alias("I")]
    [switch]$Install, # wingetでアプリをインストールするフラグ
    [Alias("L")]
    [switch]$Link     # シンボリックリンクを作成するフラグ
)

# --- 関数定義 ---

# 1. winget インストール関数
function Install-App {
    param ([string]$AppName, [string]$Id)
    Write-Host "--- Installing $AppName ---" -ForegroundColor Cyan
    if (winget list --id $Id 2>$null) {
        Write-Host "$AppName is already installed." -ForegroundColor Yellow
    } else {
        winget install --id $Id --silent --accept-source-agreements --accept-package-agreements
    }
}

# 2. シンボリックリンク作成関数
function New-DotfileLink {
    param ([string]$SourcePath, [string]$TargetPath)
    
    $targetDir = Split-Path -Parent $TargetPath
    if (-not (Test-Path $targetDir)) {
        New-Item -Path $targetDir -ItemType Directory -Force | Out-Null
    }

    if (Test-Path $TargetPath) {
        Remove-Item $TargetPath -Force
    }

    try {
        New-Item -ItemType SymbolicLink -Path $TargetPath -Value $SourcePath -ErrorAction Stop | Out-Null
        Write-Host "Linked: $SourcePath -> $TargetPath" -ForegroundColor Green
    } catch {
        Write-Host "Error: 管理者権限が必要です ($TargetPath)" -ForegroundColor Red
    }
}

# --- 設定データ ---

$apps = @(
    # Neovim
    @{ Name = "Neovim";          Id = "Neovim.Neovim"              }
    @{ Name = "tree-sitter-cli"; Id = "tree-sitter.tree-sitter-cli"}
    @{ Name = "7-Zip";           Id = "7zip.7zip"                  }
    @{ Name = "ripgrep";         Id = "BurntSushi.ripgrep.MSVC"}
    # Terminal
    @{ Name = "Starship";        Id = "Starship.Starship"          }
    @{ Name = "WezTerm";         Id = "wez.wezterm"                }
    @{ Name = "WTQ";             Id = "windows-terminal-quake"     }
)

$links = @{
    ".config\starship.toml"        = "$env:USERPROFILE\.config\starship.toml"
    "wtq\wtq.jsonc"                = "$env:APPDATA\wtq\wtq.jsonc"
    ".config\wezterm\wezterm.lua"  = "$env:USERPROFILE\.config\wezterm\wezterm.lua"
    ".config\wezterm\keybinds.lua" = "$env:USERPROFILE\.config\wezterm\keybinds.lua"
}

# --- メイン処理 ---

if (-not $Install -and -not $Link) {
    Write-Host "オプション (-Install または -Link) を指定してください。" -ForegroundColor Cyan
    return
}

if ($Install) {
    foreach ($app in $apps) {
        Install-App -AppName $app.Name -Id $app.Id
    }
}

if ($Link) {
    Write-Host "`n--- Creating Symbolic Links ---" -ForegroundColor Cyan
    foreach ($item in $links.GetEnumerator()) {
        $src = Join-Path $PSScriptRoot $item.Key
        $dest = $item.Value
        New-DotfileLink -SourcePath $src -TargetPath $dest
    }

    # $PROFILE
    $profileSrcName = "Microsoft.PowerShell_profile.ps1"
    $profileSrcPath = Join-Path $PSScriptRoot $profileSrcName

    if (Test-Path $profileSrcPath) {
        Write-Host "Processing PowerShell Profile..." -ForegroundColor Yellow
        New-DotfileLink -SourcePath $profileSrcPath -TargetPath $PROFILE
    } else {
        Write-Host "Skip: $profileSrcName not found in dotfiles." -ForegroundColor Gray
    }
}

Write-Host "`nCompleted!" -ForegroundColor Cyan
