# Colino TUI Demo Recording
# This tape demonstrates Colino's TUI functionality: search, navigation, reading

# Set up the terminal with window chrome
Output tapes/tui.gif
Output tapes/tui.ascii
Set TypingSpeed 0.05
Set Shell zsh
Set WindowBar Colorful
Set WindowBarSize 40
Set Height 1000
Set Width 1200

Sleep 1s

# Start TUI
Type "./colino tui"
Enter
Sleep 1s

# Navigate through search results
Type "j"
Sleep 100ms
Type "j"
Sleep 100ms

# Open first article
Type " "
Sleep 500ms

# Scroll through article
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms

# Go back to list
Type "q"
Sleep 2s

# Second check
Type "j"
Sleep 200ms

# Open first article
Type " "
Sleep 500ms
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms

# Go back to list
Type "q"
Sleep 2s

