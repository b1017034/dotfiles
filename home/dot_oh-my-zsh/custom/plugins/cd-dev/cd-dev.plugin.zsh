ghq-path() {
  ghq list --full-path | fzf
}

dev() {
  local moveto
  moveto=$(ghq-path) || return 1
  [[ -z $moveto ]] && return 1
  cd "$moveto" || return 1
}

