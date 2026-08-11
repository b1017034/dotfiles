# xcstrings-diff — readable git diffs for Xcode String Catalogs (*.xcstrings)
#
# Flattens the JSON catalog into "<key> [<lang>] = <value>" lines via a
# per-invocation git textconv driver, so diffs show only the strings that
# actually changed instead of raw JSON. Nothing is written to .git/config.
#
# Requires: jq, and `*.xcstrings diff=xcstrings` in the repo's .gitattributes.
#
# Usage:
#   xdiff show HEAD^^^^ -- '*.xcstrings'
#   xdiff diff
#   xdiff diff main...HEAD -- '*.xcstrings'
#   (any `git` subcommand works; textconv only affects *.xcstrings)

# jq program kept on one line; git appends the blob path as jq's input file.
_xcstrings_textconv='jq -r '\''[.strings // {} | to_entries[] | .key as $k | (.value.localizations // {} | to_entries[]) | "\($k) [\(.key)] = \(.value.stringUnit.value // (.value | tostring))"] | sort | .[]'\'''

xdiff() {
  git -c "diff.xcstrings.textconv=${_xcstrings_textconv}" "$@"
}
