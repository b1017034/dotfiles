local wezterm = require("wezterm")
local config = wezterm.config_builder()

config.automatically_reload_config = true
config.font_size = 10.0
config.use_ime = true
config.window_background_opacity = 0.85
config.macos_window_background_blur = 20

-- powershell の利用
config.default_prog = { 'pwsh' }

----------------------------------------------------
-- Tab
----------------------------------------------------
-- タブバーの表示
config.show_tabs_in_tab_bar = true

-- タブバーの透過
config.window_frame = {
  inactive_titlebar_bg = "none",
  active_titlebar_bg = "none",
}

-- タブバーを背景色に合わせる
config.window_background_gradient = {
  colors = { "#000000" },
}

-- タブの形をカスタマイズ
wezterm.on('format-tab-title', function(tab, tabs, panes, config, hover, max_width)
  local bg = '#1a1b26'
  local fg = '#565f89'

  if tab.is_active then
    bg = '#7aa2f7'
    fg = '#1a1b26'
  end

  local title = tab.active_pane.title
  if #title > max_width - 2 then
    title = wezterm.truncate_right(title, max_width - 2)
  end

  return {
    -- 左セパレータ
    { Background = { Color = '#1a1b26' } },
    { Foreground = { Color = bg } },

    -- タブ本体
    { Background = { Color = bg } },
    { Foreground = { Color = fg } },
    { Attribute = { Intensity = tab.is_active and "Bold" or "Normal" } },
    { Text = ' ' .. title .. ' ' },
  }
end)

----------------------------------------------------
-- keybinds
----------------------------------------------------
config.disable_default_key_bindings = true
config.keys = require("keybinds").keys
config.key_tables = require("keybinds").key_tables
config.leader = require("keybinds").leader
return config

