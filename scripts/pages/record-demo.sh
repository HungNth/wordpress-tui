#!/bin/sh
set -eu

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT_DIR"

command -v vhs >/dev/null 2>&1 || {
    echo "error: vhs is required (install via 'winget install charmbracelet.vhs' or brew/go install)" >&2
    exit 1
}

command -v ffmpeg >/dev/null 2>&1 || {
    echo "error: ffmpeg is required to generate video and image assets" >&2
    exit 1
}

# Ensure NO_COLOR is not set so Bubble Tea / Lip Gloss renders full truecolor
unset NO_COLOR || true
export COLORTERM=truecolor
export TERM=xterm-256color

echo "Building wptui binary..."
go build -o bin/wptui.exe ./cmd/wptui

STAGE_DIR="$ROOT_DIR/target/vhs-stage"
cleanup() {
    rm -rf "$STAGE_DIR"
}
trap cleanup EXIT INT TERM

rm -rf "$STAGE_DIR"
mkdir -p "$STAGE_DIR/.config/wptui" \
         "$STAGE_DIR/bin" \
         "$STAGE_DIR/Websites/client-store" \
         "$STAGE_DIR/Websites/developer-blog" \
         "$STAGE_DIR/Websites/marketing-site"

echo "<?php // mock WordPress" > "$STAGE_DIR/Websites/client-store/wp-config.php"
echo "<?php // mock WordPress" > "$STAGE_DIR/Websites/developer-blog/wp-config.php"
echo "<?php // mock WordPress" > "$STAGE_DIR/Websites/marketing-site/wp-config.php"

# Isolated WP-CLI stub for deterministic Toggle Debug Mode progress capture
cat <<'EOF' > "$STAGE_DIR/bin/wp.bat"
@echo off
if "%1 %2 %3"=="config get WP_DEBUG" (
  echo false
  exit /b 0
)
echo Success: Updated constant '%3'.
exit /b 0
EOF

WIN_STAGE_DIR="$(cygpath -w "$STAGE_DIR" 2>/dev/null || echo "$STAGE_DIR" | sed 's|^/\([a-zA-Z]\)/|\1:\\|; s|/|\\|g')"
WIN_STAGE_DIR_ESC="$(echo "$WIN_STAGE_DIR" | sed 's|\\|\\\\|g')"

cat <<EOF > "$STAGE_DIR/.config/wptui/config.json"
{
  "app_name": "herd",
  "app_command": "herd",
  "websites_path": "${WIN_STAGE_DIR_ESC}\\\\Websites",
  "backup_path": "${WIN_STAGE_DIR_ESC}\\\\Websites\\\\backups",
  "default_admin_username": "admin",
  "default_admin_password": "admin",
  "default_admin_email": "admin@example.test",
  "database_host": "127.0.0.1",
  "database_port": 3306,
  "db_username": "root",
  "db_password": ""
}
EOF

THEME_JSON='{ "background": "#0b1017", "foreground": "#edf2f7", "black": "#0b1017", "brightBlack": "#888888", "red": "#ff4444", "brightRed": "#ff4444", "green": "#04b575", "brightGreen": "#04b575", "yellow": "#ffa500", "brightYellow": "#ffa500", "blue": "#2271b1", "brightBlue": "#4aa8e8", "magenta": "#d946ef", "brightMagenta": "#e879f9", "cyan": "#00ffff", "brightCyan": "#00ffff", "white": "#edf2f7", "brightWhite": "#ffffff" }'

VIDEO_TAPE="$STAGE_DIR/video.tape"
cat <<EOF > "$VIDEO_TAPE"
Output website/wptui-demo.webm
Output website/wptui-demo.mp4

Set FontFamily "Cascadia Code, Consolas, monospace"
Set FontSize 18
Set Width 1080
Set Height 600
Set Padding 12
Set WindowBar "Colorful"
Set Theme ${THEME_JSON}

