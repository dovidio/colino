# Colino Demo Recording
# This tape demonstrates Colino's core functionality

# Set up the terminal - output both GIF and ASCII golden file
Output demo/demo.gif
Output demo/golden.ascii
Set TypingSpeed 0.05
Set Shell zsh

Sleep 1s

# start setup
Type "./colino setup"
Enter
Sleep 1s 
Enter
Sleep 1s

# rss config
Type "http://localhost:8080/rss"
Sleep 100ms
Enter
Sleep 1s

# no youtube setup
Type "n"
Sleep 500ms

# interval
Type "330"
Enter
Sleep 500ms

# override ai
Type "y"
Sleep 1s
Type "mmeta-llama/llama-3.3-70b-instruct:free"
Sleep 1s
Enter
Sleep 100ms
# fast backpace
Set TypingSpeed 0.001
Backspace 26
Set TypingSpeed 0.05

# base url
Type "https://openrouter.ai/api/v1"
Sleep 1s
Enter
Sleep 1s
Escape
Enter
Sleep 500ms

# confirm llm prompt
Escape
Type "[27;5;13~"
Sleep 1s
Escape

# confirm streaming
Type "y"
Sleep 1s

# mcp configuration 
Enter
Sleep 300ms

# Final confirmation
Enter
Sleep 1s

# Ingest
Type "./colino daemon"
Enter
Sleep 5s

# List
Type "./colino list"
Enter
Sleep 3s

# Digest
Type "./colino digest http://localhost:8080/articles/1"
Enter
Sleep 5s
