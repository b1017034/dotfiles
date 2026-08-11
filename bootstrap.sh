#!/bin/sh
# chezmoi をこのリポジトリに向けて設定する。
#
#   ./bootstrap.sh            設定を書いて差分を表示するだけ
#   ./bootstrap.sh --apply    差分の適用まで行う
#

set -eu

apply=0
for a in "$@"; do
	case "$a" in
	--apply) apply=1 ;;
	-h | --help)
		sed -n '2,9p' "$0" | sed 's/^# \{0,1\}//'
		exit 0
		;;
	*)
		echo "不明な引数: $a" >&2
		exit 2
		;;
	esac
done

repo="$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)"
config="${XDG_CONFIG_HOME:-$HOME/.config}/chezmoi/chezmoi.toml"

if ! command -v chezmoi >/dev/null 2>&1; then
	echo "chezmoi が見つかりません。先に導入してください:" >&2
	echo "  brew install chezmoi" >&2
	exit 1
fi

if [ -f "$config" ] && ! grep -qF "$repo" "$config"; then
	echo "既存の設定が別の場所を指しています: $config" >&2
	echo "内容を確認してから手で消すか、書き換えてください" >&2
	exit 1
fi

# chezmoi の source dir は既定で ~/.local/share/chezmoi だが、このリポジトリは
# ghq の管理下に置きたいので sourceDir を設定ファイルに書いて上書きする。
# リポジトリルートの .chezmoiroot が home/ を指すので、実際の source dir は
# <repo>/home に解決される。
mkdir -p "$(dirname "$config")"
cat >"$config" <<EOF
sourceDir = "$repo"

# .ps1 スクリプトには shebang が書けないのでインタプリタを明示する。
[interpreters.ps1]
command = "pwsh"
args = ["-NoLogo", "-NoProfile"]
EOF
echo "wrote: $config"
echo

chezmoi doctor || true
echo

if [ "$apply" -eq 1 ]; then
	chezmoi apply -v
else
	chezmoi diff
	echo
	echo "適用するには: chezmoi apply -v   (または ./bootstrap.sh --apply)"
fi
