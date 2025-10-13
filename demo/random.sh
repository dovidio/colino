#!/bin/bash

WIDTH=100
HEIGHT=15
CHARS="abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@#$%&*"

FPS=10
RANDOMIZE_FRAMES=20
ANIMATION_FRAMES=30
sleep_time=0.05

# ASCII art for "Colino"
ASCII_TEXT=(
":'######:::'#######::'##:::::::'####:'##::: ##::'#######::"
"'##... ##:'##.... ##: ##:::::::. ##:: ###:: ##:'##.... ##:"
" ##:::..:: ##:::: ##: ##:::::::: ##:: ####: ##: ##:::: ##:"
" ##::::::: ##:::: ##: ##:::::::: ##:: ## ## ##: ##:::: ##:"
" ##::::::: ##:::: ##: ##:::::::: ##:: ##. ####: ##:::: ##:"
" ##::: ##: ##:::: ##: ##:::::::: ##:: ##:. ###: ##:::: ##:"
". ######::. #######:: ########:'####: ##::. ##:. #######::"
":......::::.......:::........::....::..::::..:::.......:::"
)

declare -a buffer

# Initialize buffer with random characters
initialize_buffer() {
    for ((y=0; y<HEIGHT; y++)); do
        row=""
        for ((x=0; x<WIDTH; x++)); do
            row+="${CHARS:RANDOM%${#CHARS}:1}"
        done
        buffer[y]="$row"
    done
}

draw_buffer() {
    printf "\033[H"  # move cursor to top-left
    for ((y=0; y<HEIGHT; y++)); do
        echo "${buffer[y]}"
    done
}

shuffle_buffer() {
    for ((i=0; i<WIDTH*5; i++)); do
        x=$((RANDOM % WIDTH))
        y=$((RANDOM % HEIGHT))
        char="${CHARS:RANDOM%${#CHARS}:1}"
        buffer[y]="${buffer[y]:0:x}$char${buffer[y]:x+1}"
    done
}

fade_out_background() {
    # Calculate Colino position
    text_height=${#ASCII_TEXT[@]}
    text_width=${#ASCII_TEXT[0]}
    start_y=$(( (HEIGHT - text_height) / 2 ))
    start_x=$(( (WIDTH - text_width) / 2 ))

    for ((f=0; f<ANIMATION_FRAMES; f++)); do
        # Fade background
        for ((y=0; y<HEIGHT; y++)); do
            row="${buffer[y]}"
            new_row=""
            for ((x=0; x<WIDTH; x++)); do
                if (( RANDOM % ANIMATION_FRAMES < f )); then
                    new_row+=" "
                else
                    new_row+="${row:x:1}"
                fi
            done
            buffer[y]="$new_row"
        done

        overlay_colino_random

        draw_buffer
        sleep $sleep_time
    done
}

# Overlay Colino with random chars for ":" and gradually reveal
overlay_colino_random() {
    text_height=${#ASCII_TEXT[@]}
    text_width=${#ASCII_TEXT[0]}
    start_y=$(( (HEIGHT - text_height) / 2 ))
    start_x=$(( (WIDTH - text_width) / 2 ))

    for ((i=0; i<text_height; i++)); do
        row="${buffer[start_y + i]}"
        line="${ASCII_TEXT[i]}"
        new_row="${row:0:start_x}"

        for ((x=0; x<text_width; x++)); do
            target_char="${line:x:1}"
            if [[ "$target_char" == ":" ]]; then
                # Pick random char for animation
                if (( RANDOM % ANIMATION_FRAMES < f )); then
                    new_row+=":"
                else
                    new_row+="${CHARS:RANDOM%${#CHARS}:1}"
                fi
            else
                new_row+="$target_char"
            fi
        done

        # Pad to WIDTH
        while (( ${#new_row} < WIDTH )); do
            new_row+=" "
        done
        buffer[start_y + i]="$new_row"
    done
}

# Overlay Colino centered
show_colino() {
    text_height=${#ASCII_TEXT[@]}
    text_width=${#ASCII_TEXT[0]}
    start_y=$(( (HEIGHT - text_height) / 2 ))
    start_x=$(( (WIDTH - text_width) / 2 ))

    for ((i=0; i<text_height; i++)); do
        row="${buffer[start_y + i]}"
        line="${ASCII_TEXT[i]}"
        # Safely construct new row
        new_row="${row:0:start_x}$line"
        while (( ${#new_row} < WIDTH )); do
            new_row+=" "
        done
        buffer[start_y + i]="$new_row"
    done

    draw_buffer
}

fade_in_text() {
    local TEXT_LINES=(
        "Reclaim your attention from algorithmic feeds."
        "Choose your sources, set your pace, and consume information on your terms."
        "Colino, your personal content feed."
    )

    local text_height=${#TEXT_LINES[@]}
    local max_length=0
    for line in "${TEXT_LINES[@]}"; do
        (( ${#line} > max_length )) && max_length=${#line}
    done

    # Start positions to center text vertically and horizontally
    local start_y=$(( (HEIGHT - text_height) / 2 ))
    local start_x=$(( (WIDTH - max_length) / 2 ))

    # Initialize buffer lines with spaces
    for ((i=0; i<text_height; i++)); do
        buffer[start_y + i]=$(printf "%-${WIDTH}s" "")
    done

    # Gradually reveal characters
    for ((pos=0; pos<max_length; pos++)); do
        for ((i=0; i<text_height; i++)); do
            line="${TEXT_LINES[i]}"
            if (( pos < ${#line} )); then
                row="${buffer[start_y + i]}"
                buffer[start_y + i]="${row:0:start_x+pos}${line:pos:1}${row:start_x+pos+1}"
            fi
        done
        draw_buffer
        sleep $sleep_time
    done
}

# Main
initialize_buffer
printf "\033[2J"  # clear screen once
draw_buffer

# Shuffle grid for a few frames
for ((f=0; f<RANDOMIZE_FRAMES; f++)); do
    shuffle_buffer
    draw_buffer
    sleep $sleep_time
done

fade_out_background
show_colino

