# dotfiles

[chezmoi](https://www.chezmoi.io/) で設定ファイルを配置し、chezmoi 操作の
フロントエンドとして Go 製 TUI (`./dotfiles`) を乗せている。
パッケージの導入は `chezmoi apply` 時に `.chezmoiscripts` が行う。

## セットアップ

```sh
brew install chezmoi                            # macOS
winget install --id twpayne.chezmoi --exact     # Windows

git clone https://github.com/b1017034/dotfiles
cd dotfiles
sh bootstrap.sh --apply       # Windows: .\bootstrap.ps1 -Apply
```

bootstrap は `~/.config/chezmoi/chezmoi.toml` に sourceDir を書いて、
このリポジトリの `home/` を chezmoi の source dir にする。
apply 時に `.chezmoiscripts` が走り、パッケージも導入される。

## 構成

- `home/` — chezmoi の source dir。`$HOME` に配置されるのはここだけ
  - `.chezmoiignore` — OS 別の出し分け
  - `.chezmoidata/packages.toml` — パッケージ定義。導入スクリプトが読む
  - `.chezmoitemplates/herdr-config.toml` — herdr 設定の本体。macOS は `~/.config/herdr/`、
    Windows は `%APPDATA%\herdr\` に配置される
- `main.go` / `internal/` — TUI

## 管理対象の追加

- 設定ファイル: `chezmoi add <path>`。片方の OS 専用なら `.chezmoiignore` に条件を足す
- パッケージ: `packages.toml` に `[[packages]]` を足し、`brew` / `winget` の ID を書く。
  どちらにも無いものは `[packages.windows]` (または darwin) の `script` と `check` で入れる

## ビルド

```sh
make build      # dist/ に darwin-arm64 と windows-amd64
make local      # ホスト向けに ./dotfiles
```
