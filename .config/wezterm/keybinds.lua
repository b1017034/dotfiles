local wezterm = require("wezterm")
local act = wezterm.action

-- ===== status (tmux風) =====
wezterm.on("update-status", function(window, pane)
  local workspace = window:active_workspace()
  local tab = window:active_tab():tab_id()
  local pane_id = pane:pane_id()
  local leader = ""
  if window:leader_is_active() then
    leader = "LEADER " -- Leaderがアクティブな時の表示
  end


  -- 左側にLeader状態とワークスペースを表示
  window:set_left_status(wezterm.format({
    { Foreground = { Color = "#ffb86c" } }, -- Leader文字の色（オレンジ系）
    { Text = leader },
    { Foreground = { Color = "white" } },
    { Text = " 󱂬 " .. workspace .. " " },
  }))
  window:set_right_status(" Pane:" .. pane_id .. " Tab:" .. tab .. " ")
  
end)

return {
  -- prefix
  leader = { key = "a", mods = "CTRL", timeout_milliseconds = 1000 },

  keys = {
    -- window(tab)操作
    { key = "c", mods = "LEADER", action = act.SpawnTab("CurrentPaneDomain") },
    { key = "n", mods = "LEADER", action = act.ActivateTabRelative(1) },
    { key = "p", mods = "LEADER", action = act.ActivateTabRelative(-1) },
    { key = "&", mods = "LEADER|SHIFT", action = act.CloseCurrentTab({ confirm = true }) },

    -- 番号移動
    { key = "1", mods = "LEADER", action = act.ActivateTab(0) },
    { key = "2", mods = "LEADER", action = act.ActivateTab(1) },
    { key = "3", mods = "LEADER", action = act.ActivateTab(2) },
    { key = "4", mods = "LEADER", action = act.ActivateTab(3) },
    { key = "5", mods = "LEADER", action = act.ActivateTab(4) },
    { key = "6", mods = "LEADER", action = act.ActivateTab(5) },
    { key = "7", mods = "LEADER", action = act.ActivateTab(6) },
    { key = "8", mods = "LEADER", action = act.ActivateTab(7) },
    { key = "9", mods = "LEADER", action = act.ActivateTab(-1) },

    -- pane操作
    { key = '"', mods = "LEADER|SHIFT", action = act.SplitVertical({ domain = "CurrentPaneDomain" }) },
    { key = "%", mods = "LEADER|SHIFT", action = act.SplitHorizontal({ domain = "CurrentPaneDomain" }) },

    { key = "h", mods = "LEADER", action = act.ActivatePaneDirection("Left") },
    { key = "j", mods = "LEADER", action = act.ActivatePaneDirection("Down") },
    { key = "k", mods = "LEADER", action = act.ActivatePaneDirection("Up") },
    { key = "l", mods = "LEADER", action = act.ActivatePaneDirection("Right") },
    { key = "LeftArrow",  mods = "LEADER", action = act.ActivatePaneDirection("Left") },
    { key = "DownArrow",  mods = "LEADER", action = act.ActivatePaneDirection("Down") },
    { key = "UpArrow",    mods = "LEADER", action = act.ActivatePaneDirection("Up") },
    { key = "RightArrow", mods = "LEADER", action = act.ActivatePaneDirection("Right") },

    { key = "r", mods = "LEADER", action = act.ActivateKeyTable({ name = "resize_pane", one_shot = false }) },

    { key = "x", mods = "LEADER", action = act.CloseCurrentPane({ confirm = true }) },
    { key = "z", mods = "LEADER", action = act.TogglePaneZoomState },
    { key = "q", mods = "LEADER", action = act.PaneSelect },

    -- copy mode
    { key = "[", mods = "LEADER", action = act.ActivateCopyMode },

    -- その他
    { key = "p", mods = "CTRL|SHIFT", action = act.ActivateCommandPalette },
    { key = "r", mods = "CTRL|SHIFT", action = act.ReloadConfiguration },

    -- コピペ
    { key = "c", mods = "SUPER", action = act.CopyTo("Clipboard") },
    { key = "v", mods = "SUPER", action = act.PasteFrom("Clipboard") },
  },

  key_tables = {
    resize_pane = {
      { key = "h", action = act.AdjustPaneSize({ "Left", 2 }) },
      { key = "j", action = act.AdjustPaneSize({ "Down", 2 }) },
      { key = "k", action = act.AdjustPaneSize({ "Up", 2 }) },
      { key = "l", action = act.AdjustPaneSize({ "Right", 2 }) },
      { key = "Escape", action = "PopKeyTable" },
      { key = "Enter", action = "PopKeyTable" },
    },

    copy_mode = {
      { key = "h", action = act.CopyMode("MoveLeft") },
      { key = "j", action = act.CopyMode("MoveDown") },
      { key = "k", action = act.CopyMode("MoveUp") },
      { key = "l", action = act.CopyMode("MoveRight") },

      { key = "0", action = act.CopyMode("MoveToStartOfLine") },
      { key = "$", action = act.CopyMode("MoveToEndOfLineContent") },

      { key = "w", action = act.CopyMode("MoveForwardWord") },
      { key = "b", action = act.CopyMode("MoveBackwardWord") },

      { key = "f", mods = "CTRL", action = act.CopyMode("PageDown") },
      { key = "b", mods = "CTRL", action = act.CopyMode("PageUp") },

      { key = "v", action = act.CopyMode({ SetSelectionMode = "Cell" }) },
      { key = "V", action = act.CopyMode({ SetSelectionMode = "Line" }) },

      { key = "y", action = act.CopyTo("Clipboard") },

      { key = "Escape", action = act.CopyMode("Close") },
      { key = "q", action = act.CopyMode("Close") },
    },
  },
}
