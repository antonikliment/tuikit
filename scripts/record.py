"""Record a terminal program to an animated GIF and a still PNG.

This is the docs recorder. It needs no vhs, ttyd or asciinema, only pyte and
Pillow: the program runs in a pty, pyte plays its escape sequences back into a
screen buffer, and Pillow draws each frame. The output is therefore a render of
what the program actually emitted rather than a capture of one machine's
terminal theme, and it is reproducible on any machine with Python.

    uv venv /tmp/rec && uv pip install --python /tmp/rec/bin/python pyte pillow
    go build -o /tmp/tuikit-demo ./examples/demo
    /tmp/rec/bin/python scripts/record.py \
        docs/gifs/table.gif docs/screenshots/6-table.png -- /tmp/tuikit-demo
    magick docs/gifs/table.gif -layers optimize -colors 96 docs/gifs/table.gif

Keystrokes and timing come from SCRIPT below; retarget it to record another
page. Run it from the repository root.
"""

import fcntl
import os
import pty
import select
import struct
import sys
import termios
import time

import pyte
from PIL import Image, ImageDraw, ImageFont

COLS, ROWS = 140, 26
FONT_PATH = "/usr/share/fonts/TTF/JetBrainsMonoNerdFontMono-Regular.ttf"
FONT_SIZE = 13
PAD = 16
FRAME_MS = 120
BG = (13, 17, 23)
FG = (200, 205, 212)

# (delay before, keys to send, capture from here on)
SCRIPT = [
    (2.6, None, False),   # let the demo paint its first page
    (0.0, "6", True),     # Table page
    (2.4, " ", True),     # any key starts the simulated transfer
    (7.0, None, True),
]
STILL_AT = 4.2  # seconds into the capture, bar part-way across

# pyte reports ANSI names; the rest arrive as bare hex.
NAMED = {
    "black": (40, 44, 52), "red": (224, 108, 117), "green": (152, 195, 121),
    "brown": (229, 192, 123), "yellow": (229, 192, 123), "blue": (97, 175, 239),
    "magenta": (198, 120, 221), "cyan": (86, 182, 194), "white": (200, 205, 212),
    "brightblack": (120, 128, 140), "brightred": (255, 130, 139),
    "brightgreen": (166, 226, 120), "brightbrown": (255, 214, 130),
    "brightyellow": (255, 214, 130), "brightblue": (120, 190, 255),
    "brightmagenta": (215, 140, 235), "brightcyan": (110, 205, 215),
    "brightwhite": (255, 255, 255),
}


def color(name, default):
    if not name or name == "default":
        return default
    if name in NAMED:
        return NAMED[name]
    if len(name) == 6:
        try:
            return tuple(int(name[i:i + 2], 16) for i in (0, 2, 4))
        except ValueError:
            pass
    return default


def run(command):
    """Drive command in a pty, returning screen snapshots as (t, buffer)."""
    screen = pyte.Screen(COLS, ROWS)
    stream = pyte.ByteStream(screen)
    pid, fd = pty.fork()
    if pid == 0:
        os.environ.update(TERM="xterm-256color", COLORTERM="truecolor", LINES=str(ROWS), COLUMNS=str(COLS))
        os.execvp(command[0], command)
    fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", ROWS, COLS, 0, 0))

    frames, start, capturing = [], time.time(), False
    for delay, keys, capture in SCRIPT:
        if keys:
            os.write(fd, keys.encode())
        capturing = capturing or capture
        deadline = time.time() + delay
        while time.time() < deadline:
            ready, _, _ = select.select([fd], [], [], FRAME_MS / 1000)
            if ready:
                try:
                    data = os.read(fd, 65536)
                except OSError:
                    data = b""
                if not data:
                    break
                stream.feed(data)
            if capturing:
                frames.append((time.time() - start, snapshot(screen)))
    os.write(fd, b"\x03")
    time.sleep(0.3)
    os.close(fd)
    os.waitpid(pid, os.WNOHANG)
    return frames


def snapshot(screen):
    return [[screen.buffer[y][x] for x in range(COLS)] for y in range(ROWS)]


def draw(buffer, font, cell_w, cell_h):
    image = Image.new("RGB", (COLS * cell_w + 2 * PAD, ROWS * cell_h + 2 * PAD), BG)
    canvas = ImageDraw.Draw(image)
    for y, row in enumerate(buffer):
        for x, char in enumerate(row):
            px, py = PAD + x * cell_w, PAD + y * cell_h
            bg = color(char.bg, BG)
            if char.reverse:
                bg = color(char.fg, FG)
            if bg != BG:
                canvas.rectangle([px, py, px + cell_w, py + cell_h], fill=bg)
            if char.data.strip():
                fg = BG if char.reverse else color(char.fg, FG)
                canvas.text((px, py), char.data, font=font, fill=fg)
    return image


def main():
    gif_path, png_path = sys.argv[1], sys.argv[2]
    command = sys.argv[sys.argv.index("--") + 1:]

    frames = run(command)
    font = ImageFont.truetype(FONT_PATH, FONT_SIZE)
    cell_w = round(font.getlength("M"))
    ascent, descent = font.getmetrics()
    cell_h = ascent + descent

    images, times, previous = [], [], None
    for t, buffer in frames:
        if buffer == previous:
            continue
        previous = buffer
        images.append(draw(buffer, font, cell_w, cell_h))
        times.append(t)
    if not images:
        sys.exit("no frames captured")

    durations = [int(1000 * (b - a)) for a, b in zip(times, times[1:])] + [1200]
    images[0].save(gif_path, save_all=True, append_images=images[1:],
                   duration=durations, loop=0, optimize=True)

    still = min(range(len(times)), key=lambda i: abs(times[i] - STILL_AT))
    images[still].save(png_path)
    print(f"{gif_path}: {len(images)} frames, {sum(durations)/1000:.1f}s")
    print(f"{png_path}: frame at t={times[still]:.1f}s")


main()
