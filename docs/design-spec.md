# GoSX 3D Studio scaffold design contract

Status: initial implementation contract
Product: standalone GoSX Desktop scene, animation, game, and simulation workbench

## Visual System

### Territory

**Industrial Void** combines Dark Elegance with an industrial precision-tool
grammar. The interface feels physical; the scene feels infinite. Warm graphite
application chrome surrounds a near-black viewport. Authored state is orange,
observed runtime state is cyan, and trusted certification is gold.

### Typography

- Display: Space Grotesk, 600 and 700.
- Body: Work Sans, 400, 500, and 600.
- Mono: JetBrains Mono, 500.
- Scale: Minor Third (1.2), optimized for a compact technical UI.

| Token | Size |
|---|---|
| `--type-xs` | `clamp(0.6875rem, 0.66rem + 0.08vw, 0.75rem)` |
| `--type-sm` | `clamp(0.75rem, 0.72rem + 0.10vw, 0.8125rem)` |
| `--type-md` | `clamp(0.875rem, 0.84rem + 0.12vw, 0.9375rem)` |
| `--type-lg` | `clamp(1rem, 0.94rem + 0.18vw, 1.125rem)` |
| `--type-xl` | `clamp(1.2rem, 1.08rem + 0.30vw, 1.44rem)` |
| `--type-2xl` | `clamp(1.44rem, 1.24rem + 0.50vw, 1.728rem)` |

### Color architecture

- Dominant 60%: viewport/canvas `#0b0d10`.
- Secondary 30%: panels `#14181c` and raised controls `#23272c`.
- Accent 10%: authoring orange `#ff8a2a`, with cyan and gold reserved for
  semantic runtime and certification states.
- Primary text `#e6e2d6` has approximately 14.4:1 contrast on the dominant
  canvas (WCAG AAA).
- Secondary text `#b9b4a8` has approximately 9.2:1 contrast (WCAG AAA).
- Muted text `#918d84` has approximately 6.0:1 contrast (WCAG AA).

Orange means authored selection or manipulation. Cyan means observed runtime
telemetry. Gold means validated or certified output. These colors are not
interchangeable decoration.

### Motion

Minimal motion: 140 ms for hover/focus and 200 ms for panel/state changes.
Use `cubic-bezier(0.16, 1, 0.3, 1)` for ease-out and
`cubic-bezier(0.34, 1.56, 0.64, 1)` only for short tactile feedback. No ambient
interface animation. Honor reduced-motion preferences.

### Spacing

Dense 4 px base scale:

- `--space-xs: clamp(0.25rem, 0.22rem + 0.08vw, 0.375rem)`
- `--space-sm: clamp(0.5rem, 0.46rem + 0.10vw, 0.625rem)`
- `--space-md: clamp(0.75rem, 0.68rem + 0.16vw, 0.875rem)`
- `--space-lg: clamp(1rem, 0.90rem + 0.24vw, 1.25rem)`
- `--space-xl: clamp(1.5rem, 1.32rem + 0.40vw, 1.875rem)`
- `--space-2xl: clamp(2rem, 1.72rem + 0.65vw, 2.5rem)`
- `--space-3xl: clamp(3rem, 2.5rem + 1vw, 4rem)`

### Canonical tokens

```css
:root {
  --font-display: "Space Grotesk", "Aptos Display", sans-serif;
  --font-body: "Work Sans", "Aptos", sans-serif;
  --font-mono: "JetBrains Mono", "Cascadia Mono", monospace;

  --type-xs: clamp(0.6875rem, 0.66rem + 0.08vw, 0.75rem);
  --type-sm: clamp(0.75rem, 0.72rem + 0.10vw, 0.8125rem);
  --type-md: clamp(0.875rem, 0.84rem + 0.12vw, 0.9375rem);
  --type-lg: clamp(1rem, 0.94rem + 0.18vw, 1.125rem);
  --type-xl: clamp(1.2rem, 1.08rem + 0.30vw, 1.44rem);
  --type-2xl: clamp(1.44rem, 1.24rem + 0.50vw, 1.728rem);

  --color-canvas: #0b0d10;
  --color-panel: #14181c;
  --color-panel-raised: #23272c;
  --color-panel-sunken: #101317;
  --color-border: #2d3239;
  --color-border-strong: #454c55;
  --color-text: #e6e2d6;
  --color-text-secondary: #b9b4a8;
  --color-text-muted: #918d84;
  --color-author: #ff8a2a;
  --color-author-soft: #5a321c;
  --color-runtime: #2ec7e6;
  --color-runtime-soft: #173b43;
  --color-certified: #d4af37;
  --color-warning: #f2b134;
  --color-error: #e05252;
  --color-success: #4dbb78;
  --color-focus: #f6a45e;
  --color-shadow: rgba(0, 0, 0, 0.42);

  --space-xs: clamp(0.25rem, 0.22rem + 0.08vw, 0.375rem);
  --space-sm: clamp(0.5rem, 0.46rem + 0.10vw, 0.625rem);
  --space-md: clamp(0.75rem, 0.68rem + 0.16vw, 0.875rem);
  --space-lg: clamp(1rem, 0.90rem + 0.24vw, 1.25rem);
  --space-xl: clamp(1.5rem, 1.32rem + 0.40vw, 1.875rem);
  --space-2xl: clamp(2rem, 1.72rem + 0.65vw, 2.5rem);
  --space-3xl: clamp(3rem, 2.5rem + 1vw, 4rem);

  --duration-fast: 140ms;
  --duration-panel: 200ms;
  --ease-out: cubic-bezier(0.16, 1, 0.3, 1);
  --ease-spring: cubic-bezier(0.34, 1.56, 0.64, 1);
}
```

## Scaffold boundary

The first scaffold presents application regions, semantic states, and package
seams. It does not claim that modeling, rigging, simulation, Selena compilation,
or certification actions are implemented. Mock values are visibly labeled as
scaffold data, and the viewport is a placeholder for the shared Scene3D engine.