Hide
Type "set NO_COLOR="
Enter
Type "set COLORTERM=truecolor"
Enter
Type "set TERM=xterm-256color"
Enter
Type "set USERPROFILE=${WIN_STAGE_DIR}"
Enter
Sleep 200ms
Type "cls"
Enter
Sleep 200ms
Type ".\\\\bin\\\\wptui.exe"
Enter
Sleep 2s
Show

Sleep 1.5s
Tab
Sleep 800ms
Space
Sleep 800ms
Down
Sleep 600ms
Space
Sleep 1s
Down
Sleep 600ms
Space
Sleep 1s
Escape
Sleep 1s
Down
Sleep 600ms
Down
Sleep 600ms
Up
Sleep 600ms
Up
Sleep 1s
Type "q"
Sleep 800ms
EOF

SHOTS_TAPE="$STAGE_DIR/screenshots.tape"
cat <<EOF > "$SHOTS_TAPE"
Output target/vhs-stage/shots-unused.mp4

Set FontFamily "Cascadia Code, Consolas, monospace"
Set FontSize 18
Set Width 1080
Set Height 600
Set Padding 12
Set WindowBar "Colorful"
Set Theme ${THEME_JSON}

Hide
Type "set NO_COLOR="
Enter
Type "set COLORTERM=truecolor"
Enter
Type "set TERM=xterm-256color"
Enter
Type "set USERPROFILE=${WIN_STAGE_DIR}"
Enter
Type "set PATH=${WIN_STAGE_DIR}\\bin;%PATH%"
Enter
Sleep 200ms
Type "cls"
Enter
Sleep 200ms
Type ".\\\\bin\\\\wptui.exe"
Enter
Sleep 2s
Show

# 1. Websites Hub (with multi-select checkboxes)
Tab
Sleep 300ms
Space
Sleep 200ms
Down
Sleep 200ms
Space
Sleep 400ms
Screenshot website/screenshot-hub.png
Sleep 300ms

# Reset selection and move to client-store -> Website Actions -> Backup Strategy
Space
Sleep 150ms
Up
Sleep 150ms
Space
Sleep 200ms
Enter
Sleep 300ms
Down
Down
Down
Sleep 300ms
Enter
Sleep 500ms
Screenshot website/screenshot-backup.png
Sleep 300ms

# 3. Return to Website Actions -> Toggle Debug Mode -> In-Pane Progress Monitor
Escape
Sleep 300ms
Up
Sleep 300ms
Enter
Sleep 1.8s
Screenshot website/screenshot-progress.png
Sleep 300ms

# 4. Dismiss completion -> Sidebar -> Create Wizard (Provisioning)
Enter
Sleep 400ms
Escape
Sleep 300ms
Escape
Sleep 300ms
Down
Sleep 300ms
Enter
Sleep 300ms
Type "Portfolio Shop"
Sleep 200ms
Down
Sleep 200ms
Type "portfolio-shop"
Sleep 200ms
Down
Sleep 400ms
Screenshot website/screenshot-provisioning.png
Sleep 300ms

Ctrl+C
Sleep 400ms
EOF

echo "Recording walkthrough video..."
vhs "$VIDEO_TAPE"

echo "Extracting preview poster at peak interaction..."
ffmpeg -ss 00:00:04.5 -i website/wptui-demo.mp4 -vframes 1 -update 1 website/wptui-demo-preview.png -y >/dev/null 2>&1

echo "Capturing 4 workflow screenshots..."
vhs "$SHOTS_TAPE"

echo "Converting screenshots to lossless WebP..."
for name in screenshot-hub screenshot-backup screenshot-progress screenshot-provisioning; do
    ffmpeg -i "website/${name}.png" -c:v libwebp -lossless 1 "website/${name}.webp" -y >/dev/null 2>&1
done

echo "Generated assets:"
ls -lh website/wptui-demo* website/screenshot-*
