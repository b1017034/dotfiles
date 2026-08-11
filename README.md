# dotfiles

設定ファイルの配置は [chezmoi](https://www.chezmoi.io/) が行う。
その上に、パッケージの導入と chezmoi の操作をまとめた TUI (`./dotfiles`) を乗せている。
TUI は Go 製なのでクロスコンパイルして Windows 機にバイナリを持っていける。

## 構成

```
.
├── .chezmoiroot                        "home" — chezmoi の source dir をここに向ける
├── bootstrap.sh / bootstrap.ps1        chezmoi の設定を書いて apply する
├── home/                               ← chezmoi の source dir
│   ├── .chezmoiignore                  OS 別に配置しないものを除外 (テンプレート)
│   ├── .chezmoidata/packages.toml      パッケージ定義
│   ├── .chezmoiscripts/                apply 時に走るスクリプト
│   ├── dot_config/
│   │   ├── starship.toml               → ~/.config/starship.toml           (共通)
│   │   ├── herdr/config.toml           → ~/.config/herdr/config.toml       (macOS)
│   │   └── powershell/profile.ps1      → ~/.config/powershell/profile.ps1  (Windows)
│   └── AppData/Roaming/wtq/wtq.jsonc   → %APPDATA%\wtq\wtq.jsonc           (Windows)
├── main.go
└── internal/
    ├── chezmoi/    chezmoi コマンドのラッパー
    ├── packages/   packages.toml の読み込みと brew / winget
    ├── run/        外部コマンドの実行とログ
    └── ui/         Bubble Tea の管理画面
```

`.chezmoiroot` が `home` を指しているので、`README.md` や `main.go` や `internal/` は
chezmoi から見えない。`$HOME` に配置されるのは `home/` 配下だけ。

OS の出し分けは `home/.chezmoiignore` のテンプレートで行う。
macOS では `AppData` と `.config/powershell` が、Windows では `.config/herdr` が除外される。

## セットアップ

chezmoi を入れる。

```sh
brew install chezmoi                              # macOS
winget install --id twpayne.chezmoi --exact       # Windows
```

このリポジトリを clone して bootstrap を走らせる。

```sh
git clone https://github.com/b1017034/dotfiles
cd dotfiles
sh bootstrap.sh              # 設定を書いて chezmoi diff を表示
sh bootstrap.sh --apply      # 適用まで行う
```

```powershell
.\bootstrap.ps1
.\bootstrap.ps1 -Apply
```

bootstrap がやるのは 1 つだけで、`~/.config/chezmoi/chezmoi.toml` に
`sourceDir` を書いてこのリポジトリを chezmoi の source dir に指定する。
chezmoi の既定は `~/.local/share/chezmoi` だが、このリポジトリは ghq の管理下に
置きたいので上書きしている。`.ps1` スクリプト用のインタプリタ設定も同時に書く。

`sh bootstrap.sh` のかわりに `./bootstrap.sh` で実行したい場合は `chmod +x bootstrap.sh` する。

## 日常の操作

### 設定ファイル

```sh
chezmoi edit ~/.config/starship.toml   # ソースを編集
chezmoi diff                           # 差分を確認
chezmoi apply -v                       # 適用

chezmoi re-add                         # ターゲットを直接いじったときソースへ取り込む
chezmoi cd                             # リポジトリへ移動
```

リポジトリのファイルを直接編集してもよい。その場合は `chezmoi apply` で反映する。

### パッケージ

```sh
./dotfiles                        管理画面 (TUI)
./dotfiles status                 パッケージと設定の状態
./dotfiles install starship       個別に導入
./dotfiles install all            全部導入 (その OS で導入できるものだけ)
./dotfiles uninstall herdr        削除 (-y で確認省略)
./dotfiles apply                  chezmoi apply のショートカット (-y で確認省略)
./dotfiles diff                   chezmoi diff のショートカット
```

`apply` は、ターゲットを chezmoi の外で直接編集していた場合だけ確認を出す。

```
$ ./dotfiles apply
以下は chezmoi の外で編集されています。apply すると変更は失われます:
  ~/.config/starship.toml
残したい場合は先に chezmoi re-add でソースへ取り込んでください。
上書きしますか? [y/N]:
```

`chezmoi apply` 時にも `.chezmoiscripts` の導入スクリプトが走るので、
新しいマシンでは bootstrap だけで一通り揃う。TUI は「後から個別に足す / 消す」ためのもの。

## TUI

```
 dotfiles                                      macOS · brew · chezmoi · 2 selected
 1 Packages (6/8)   2 Configs (1)
 ────────────────

   PACKAGE      STATE               DESCRIPTION
 ❯ ◉ starship   ✔ installed         cross-shell prompt
   ○ neovim     ✔ installed         editor
   ◉ herdr      ✘ not installed     terminal agent manager
   · wtq        — Windows のみ      quake style terminal

╭─ starship ──────────────────────────────────────────────────╮
│ cross-shell prompt                                          │
│ 対応     macOS / Windows                                    │
│ brew     starship         installed                         │
╰─────────────────────────────────────────────────────────────╯

 tab タブ · ↑↓ 移動 · space 選択 · a 全選択 · i install · X uninstall · r 更新 · ? ヘルプ · q 終了
```

| キー | 動作 |
| --- | --- |
| `tab` / `1` `2` | タブ切替 (Packages / Configs) |
| `↑` `↓` / `k` `j` | カーソル移動 |
| `g` / `G` | 先頭 / 末尾 |
| `space` | 選択のトグル |
| `a` | 全選択 / 全解除 |
| `i` | install（Packages） |
| `X` | uninstall（Packages。確認あり） |
| `A` | apply（Configs。ターゲットを直接編集していれば確認あり） |
| `d` | diff（Configs） |
| `R` | re-add（Configs。確認あり） |
| `r` | 状態を再取得 |
| `?` | ヘルプ |
| `q` | 終了 |

選択が無いときはカーソル行が対象になる。Configs タブで差分が無いときは管理対象すべてが対象になる。
グレーの行（その OS で導入できないパッケージ）は `space` でも `a` でも選択されない。
実行中はコマンドの出力がストリーム表示される。

## ビルド

```sh
make build          # dist/dotfiles-darwin-arm64 と dist/dotfiles.exe を生成
make build-mac      # darwin/arm64 のみ
make build-windows  # windows/amd64 のみ
make build-linux    # linux/amd64 のみ
make local          # ホスト向けに ./dotfiles を生成
make run            # ビルドせず go run . で起動
make help           # ターゲット一覧
```

`dist/` と `./dotfiles` は `.gitignore` 済み。Windows 機には `dist/dotfiles.exe` を持っていく。

## 管理対象を追加するとき

### 設定ファイル

```sh
chezmoi add ~/.config/foo/config.toml
```

`home/dot_config/foo/config.toml` に取り込まれる。
片方の OS でしか使わないものは `home/.chezmoiignore` に条件を足す。

```
{{ if ne .chezmoi.os "darwin" }}
.config/foo
{{ end }}
```

Windows 専用のものは `home/AppData/...` のようにターゲット相対のパスで置く。

### パッケージ

`home/.chezmoidata/packages.toml` にブロックを 1 つ足す。1 パッケージ 1 ブロックで、
OS ごとの識別子を並べて書く。

```toml
[[packages]]
name = "lazygit"                        # 表示名。OS に依らない。必須
description = "git TUI"
brew = "lazygit"                        # 無ければ macOS では導入対象外
winget = "JesseDuffield.lazygit"        # 無ければ Windows では導入対象外
```

このファイルは chezmoi のテンプレートデータであると同時に `./dotfiles` の入力でもあるので、
ここだけ直せば `chezmoi apply` の自動導入と TUI の両方に反映される。

パッケージマネージャに存在しないものは、OS 別のサブテーブルで逃がす。

```toml
[[packages]]
name = "herdr"
description = "terminal agent manager"
brew = "herdr"

  # winget にも scoop にも無いので公式インストーラを叩く
  [packages.windows]
  script = "irm https://herdr.dev/install.ps1 | iex"
  check = "herdr"                       # このコマンドがあれば導入済みとみなす
```

`script` は `sh -c`（macOS）/ `pwsh -Command`（Windows）で実行される。
この経路で入れたものはパッケージマネージャの管理外なので、**uninstall はできない**。

### 他 OS 専用のパッケージ

`brew` も `winget` も持たない OS では、そのパッケージは一覧にグレーで表示され、選択できない。
消してしまうと「片方の OS にだけ足したこと」に気づけないので、存在は見えるようにしてある。

```
   PACKAGE      STATE               DESCRIPTION
 ❯ ◉ herdr      ✔ installed         terminal agent manager
   · wtq        — Windows のみ      quake style terminal
   · 7zip       — Windows のみ      archiver
```

タブの件数も `Packages (6/8)` のように「導入できる数 / 全体」で出る。

## 補足

- PowerShell の profile は `~/.config/powershell/profile.ps1` を本体とし、
  `$PROFILE` には「本体を読み込む 1 行」だけを `.chezmoiscripts` が書き込む。
  `$PROFILE` の場所は PowerShell 自身が解決するので、OneDrive による Documents の
  リダイレクトやホスト (pwsh / Windows PowerShell / VSCode) の違いに追従できる
- `.chezmoiscripts` の導入スクリプトは `run_onchange_` なので、
  `packages.toml` を増減したときだけ走る
- `chezmoi` は「最後に書いた後にターゲットが変わっている」と上書き確認のプロンプトを出すが、
  `./dotfiles` は chezmoi に TTY を渡していないので答えられない。そのため
  `--no-tty` を付けた上で、確認は `./dotfiles` 側が `chezmoi status` の 1 文字目を見て出す
- Windows 側は実機での動作確認が未了（テンプレートの展開結果までしか確認していない）。
  この Mac に `pwsh` が無いため、生成される `.ps1` の構文チェックも通せていない
- `wtq` の winget ID `windows-terminal-quake` は他と違い `Publisher.Package` 形式でない。
  旧 `windows/setup.ps1` から引き継いだ値なので、実機で `winget search wtq` して確かめること
