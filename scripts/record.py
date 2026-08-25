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

Keystrokes and timing come from SCRIPTS below; RECORD=<name> picks one, and a
new entry records another page. Run it from the repository root.
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
QUIET_MS = 8  # gap in pty output that marks a repaint as finished
BG = (13, 17, 23)
FG = (200, 205, 212)

# Named recordings: (delay before, keys to send, capture from here on) plus the
# offset the still is taken at. Pick one with RECORD=<name>.
SCRIPTS = {
    "table": ([
        (2.6, None, False),   # let the demo paint its first page
        (0.0, "6", True),     # Table page
        (2.4, " ", True),     # any key starts the simulated transfer
        (7.0, None, True),
    ], 4.2),                  # bar part-way across
    "streaming": ([
        (1.5, None, False),   # let the first blocks arrive
        (26.0, None, True),   # the stream renders itself; no input needed
    ], 22.5),                 # on a code fence, which is what the tail handling
                              # is most visibly doing something about
    # The two rounds of the display clock, same seed and rate, so the pair can
    # be watched against each other. Both are stamped with a clock (see draw)
    # because the thing being compared is *how long the screen stops moving*,
    # which no still frame shows.
    "reveal-eager": ([
        (0.4, None, False),
        (24.0, None, True),
    ], 12.0),
    "reveal-buffered": ([
        (0.4, None, False),
        (24.0, None, True),
    ], 12.0),
    # Three chroma stylesheets, tab between them. Held long enough on each to
    # read the code, since the whole subject is which tokens changed color.
    # A mouse drag, sent as SGR sequences straight into the pty: press at
    # (col;row), three motions with the button held (button code +32), then
    # release. The app paints the highlight, so the recording shows exactly what
    # a viewer would see.
    "selection": ([
        (1.4, None, False),                  # first paint
        (0.7, "\x1b[<0;10;5M", True),        # press, mid-sentence
        (0.5, "\x1b[<32;57;5M", True),       # drag to the end of the line
        (0.5, "\x1b[<32;44;6M", True),       # and down one
        (0.9, "\x1b[<32;31;7M", True),       # and down another
        (1.6, "\x1b[<0;31;7m", True),        # release: the footer echoes the copy
        (1.2, None, True),
    ], 2.6),                                 # mid-drag, two lines in
    # The same drag twice over the three-panel row, linear then alt-held: the
    # first runs to the ends of the lines and takes the neighbouring panels
    # with it, the second stays inside the rectangle. Alt is button code +8.
    "selection-block": ([
        (1.4, None, False),                  # first paint
        (0.7, "\x1b[<0;3;15M", True),        # press inside the left panel
        (0.5, "\x1b[<32;12;16M", True),      # drag down and right
        (0.6, "\x1b[<32;21;17M", True),
        (1.8, "\x1b[<0;21;17m", True),       # release: all three panels copied
        (0.9, "\x1b[<8;3;15M", True),        # same press, alt held
        (0.5, "\x1b[<40;12;16M", True),
        (0.6, "\x1b[<40;21;17M", True),
        (1.8, "\x1b[<8;21;17m", True),       # release: just the left panel
        (1.0, None, True),
    ], 8.0),                                 # mid alt-drag, the block visible
    "syntax": ([
        (1.5, None, False),   # first paint
        (2.6, None, True),    # tokyonight-night
        (2.6, "\t", True),    # tokyonight-day
        (2.6, "\t", True),    # catppuccin-mocha
        (1.8, "\t", True),    # back to the start
    ], 6.5),                  # the light stylesheet, most obviously different
    # One edit rendered both ways. Held long enough on each layout to read the
    # modified pairs, since the intraline emphasis is the subject; the last beat
    # widens the context to show the hunk growing.
    "diffview": ([
        (1.6, None, False),   # first paint, unified
        (3.4, "\t", True),    # unified, then switch
        (3.6, "\t", True),    # side-by-side, then back
        (2.2, "+", True),     # one more line of context
        (1.8, None, True),
    ], 4.8),                  # side-by-side, both columns full
}
RECORD = os.environ.get("RECORD", "table")
SCRIPT, STILL_AT = SCRIPTS[RECORD]

# A clocked recording stamps every frame with elapsed time and keeps sampling
# while the screen is unchanged. Both are needed to record a *freeze*: a program
# that has stopped drawing emits nothing, so the usual dirty-and-deduplicate
# path would collapse the freeze into a single frame and the clock would appear
# to stop with it.
CLOCK = RECORD.startswith("reveal-")

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
    last, dirty = 0.0, False
    for delay, keys, capture in SCRIPT:
        if keys:
            os.write(fd, keys.encode())
        capturing = capturing or capture
        deadline = time.time() + delay
        while time.time() < deadline:
            ready, _, _ = select.select([fd], [], [], QUIET_MS / 1000)
            if ready:
                try:
                    data = os.read(fd, 65536)
                except OSError:
                    data = b""
                if not data:
                    break
                stream.feed(data)
                dirty = True
                continue
            # The pty has gone quiet, so the last repaint is complete. Sampling
            # while a write is still in flight catches the screen mid-escape-
            # sequence and tears the frame — a program that repaints on every
            # character is almost always mid-write. Waiting for the gap is what
            # makes each captured frame a whole one.
            now = time.time()
            if capturing and (dirty or CLOCK) and now - last >= FRAME_MS / 1000:
                last, dirty = now, False
                frames.append((now - start, snapshot(screen)))
    os.write(fd, b"\x03")
    time.sleep(0.3)
    os.close(fd)
    os.waitpid(pid, os.WNOHANG)
    return frames


def snapshot(screen):
    return [[screen.buffer[y][x] for x in range(COLS)] for y in range(ROWS)]


def draw(buffer, font, cell_w, cell_h, t=None):
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
    if t is not None:
        # A wall clock, drawn by the recorder rather than the program, so two
        # recordings of different builds share one timebase.
        label = f"{t:5.1f}s"
        w = font.getlength(label)
        canvas.rectangle([image.width - PAD - w - 8, 2, image.width, PAD + cell_h],
                         fill=BG)
        canvas.text((image.width - PAD - w, 4), label, font=font, fill=(120, 190, 255))
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
        if buffer == previous and not CLOCK:
            continue
        previous = buffer
        images.append(draw(buffer, font, cell_w, cell_h, t - frames[0][0] if CLOCK else None))
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
