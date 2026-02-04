<template>
  <!--
    AppLogo (currentColor)
    - Use `class` on this component to control size, e.g. `w-4 h-4` / `w-8 h-8`.
    - The icon inherits color from CSS via `currentColor` (so it follows theme/text color).
  -->
  <svg
    viewBox="0 0 24 24"
    xmlns="http://www.w3.org/2000/svg"
    role="img"
    aria-label="HotSpot"
    preserveAspectRatio="xMidYMid meet"
    class="block"
  >
    <!--
      NOTE:
      This is a clean, generic mark designed for currentColor usage.
      If you later obtain your brand's real SVG paths, replace the shapes below
      while keeping:
      - `viewBox`
      - `preserveAspectRatio`
      - `fill="currentColor"` and/or `stroke="currentColor"`
    -->

    <!-- soft backdrop -->
    <circle cx="12" cy="12" r="10" fill="currentColor" opacity="0.14" />

    <!-- hotspot rings -->
    <path
      d="M12 6.5c2.9 0 5.25 2.35 5.25 5.25S14.9 17 12 17s-5.25-2.35-5.25-5.25S9.1 6.5 12 6.5zm0 2c-1.8 0-3.25 1.46-3.25 3.25S10.2 15 12 15s3.25-1.46 3.25-3.25S13.8 8.5 12 8.5z"
      fill="currentColor"
      fill-rule="evenodd"
      clip-rule="evenodd"
      opacity="0.9"
    />

    <!-- center dot -->
    <circle cx="12" cy="11.75" r="1.25" fill="currentColor" />

    <!-- location pin tail (hint) -->
    <path
      d="M12 13.2c-1.6 0-2.9 1.26-2.9 2.82 0 1.64 1.16 3.25 2.33 4.56.3.33.83.33 1.13 0 1.17-1.31 2.34-2.92 2.34-4.56 0-1.56-1.3-2.82-2.9-2.82zm0 1.65c.66 0 1.2.52 1.2 1.17 0 .64-.54 1.16-1.2 1.16s-1.2-.52-1.2-1.16c0-.65.54-1.17 1.2-1.17z"
      fill="currentColor"
      opacity="0.9"
    />
  </svg>
</template>
