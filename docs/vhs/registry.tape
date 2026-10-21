# VHS tape: Lenger Registry Search, Tag Browsing & Update Checks
Output docs/vhs/registry.gif

Set Shell "bash"
Set FontSize 14
Set Width 1280
Set Height 720
Set Padding 10
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful

Type "lenger --theme omarchy"
Enter
Sleep 2s

# Navigate to Images
Type "2"
Sleep 1s

# Open Registry Search overlay
Type "S"
Sleep 1.5s

# Search for official images
Type "nginx"
Enter
Sleep 2s

# Browse remote tags
Type "t"
Sleep 2s
Down 2
Sleep 1s

# Back to search results
Escape
Sleep 1s
Escape
Sleep 1s

# Check update on selected image
Type "U"
Sleep 2s

# Scan vulnerabilities
Type "V"
Sleep 2s

Type "q"
Sleep 1s
