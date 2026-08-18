# tuikit

A small, reusable [Bubble Tea](https://charm.land) frame kit: the structural
chrome you rebuild in every terminal app — a numbered page wrapper with
navigation, chip tabs, and bordered panels — decoupled from any one app and
driven by a swappable theme.

## Demos

From `go run ./examples/demo` (and `./examples/themed` for the last one):

| Chip sub-tabs (`Tab` cycles) | Scrolling viewport |
| --- | --- |
| ![Panels](docs/gifs/panels.gif) | ![Reader](docs/gifs/reader.gif) |

| SearchView, ActionRow, and Help | Live theme switching (`t`) |
| --- | --- |
| ![Search](docs/gifs/search.gif) | ![Theme switch](docs/gifs/theme.gif) |

`Table`, `Pairs` and the formatters, with a simulated transfer driving `Meter`
and `TransferredBytes`. The `STATE` words are graded by `StatusWord` — green,
amber, red — and cost the columns beside them nothing, because widths are
measured by display width rather than by string length:

![Table](docs/gifs/table.gif)

`Overlay`, a modal popup composited over the page. `p` pins it to each of the
nine anchors in turn and `hjkl` nudges it off one; the page underneath never
reflows, and closing it leaves the page exactly as it was:

![Overlay](docs/gifs/overlay.gif)

<details>
<summary>Static screenshots</summary>

| Panels | Search |
| --- | --- |
| ![Panels page](docs/screenshots/1-panels.png) | ![Search page](docs/screenshots/4-search.png) |

| Reader | About |
| --- | --- |
| ![Reader page](docs/screenshots/2-reader.png) | ![About page](docs/screenshots/3-about.png) |

| Widgets (Meter · Status · text helpers) | Table (Table · Pairs · StatusWord) |
| --- | --- |
| ![Widgets page](docs/screenshots/5-widgets.png) | ![Table page](docs/screenshots/6-table.png) |

| Overlay (modal popup, pinned right) |
| --- |
| ![Overlay page](docs/screenshots/7-overlay.png) |

</details>

## Components

- **`Frame`** — a stateful `tea.Model` that hosts a list of pages, renders a
  numbered header (`[1] Foo  [2] Bar …`), delegates the body to the active page,
  and draws a status footer. Number keys `1`–`9` switch pages; `Ctrl+C` quits.
- **`Page`** — the seam you implement, a plain 3-method interface:
  ```go
  type Page interface {
      Title() string
      Update(msg tea.Msg) tea.Cmd
      View(width, height int) string
  }
  ```
  Size is passed into `View`, so pages never track their own dimensions.
  Optionally implement `InputCapturer` so the Frame stops treating number keys
  as navigation while a field is focused.
- **`Theme`** — the palette every component draws from. `DefaultTheme()` or roll
  your own and pass `WithTheme`.
- **`TabStrip`** — a row of active/inactive chip tabs for sub-navigation within
  a page.
- **`Panel`** / **`PanelStyle`** — bordered panels with a focused state.
- **`SearchView`** — a scrollable text pane with an incremental substring
  filter and follow-to-bottom behavior: feed it lines, it renders the matching
  subset, stays pinned to the bottom as new lines arrive (until you scroll up),
  and toggles a search input on `/`. Matching is against each line's visible
  text (ANSI styling is stripped first), so colored lines still search cleanly.
  The log/reader viewport every terminal app rebuilds by hand.
- **`StreamingMarkdown`** — markdown rendered while it is still arriving. Blocks
  that have provably settled are formatted and cached; the unfinished tail is
  wrapped verbatim and redrawn each frame, so a code fence streams as plain text
  and snaps to highlighted the moment it closes, instead of the whole answer
  popping into shape at the end. Takes a `RenderFunc`; `tuikit/markdown`
  provides one backed by glamour, in a separate package so importing tuikit does
  not pull in a markdown engine.
- **`Overlay`** — a dismissable floating panel: a titled, bordered, scrollable
  box drawn *over* the host's view rather than stacked above or below it, for
  the reference output (a help table, a usage report) that otherwise has nowhere
  to go but the main content stream. `Render` returns a frame of exactly the
  size it was given, so adopting one changes no layout math; `Align` pins it to
  any of nine anchors with a cell-wise nudge; `Help` adds host bindings to the
  hint row, which is always drawn — a modal with no visible way out is a trap.
- **`ActionRow`** — a labelled row of selectable actions (`Actions:  Start
  [Stop]  Restart`); the selected action is bracketed and highlighted when the
  row is focused, muted otherwise.
- **`Help`** / **`HelpLine`** — a `bubbles/help` model with brighter key and
  description colors than the dim bubbles default, plus a one-line short-help
  renderer.
- **`Meter`** — a fixed-width horizontal gauge (filled/empty bar, no percentage
  label) over `bubbles/progress`, clamped to 0–100. The CPU/RAM/disk dial every
  dashboard needs.
- **`Status`** — the "press again to confirm" destructive-action flow bundled
  with the success/error message it leaves behind: `Confirm` arms then fires,
  `SetResult` records the outcome, `AppendRows` renders it in the theme's colors.
- **`Table`** / **`Pairs`** — a header and `[][]string` body in, an aligned
  block out: widths measured over the header too, header upper-cased and muted.
  `Pairs` is the one-dimensional case, for key/value data. Both are built on
  `Columns`, `JoinCells`, `Pad` and `Widest`, which stay exported for layouts
  they do not cover. All measure with `ansi.StringWidth`, so a styled cell
  aligns like its plain equivalent instead of padding by the length of its
  escape sequence.
- **`Painter`** — `PainterFor(w)` decides once whether output may carry escapes
  (`w` is a terminal, `NO_COLOR` unset) and returns `Paint` or `Plain`, so no
  call site repeats that test — or gets it subtly different, which is how half a
  command's output ends up coloured. `Plain` also keeps test assertions readable.
- **`ClassifyStatus`** / **`StatusWord`** — grade a status word into a `Level`
  and paint it: green for a healthy state, red for a failure, amber for anything
  unfamiliar, so an operator only reads the words that are not green.
- **Layout & text helpers** — `Titleize` (`running_profiles` → `Running
  profiles`), `StatusTitle`, `Field`, `Rule`, `VerticalSlice`,
  `Flow`, `AdaptiveWidth` (responsive column width), `Indent`/`IndentLines`,
  `TruncMiddle` (rune-aware middle-ellipsis), `FormatBytes` (IEC sizes),
  `TransferredBytes` ("4.1 GiB / 6.6 GiB"), `CoarseDuration` (three significant
  figures), `Age` ("3m0s ago"), and `EmptyPanel` (placeholder).

## Usage

```go
frame := tuikit.New(
    tuikit.WithBrand("myapp", "does a thing"),
    tuikit.WithPages(newHomePage(), newSettingsPage()),
    tuikit.WithStatus(func() (string, tuikit.Level) { return "Ready", tuikit.LevelInfo }),
)
tea.NewProgram(frame).Run()
```

## Docs

- [docs/examples.md](docs/examples.md) — copy-paste snippets for every component.
- [scripts/record.py](scripts/record.py) — regenerates the gif and screenshot above.
- Package overview / API reference: `go doc github.com/antonikliment/tuikit`.

## Demo

```sh
go run ./examples/demo    # pages, tabs, reader, SearchView, ActionRow, Help, Meter/Status, Table/Pairs, Overlay
go run ./examples/streaming -seed=7 -cps=80 -block=3 -debug   # StreamingMarkdown under an endless generated stream
go run ./examples/themed  # live theme switching — press t to cycle palettes
```

Number keys switch pages; on the Panels page `Tab` switches sub-panels; on the
Search page `/` focuses the field (and digits then type instead of navigating); on
the Table page any key starts the simulated transfer; on the Overlay page `p`
pins the popup to the next anchor and `hjkl` nudges it.

## License

[MIT](LICENSE)
