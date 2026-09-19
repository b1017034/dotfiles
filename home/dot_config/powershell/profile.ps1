# Starship
if (Get-Command starship -ErrorAction SilentlyContinue) {
    Invoke-Expression (&starship init powershell)
} else {
    Write-Warning "starship is not installed. See: https://starship.rs/"
}

# mise (各ランタイムを PATH の先頭に置くので、他の PATH 設定より後に読むこと)
if (Get-Command mise -ErrorAction SilentlyContinue) {
    Invoke-Expression (&mise activate pwsh)
} else {
    Write-Warning "mise is not installed. See: https://mise.jdx.dev/"
}
