# dotfiles

## セットアップ

```sh
brew install chezmoi                            # macOS
winget install --id twpayne.chezmoi --exact     # Windows

git clone https://github.com/b1017034/dotfiles
cd dotfiles
sh bootstrap.sh --apply       # Windows: .\bootstrap.ps1 -Apply
```

## 構成

- `home/` — chezmoi の source dir。
  - `.chezmoiignore` — OS 別の出し分け
  - `.chezmoidata/packages.toml` — パッケージ定義。
  - `.chezmoiexternal.toml` — 外部リポジトリの取得 (tmux の tpm)
  - `.chezmoitemplates/herdr-config.toml` — herdr 設定の本体。macOS は `~/.config/herdr/`、
    Windows は `%APPDATA%\herdr\` に配置される
- `main.go` / `internal/` — TUI

## 管理対象の追加

- 設定ファイル: `chezmoi add <path>`。片方の OS 専用なら `.chezmoiignore` に条件を足す
- パッケージ: `packages.toml` に `[[packages]]` を足し、`brew` / `winget` の ID を書く。
  どちらにも無いものは `[packages.windows]` (または darwin) の `script` と `check` で入れる
- macOS の OS 設定: `.chezmoiscripts/run_onchange_after_30-macos-defaults.sh.tmpl` に `defaults` を足す

## ビルド

```sh
make build      # dist/ に darwin-arm64 と windows-amd64
make local      # ホスト向けに ./dotfiles
```
