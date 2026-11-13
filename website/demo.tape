# VHS tape to record WPTUI terminal walkthrough in HD TrueColor
# Run via: sh scripts/pages/record-demo.sh
# Or directly: vhs website/demo.tape

Output website/wptui-demo.webm
Output website/wptui-demo.mp4

Set FontFamily "Cascadia Code, Consolas, monospace"
Set FontSize 18
Set Width 1080
Set Height 600
Set Padding 12
Set WindowBar "Colorful"
Set Theme { "background": "#0b1017", "foreground": "#edf2f7", "black": "#0b1017", "brightBlack": "#888888", "red": "#ff4444", "brightRed": "#ff4444", "green": "#04b575", "brightGreen": "#04b575", "yellow": "#ffa500", "brightYellow": "#ffa500", "blue": "#2271b1", "brightBlue": "#4aa8e8", "magenta": "#d946ef", "brightMagenta": "#e879f9", "cyan": "#00ffff", "brightCyan": "#00ffff", "white": "#edf2f7", "brightWhite": "#ffffff" }

Hide
Type "set NO_COLOR="
Enter
Type "set COLORTERM=truecolor"
Enter
Type "set TERM=xterm-256color"
Enter
Type ".\bin\wptui.exe"
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
