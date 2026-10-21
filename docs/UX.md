# UI implementation guide

Visual requirements and interaction contracts live in
[PRD section 7](PRD.md#7-ux-and-visual-design).

## Tokens and components

- Add geometry constants to `internal/ui/tokens`, not view-local padding.
- Views reference `tokens.Palette` roles, never literal colors.
- `style.Sheet` is passed by value; it carries plain/ASCII capabilities and the
  current semantic palette. No package-level mutable style state is needed.
- Use `Sheet.Panel` for bounded resource surfaces and `components.Modal` for
  compositor overlays. Use `style.Clean` on all daemon/user-originated strings
  before styling them. `style.Fit` clips trusted styled content to a rectangle.
- Gradients belong only on the wordmark, tab underline and progress fills.
  `theme.Blend` interpolates in OKLab.
- Meaning uses words and glyphs, not color alone. `NO_COLOR` disables colored
  application styling. `ascii_icons` selects a complete ASCII status vocabulary.
- Real telemetry is never fabricated. Fake-mode charts are labeled fixture data.

## Implemented interactions

`1`–`7` changes tabs. Images supports `j/k`, arrows, space selection,
shift-arrow range extension, `/` filtering, `o` sorting, `i`/Enter inspection,
and `d` for a non-forced removal preview. The preview uses `y` to confirm.

`p` opens a validated image-reference prompt. Enter begins an asynchronous pull.
Esc hides it without cancelling; `J` opens session job history, Enter opens the
most recent job, `x` cancels its pull, and `R` retries a finished pull.

`:` or Ctrl+P opens fuzzy action/resource search with a command preview.
`?` shows implemented bindings. `q` or Ctrl+C exits and cancels session work.

Theme defaults to Omarchy when available, with a Midnight fallback. Explicit
`--theme` overrides the configured choice. Theme watching is a desktop file
watcher poll, separate from animation scheduling.

## Deferred polish

- Springs are implemented as finite primitives, but selection glide, animated
  tab underline, count-up, toast movement and overlay fade are not wired yet.
  No decorative heartbeat or unused animation ticks are scheduled.
- Narrow detail replaces the list. Wide layout reserves a third pane for M2
  telemetry, explicitly labeled unavailable rather than showing fake live data.
- Detail content is clipped rather than scrollable; a viewport/inspect tree is
  still required for images with extensive metadata.
- Palette match highlighting, recents, persisted filters, full remapping,
  scrollbar dragging, Nerd Font detection and density-specific geometry remain.
- Golden frames and recordings have not been generated or visually reviewed.
