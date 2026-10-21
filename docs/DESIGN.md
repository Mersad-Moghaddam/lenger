# Visual direction — Lenger Midnight

This document records the design decisions behind Lenger's interface. The PRD
remains the product contract; this is the interpretation used by the code.

## Concept

**A harbor instrument, not a dashboard template.** Lenger should feel like a
well-made bridge instrument: deep water, clear soundings, one warm signal light,
and nothing decorative moving when the vessel is at rest. The nautical metaphor
is structural, not a costume: depths are gauges, jobs are wakes, health is a
watch, and the anchor is the only wordmark ornament.

## Palette

The bundled `midnight` theme is the reference palette. Every other theme keeps
the same roles so the layout never depends on a particular hue.

| Role | Value | Use |
| --- | --- | --- |
| Deep water | `#0B1220` | terminal background |
| Hull | `#111D2E` | panels and chart plates |
| Deck | `#1A2940` | raised overlays and selected rows |
| Channel | `#3B506B` | idle borders and dividers |
| Foam | `#6DD6D0` | primary accent, active underline, progress |
| Signal | `#B4A1FF` | second gradient stop only |
| Lantern | `#F1CE86` | warning / degraded state |
| Paper | `#EDF3FF` | primary text |

Status colors are never used alone: every state carries a glyph and a word.
Large surfaces stay flat. Borders are one rounded weight; focus changes color,
not thickness. The accent gradient appears only on the wordmark, the active tab
underline and progress fills.

## Typography and voice

The terminal font is the typeface. Hierarchy comes from weight, spacing and
case, not from extra faces:

- The wordmark is the one display element: `⚓ LENGER`, tracked by its glyph.
- Section labels are spaced uppercase in the established PRD style.
- Data is tabular and right-aligned where comparison matters.
- Hints are muted and use plain verbs. An action keeps its name through the
  confirmation and the resulting message.

No synthetic marketing copy appears inside the app. Empty states say what is
missing and the next concrete action. Errors state what failed and what to do.

## Layout

- **Chart table.** The dashboard is asymmetric: health and attention in a narrow
  left watch, live data and soundings to the right. It is not a grid of identical
  cards.
- **Master/detail.** Images and containers keep the list stable on the left and
  put metadata on the right at wide sizes. At narrow sizes the detail replaces
  the list instead of squeezing both.
- **One composed frame.** Overlays sit on a dimmed backdrop with a single raised
  panel. No stacked modal chrome.
- **Footer as a chart legend.** Keys and job state share the bottom rule; the
  status line is a place for truth, not decoration.

## Signature graphics

- **Sounding gauge.** Disk usage is a measured depth, drawn with quarter marks
  and a fill rather than a generic progress bar. It answers "how full is this
  harbor?" without pretending unused image bytes are reclaimable disk.
- **Layer ribbon.** A stacked layer bar uses the theme series colors; the
  selected image's largest layer is visually dominant.
- **Wake.** Pull progress uses real layer states and known bytes. No invented ETA
  or synthetic throughput.
- **Anchor-and-waves empty state.** A small three-line mark appears only when a
  view has no resources; it never displaces the next action.

## Motion

Motion explains change, never fills silence:

- selection follows the cursor with a short spring;
- tab underline moves between tabs;
- progress fills ease while a job is active;
- toast enters and leaves once;
- idle schedules no frame ticks.

`reduce_motion`, `NO_MOTION=1`, SSH sessions and `TERM=dumb` resolve to instant
state changes. The current implementation schedules no decorative idle timers.

## Review against the brief

This direction deliberately avoids the generic dark-dashboard kit: no neon
green-on-black, no identical SaaS cards, no ornamental gradients, no badges
without meaning. The harbor vocabulary comes from the product's name and the
PRD's "anchor your containers" promise. The boldness is concentrated in the
sounding gauge and the wordmark; everything else stays quiet so data remains
legible at 80×24 and in ASCII.
