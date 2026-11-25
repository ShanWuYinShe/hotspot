// 自动生成，勿手改；重新生成：node scripts/design-tokens/gen-android.mjs
// 数据源：frontend/src/styles/theme/（CSS token 是唯一权威，校验脚本见
// frontend/scripts/check-cross-platform-android.mjs，漂移即 CI 红）。
// 颜色注释里的 token 名与 Web 端一一对应；颜色值为 #AARRGGBB 的 Compose Color 入参。
package com.hotspot.app.ui.theme

import androidx.compose.animation.core.CubicBezierEasing
import androidx.compose.ui.graphics.Color

/** Web 端设计 token 的 Compose 镜像；由 scripts/design-tokens/gen-android.mjs 生成。 */
object Tokens {
    object Palette {
        /** Vibrant · light */
        object VibrantLight : PaletteColors {
            override val accentPrimary: Color = Color(0xFF2B5EB2) // --cst-palette-accent-primary
            override val accentSoft: Color = Color(0x1A2B5EB2) // --cst-palette-accent-soft
            override val accentSoftBorder: Color = Color(0x472B5EB2) // --cst-palette-accent-soft-border
            override val activeHover: Color = Color(0x292B5EB2) // --cst-palette-active-hover
            override val bg: Color = Color(0xFFFFFFFF) // --cst-palette-bg
            override val border: Color = Color(0x2E2B5EB2) // --cst-palette-border
            override val borderPressed: Color = Color(0x522B5EB2) // --cst-palette-border-pressed
            override val chatBg: Color = Color(0xFFEDF3FC) // --cst-palette-chat-bg
            override val glow: Color = Color(0x00000000) // --cst-palette-glow
            override val hot: Color = Color(0xFFC81E4E) // --cst-palette-hot
            override val hover: Color = Color(0x1A2B5EB2) // --cst-palette-hover
            override val mapGroup: Color = Color(0xFF0B7653) // --cst-palette-map-group
            override val mapMoment: Color = Color(0xFFC2470B) // --cst-palette-map-moment
            override val mapUser: Color = Color(0xFF2B5EB2) // --cst-palette-map-user
            override val onAccent: Color = Color(0xFFFFFFFF) // --cst-palette-on-accent
            override val online: Color = Color(0xFF0D6A62) // --cst-palette-online
            override val overlayStrong: Color = Color(0x850C1930) // --cst-palette-overlay-strong
            override val pageBg: Color = Color(0xFFEFF3F9) // --cst-palette-page-bg
            override val shadowDark: Color = Color(0x14000000) // --cst-palette-shadow-dark
            override val shadowLight: Color = Color(0x00FFFFFF) // --cst-palette-shadow-light
            override val text: Color = Color(0xFF16233A) // --cst-palette-text
            override val textMuted: Color = Color(0xFF47586F) // --cst-palette-text-muted
            override val warning: Color = Color(0xFF9C4708) // --cst-palette-warning
        }
        /** Vibrant · dark */
        object VibrantDark : PaletteColors {
            override val accentPrimary: Color = Color(0xFF5B9CF6) // --cst-palette-accent-primary
            override val accentSoft: Color = Color(0x1F5B9CF6) // --cst-palette-accent-soft
            override val accentSoftBorder: Color = Color(0x425B9CF6) // --cst-palette-accent-soft-border
            override val activeHover: Color = Color(0x2E5B9CF6) // --cst-palette-active-hover
            override val bg: Color = Color(0xFF121622) // --cst-palette-bg
            override val border: Color = Color(0x1CA0B9DC) // --cst-palette-border
            override val borderPressed: Color = Color(0x38A0B9DC) // --cst-palette-border-pressed
            override val chatBg: Color = Color(0xFF0E121B) // --cst-palette-chat-bg
            override val glow: Color = Color(0x00000000) // --cst-palette-glow
            override val hot: Color = Color(0xFFF87171) // --cst-palette-hot
            override val hover: Color = Color(0x1F5B9CF6) // --cst-palette-hover
            override val mapGroup: Color = Color(0xFF34D399) // --cst-palette-map-group
            override val mapMoment: Color = Color(0xFFFB923C) // --cst-palette-map-moment
            override val mapUser: Color = Color(0xFF5B9CF6) // --cst-palette-map-user
            override val onAccent: Color = Color(0xFF0B0E14) // --cst-palette-on-accent
            override val online: Color = Color(0xFF34D399) // --cst-palette-online
            override val overlayStrong: Color = Color(0xA600040C) // --cst-palette-overlay-strong
            override val pageBg: Color = Color(0xFF0B0E14) // --cst-palette-page-bg
            override val shadowDark: Color = Color(0xBF000000) // --cst-palette-shadow-dark
            override val shadowLight: Color = Color(0x0D94A3B8) // --cst-palette-shadow-light
            override val text: Color = Color(0xFFF1F5F9) // --cst-palette-text
            override val textMuted: Color = Color(0xFFA3B1C4) // --cst-palette-text-muted
            override val warning: Color = Color(0xFFFBBF24) // --cst-palette-warning
        }
        /** Forest · light */
        object ForestLight : PaletteColors {
            override val accentPrimary: Color = Color(0xFF39714F) // --cst-palette-accent-primary
            override val accentSoft: Color = Color(0x1C39714F) // --cst-palette-accent-soft
            override val accentSoftBorder: Color = Color(0x4D39714F) // --cst-palette-accent-soft-border
            override val activeHover: Color = Color(0x2B39714F) // --cst-palette-active-hover
            override val bg: Color = Color(0xFFF6F7F3) // --cst-palette-bg
            override val border: Color = Color(0x3D2A4934) // --cst-palette-border
            override val borderPressed: Color = Color(0x612A4934) // --cst-palette-border-pressed
            override val chatBg: Color = Color(0xFFF2F3F2) // --cst-palette-chat-bg
            override val glow: Color = Color(0x00000000) // --cst-palette-glow
            override val hot: Color = Color(0xFFA04B37) // --cst-palette-hot
            override val hover: Color = Color(0x1F39714F) // --cst-palette-hover
            override val mapGroup: Color = Color(0xFF4A703F) // --cst-palette-map-group
            override val mapMoment: Color = Color(0xFF8A6426) // --cst-palette-map-moment
            override val mapUser: Color = Color(0xFF39714F) // --cst-palette-map-user
            override val onAccent: Color = Color(0xFFFFFFFF) // --cst-palette-on-accent
            override val online: Color = Color(0xFF39714F) // --cst-palette-online
            override val overlayStrong: Color = Color(0x6B141C14) // --cst-palette-overlay-strong
            override val pageBg: Color = Color(0xFFF6F7F3) // --cst-palette-page-bg
            override val shadowDark: Color = Color(0x00000000) // --cst-palette-shadow-dark
            override val shadowLight: Color = Color(0x00FFFFFF) // --cst-palette-shadow-light
            override val text: Color = Color(0xFF152219) // --cst-palette-text
            override val textMuted: Color = Color(0xFF354B3D) // --cst-palette-text-muted
            override val warning: Color = Color(0xFF7E602B) // --cst-palette-warning
        }
        /** Forest · dark */
        object ForestDark : PaletteColors {
            override val accentPrimary: Color = Color(0xFF52B878) // --cst-palette-accent-primary
            override val accentSoft: Color = Color(0x1F52B878) // --cst-palette-accent-soft
            override val accentSoftBorder: Color = Color(0x5752B878) // --cst-palette-accent-soft-border
            override val activeHover: Color = Color(0x3D52B878) // --cst-palette-active-hover
            override val bg: Color = Color(0xFF161A16) // --cst-palette-bg
            override val border: Color = Color(0x2EA9C7B2) // --cst-palette-border
            override val borderPressed: Color = Color(0x4DA9C7B2) // --cst-palette-border-pressed
            override val chatBg: Color = Color(0xFF0E100E) // --cst-palette-chat-bg
            override val glow: Color = Color(0x00000000) // --cst-palette-glow
            override val hot: Color = Color(0xFFD97C5A) // --cst-palette-hot
            override val hover: Color = Color(0x2E52B878) // --cst-palette-hover
            override val mapGroup: Color = Color(0xFF88B85A) // --cst-palette-map-group
            override val mapMoment: Color = Color(0xFFC4A75A) // --cst-palette-map-moment
            override val mapUser: Color = Color(0xFF52B878) // --cst-palette-map-user
            override val onAccent: Color = Color(0xFF0C1911) // --cst-palette-on-accent
            override val online: Color = Color(0xFF52B878) // --cst-palette-online
            override val overlayStrong: Color = Color(0x8A000000) // --cst-palette-overlay-strong
            override val pageBg: Color = Color(0xFF161A16) // --cst-palette-page-bg
            override val shadowDark: Color = Color(0x00000000) // --cst-palette-shadow-dark
            override val shadowLight: Color = Color(0x00FFFFFF) // --cst-palette-shadow-light
            override val text: Color = Color(0xFFE8F1E9) // --cst-palette-text
            override val textMuted: Color = Color(0xFFB7CDBD) // --cst-palette-text-muted
            override val warning: Color = Color(0xFFC4A75A) // --cst-palette-warning
        }
        /** Inkwash · light */
        object InkwashLight : PaletteColors {
            override val accentPrimary: Color = Color(0xFFA63A2E) // --cst-palette-accent-primary
            override val accentSoft: Color = Color(0x1AA63A2E) // --cst-palette-accent-soft
            override val accentSoftBorder: Color = Color(0x47A63A2E) // --cst-palette-accent-soft-border
            override val activeHover: Color = Color(0x2BA63A2E) // --cst-palette-active-hover
            override val bg: Color = Color(0xFFFBF9F3) // --cst-palette-bg
            override val border: Color = Color(0x332B2A26) // --cst-palette-border
            override val borderPressed: Color = Color(0x522B2A26) // --cst-palette-border-pressed
            override val chatBg: Color = Color(0xFFF1EDE3) // --cst-palette-chat-bg
            override val glow: Color = Color(0x00000000) // --cst-palette-glow
            override val hot: Color = Color(0xFF9E2140) // --cst-palette-hot
            override val hover: Color = Color(0x1FA63A2E) // --cst-palette-hover
            override val mapGroup: Color = Color(0xFF4A7048) // --cst-palette-map-group
            override val mapMoment: Color = Color(0xFF8A6426) // --cst-palette-map-moment
            override val mapUser: Color = Color(0xFFA63A2E) // --cst-palette-map-user
            override val onAccent: Color = Color(0xFFFFFFFF) // --cst-palette-on-accent
            override val online: Color = Color(0xFF3F7256) // --cst-palette-online
            override val overlayStrong: Color = Color(0x6B1A1814) // --cst-palette-overlay-strong
            override val pageBg: Color = Color(0xFFF5F2EA) // --cst-palette-page-bg
            override val shadowDark: Color = Color(0x00000000) // --cst-palette-shadow-dark
            override val shadowLight: Color = Color(0x00FFFFFF) // --cst-palette-shadow-light
            override val text: Color = Color(0xFF2B2A26) // --cst-palette-text
            override val textMuted: Color = Color(0xFF5D594F) // --cst-palette-text-muted
            override val warning: Color = Color(0xFF7F6422) // --cst-palette-warning
        }
        /** Inkwash · dark */
        object InkwashDark : PaletteColors {
            override val accentPrimary: Color = Color(0xFFDF7F6A) // --cst-palette-accent-primary
            override val accentSoft: Color = Color(0x1FDF7F6A) // --cst-palette-accent-soft
            override val accentSoftBorder: Color = Color(0x4DDF7F6A) // --cst-palette-accent-soft-border
            override val activeHover: Color = Color(0x3DDF7F6A) // --cst-palette-active-hover
            override val bg: Color = Color(0xFF212426) // --cst-palette-bg
            override val border: Color = Color(0x29EAE6DA) // --cst-palette-border
            override val borderPressed: Color = Color(0x47EAE6DA) // --cst-palette-border-pressed
            override val chatBg: Color = Color(0xFF1D2022) // --cst-palette-chat-bg
            override val glow: Color = Color(0x00000000) // --cst-palette-glow
            override val hot: Color = Color(0xFFDE8496) // --cst-palette-hot
            override val hover: Color = Color(0x2EDF7F6A) // --cst-palette-hover
            override val mapGroup: Color = Color(0xFF7FA88B) // --cst-palette-map-group
            override val mapMoment: Color = Color(0xFFC89B5A) // --cst-palette-map-moment
            override val mapUser: Color = Color(0xFFDF7F6A) // --cst-palette-map-user
            override val onAccent: Color = Color(0xFF1F1512) // --cst-palette-on-accent
            override val online: Color = Color(0xFF82B297) // --cst-palette-online
            override val overlayStrong: Color = Color(0x94000000) // --cst-palette-overlay-strong
            override val pageBg: Color = Color(0xFF181A1C) // --cst-palette-page-bg
            override val shadowDark: Color = Color(0x99000000) // --cst-palette-shadow-dark
            override val shadowLight: Color = Color(0x00FFFFFF) // --cst-palette-shadow-light
            override val text: Color = Color(0xFFEAE6DA) // --cst-palette-text
            override val textMuted: Color = Color(0xFFB7B3A8) // --cst-palette-text-muted
            override val warning: Color = Color(0xFFD6A54E) // --cst-palette-warning
        }
    }

    object Semantic {
        /** Vibrant · light */
        object VibrantLight : SemanticPalette {
            override val accentBorder: Color = Color(0x472B5EB2) // --cst-accent-border
            override val accentBorderSoft: Color = Color(0x402B5EB2) // --cst-accent-border-soft
            override val accentDefault: Color = Color(0xFF2B5EB2) // --cst-accent-default
            override val accentSoft: Color = Color(0x1A2B5EB2) // --cst-accent-soft
            override val accentSoftBg: Color = Color(0x1A2B5EB2) // --cst-accent-soft-bg
            override val borderDefault: Color = Color(0x2E2B5EB2) // --cst-border-default
            override val borderStrong: Color = Color(0x522B5EB2) // --cst-border-strong
            override val borderSubtle: Color = Color(0x2E2B5EB2) // --cst-border-subtle
            override val contentInverse: Color = Color(0xFFFFFFFF) // --cst-content-inverse
            override val contentInverseMuted: Color = Color(0xB3FFFFFF) // --cst-content-inverse-muted
            override val contentInverseSecondary: Color = Color(0xCCFFFFFF) // --cst-content-inverse-secondary
            override val contentOnMarker: Color = Color(0xFFFFFFFF) // --cst-content-on-marker
            override val fieldBg: Color = Color(0xFFFFFFFF) // --cst-field-bg
            override val fieldBgDisabled: Color = Color(0xFFECEDEF) // --cst-field-bg-disabled
            override val fieldBorder: Color = Color(0x2E2B5EB2) // --cst-field-border
            override val fieldBorderFocus: Color = Color(0xFF2B5EB2) // --cst-field-border-focus
            override val fieldBorderHover: Color = Color(0x522B5EB2) // --cst-field-border-hover
            override val fieldRing: Color = Color(0x382B5EB2) // --cst-field-ring
            override val fieldRingDanger: Color = Color(0x3DC81E4E) // --cst-field-ring-danger
            override val glowAccent: Color = Color(0x00000000) // --cst-glow-accent
            override val glowDanger: Color = Color(0x1F000000) // --cst-glow-danger
            override val interactiveActive: Color = Color(0x292B5EB2) // --cst-interactive-active
            override val interactiveActiveFill: Color = Color(0x292B5EB2) // --cst-interactive-active-fill
            override val interactiveHover: Color = Color(0x1A2B5EB2) // --cst-interactive-hover
            override val interactiveHoverFill: Color = Color(0x1A2B5EB2) // --cst-interactive-hover-fill
            override val markerGroup: Color = Color(0xFF0B7653) // --cst-marker-group
            override val markerGroupBg: Color = Color(0x210B7653) // --cst-marker-group-bg
            override val markerGroupBorder: Color = Color(0x3D0B7653) // --cst-marker-group-border
            override val markerGroupGlow: Color = Color(0x730B7653) // --cst-marker-group-glow
            override val markerMoment: Color = Color(0xFFC2470B) // --cst-marker-moment
            override val markerMomentBg: Color = Color(0x21C2470B) // --cst-marker-moment-bg
            override val markerMomentBorder: Color = Color(0x3DC2470B) // --cst-marker-moment-border
            override val markerMomentGlow: Color = Color(0x73C2470B) // --cst-marker-moment-glow
            override val markerUser: Color = Color(0xFF2B5EB2) // --cst-marker-user
            override val markerUserBg: Color = Color(0x212B5EB2) // --cst-marker-user-bg
            override val markerUserBorder: Color = Color(0x3D2B5EB2) // --cst-marker-user-border
            override val markerUserGlow: Color = Color(0x732B5EB2) // --cst-marker-user-glow
            override val navigationActiveIndicator: Color = Color(0xFF2B5EB2) // --cst-navigation-active-indicator
            override val navigationContent: Color = Color(0xFF47586F) // --cst-navigation-content
            override val navigationContentActive: Color = Color(0xFF2B5EB2) // --cst-navigation-content-active
            override val statusDanger: Color = Color(0xFFC81E4E) // --cst-status-danger
            override val statusDangerBg: Color = Color(0x21C81E4E) // --cst-status-danger-bg
            override val statusDangerBorder: Color = Color(0x3DC81E4E) // --cst-status-danger-border
            override val statusOnline: Color = Color(0xFF0D6A62) // --cst-status-online
            override val statusOnlineBg: Color = Color(0x210D6A62) // --cst-status-online-bg
            override val statusOnlineBorder: Color = Color(0x3D0D6A62) // --cst-status-online-border
            override val statusWarning: Color = Color(0xFF9C4708) // --cst-status-warning
            override val statusWarningBg: Color = Color(0x219C4708) // --cst-status-warning-bg
            override val statusWarningBorder: Color = Color(0x3D9C4708) // --cst-status-warning-border
            override val surfaceCardMenu: Color = Color(0xFFFFFFFF) // --cst-surface-card-menu
            override val surfaceChat: Color = Color(0xFFEDF3FC) // --cst-surface-chat
            override val surfaceGlassBadge: Color = Color(0x2EFFFFFF) // --cst-surface-glass-badge
            override val surfaceIsland: Color = Color(0xDBFFFFFF) // --cst-surface-island
            override val surfaceMediaScrimStrong: Color = Color(0x8C000000) // --cst-surface-media-scrim-strong
            override val surfaceNavGlass: Color = Color(0xE6FFFFFF) // --cst-surface-nav-glass
            override val surfaceNavigationActive: Color = Color(0xFFBCCCE7) // --cst-surface-navigation-active
            override val surfaceNavigationActiveHover: Color = Color(0xFFACC0E1) // --cst-surface-navigation-active-hover
            override val surfaceNavigationHover: Color = Color(0xFFD5DAE2) // --cst-surface-navigation-hover
            override val surfaceOverlay: Color = Color(0x610E0A23) // --cst-surface-overlay
            override val surfaceOverlayStrong: Color = Color(0x850C1930) // --cst-surface-overlay-strong
            override val surfacePage: Color = Color(0xFFEFF3F9) // --cst-surface-page
            override val surfacePanel: Color = Color(0xFFFFFFFF) // --cst-surface-panel
            override val surfacePrimary: Color = Color(0xFFFFFFFF) // --cst-surface-primary
            override val surfaceRaised: Color = Color(0xFFFFFFFF) // --cst-surface-raised
            override val surfaceScrim: Color = Color(0x610E0A23) // --cst-surface-scrim
            override val surfaceSubtle: Color = Color(0xFFECEDEF) // --cst-surface-subtle
            override val surfaceToggleOff: Color = Color(0x38787880) // --cst-surface-toggle-off
            override val surfaceToggleThumb: Color = Color(0xFFFFFFFF) // --cst-surface-toggle-thumb
            override val textAccentOnSoft: Color = Color(0xFF27529A) // --cst-text-accent-on-soft
            override val textOnAccent: Color = Color(0xFFFFFFFF) // --cst-text-on-accent
            override val textOnStatus: Color = Color(0xFF0B1424) // --cst-text-on-status
            override val textPrimary: Color = Color(0xFF16233A) // --cst-text-primary
            override val textSecondary: Color = Color(0xFF47586F) // --cst-text-secondary
            override val textStatusDangerOnSoft: Color = Color(0xFFA41F4A) // --cst-text-status-danger-on-soft
            override val textStatusOnlineOnSoft: Color = Color(0xFF0F5C5A) // --cst-text-status-online-on-soft
            override val textStatusWarningOnSoft: Color = Color(0xFF814012) // --cst-text-status-warning-on-soft
        }
        /** Vibrant · dark */
        object VibrantDark : SemanticPalette {
            override val accentBorder: Color = Color(0x425B9CF6) // --cst-accent-border
            override val accentBorderSoft: Color = Color(0x4D5B9CF6) // --cst-accent-border-soft
            override val accentDefault: Color = Color(0xFF5B9CF6) // --cst-accent-default
            override val accentSoft: Color = Color(0x1F5B9CF6) // --cst-accent-soft
            override val accentSoftBg: Color = Color(0x1F5B9CF6) // --cst-accent-soft-bg
            override val borderDefault: Color = Color(0x1CA0B9DC) // --cst-border-default
            override val borderStrong: Color = Color(0x38A0B9DC) // --cst-border-strong
            override val borderSubtle: Color = Color(0x1CA0B9DC) // --cst-border-subtle
            override val contentInverse: Color = Color(0xFFFFFFFF) // --cst-content-inverse
            override val contentInverseMuted: Color = Color(0xB3FFFFFF) // --cst-content-inverse-muted
            override val contentInverseSecondary: Color = Color(0xCCFFFFFF) // --cst-content-inverse-secondary
            override val contentOnMarker: Color = Color(0xFF071426) // --cst-content-on-marker
            override val fieldBg: Color = Color(0xFF121622) // --cst-field-bg
            override val fieldBgDisabled: Color = Color(0xFF242833) // --cst-field-bg-disabled
            override val fieldBorder: Color = Color(0x1CA0B9DC) // --cst-field-border
            override val fieldBorderFocus: Color = Color(0xFF5B9CF6) // --cst-field-border-focus
            override val fieldBorderHover: Color = Color(0x38A0B9DC) // --cst-field-border-hover
            override val fieldRing: Color = Color(0x385B9CF6) // --cst-field-ring
            override val fieldRingDanger: Color = Color(0x3DF87171) // --cst-field-ring-danger
            override val glowAccent: Color = Color(0x00000000) // --cst-glow-accent
            override val glowDanger: Color = Color(0x1F000000) // --cst-glow-danger
            override val interactiveActive: Color = Color(0x2E5B9CF6) // --cst-interactive-active
            override val interactiveActiveFill: Color = Color(0x2E5B9CF6) // --cst-interactive-active-fill
            override val interactiveHover: Color = Color(0x1F5B9CF6) // --cst-interactive-hover
            override val interactiveHoverFill: Color = Color(0x1F5B9CF6) // --cst-interactive-hover-fill
            override val markerGroup: Color = Color(0xFF34D399) // --cst-marker-group
            override val markerGroupBg: Color = Color(0x2434D399) // --cst-marker-group-bg
            override val markerGroupBorder: Color = Color(0x4D34D399) // --cst-marker-group-border
            override val markerGroupGlow: Color = Color(0x8034D399) // --cst-marker-group-glow
            override val markerMoment: Color = Color(0xFFFB923C) // --cst-marker-moment
            override val markerMomentBg: Color = Color(0x24FB923C) // --cst-marker-moment-bg
            override val markerMomentBorder: Color = Color(0x4DFB923C) // --cst-marker-moment-border
            override val markerMomentGlow: Color = Color(0x80FB923C) // --cst-marker-moment-glow
            override val markerUser: Color = Color(0xFF5B9CF6) // --cst-marker-user
            override val markerUserBg: Color = Color(0x245B9CF6) // --cst-marker-user-bg
            override val markerUserBorder: Color = Color(0x4D5B9CF6) // --cst-marker-user-border
            override val markerUserGlow: Color = Color(0x805B9CF6) // --cst-marker-user-glow
            override val navigationActiveIndicator: Color = Color(0xFF5B9CF6) // --cst-navigation-active-indicator
            override val navigationContent: Color = Color(0xFFA3B1C4) // --cst-navigation-content
            override val navigationContentActive: Color = Color(0xFF5B9CF6) // --cst-navigation-content-active
            override val statusDanger: Color = Color(0xFFF87171) // --cst-status-danger
            override val statusDangerBg: Color = Color(0x24F87171) // --cst-status-danger-bg
            override val statusDangerBorder: Color = Color(0x47F87171) // --cst-status-danger-border
            override val statusOnline: Color = Color(0xFF34D399) // --cst-status-online
            override val statusOnlineBg: Color = Color(0x2434D399) // --cst-status-online-bg
            override val statusOnlineBorder: Color = Color(0x4734D399) // --cst-status-online-border
            override val statusWarning: Color = Color(0xFFFBBF24) // --cst-status-warning
            override val statusWarningBg: Color = Color(0x24FBBF24) // --cst-status-warning-bg
            override val statusWarningBorder: Color = Color(0x47FBBF24) // --cst-status-warning-border
            override val surfaceCardMenu: Color = Color(0xFF121622) // --cst-surface-card-menu
            override val surfaceChat: Color = Color(0xFF0E121B) // --cst-surface-chat
            override val surfaceGlassBadge: Color = Color(0x2EFFFFFF) // --cst-surface-glass-badge
            override val surfaceIsland: Color = Color(0xFF181E2B) // --cst-surface-island
            override val surfaceMediaScrimStrong: Color = Color(0x8C000000) // --cst-surface-media-scrim-strong
            override val surfaceNavGlass: Color = Color(0xE6121622) // --cst-surface-nav-glass
            override val surfaceNavigationActive: Color = Color(0xFF20334F) // --cst-surface-navigation-active
            override val surfaceNavigationActiveHover: Color = Color(0xFF263E61) // --cst-surface-navigation-active-hover
            override val surfaceNavigationHover: Color = Color(0xFF272A2F) // --cst-surface-navigation-hover
            override val surfaceOverlay: Color = Color(0x6B000000) // --cst-surface-overlay
            override val surfaceOverlayStrong: Color = Color(0xA600040C) // --cst-surface-overlay-strong
            override val surfacePage: Color = Color(0xFF0B0E14) // --cst-surface-page
            override val surfacePanel: Color = Color(0xFF121622) // --cst-surface-panel
            override val surfacePrimary: Color = Color(0xFF121622) // --cst-surface-primary
            override val surfaceRaised: Color = Color(0xFF121622) // --cst-surface-raised
            override val surfaceScrim: Color = Color(0x6B000000) // --cst-surface-scrim
            override val surfaceSubtle: Color = Color(0xFF242833) // --cst-surface-subtle
            override val surfaceToggleOff: Color = Color(0x38787880) // --cst-surface-toggle-off
            override val surfaceToggleThumb: Color = Color(0xFF121622) // --cst-surface-toggle-thumb
            override val textAccentOnSoft: Color = Color(0xFF79AEF7) // --cst-text-accent-on-soft
            override val textOnAccent: Color = Color(0xFF0B0E14) // --cst-text-on-accent
            override val textOnStatus: Color = Color(0xFF071426) // --cst-text-on-status
            override val textPrimary: Color = Color(0xFFF1F5F9) // --cst-text-primary
            override val textSecondary: Color = Color(0xFFA3B1C4) // --cst-text-secondary
            override val textStatusDangerOnSoft: Color = Color(0xFFF78B8C) // --cst-text-status-danger-on-soft
            override val textStatusOnlineOnSoft: Color = Color(0xFF5ADAAC) // --cst-text-status-online-on-soft
            override val textStatusWarningOnSoft: Color = Color(0xFFF9CA4F) // --cst-text-status-warning-on-soft
        }
        /** Forest · light */
        object ForestLight : SemanticPalette {
            override val accentBorder: Color = Color(0x4D39714F) // --cst-accent-border
            override val accentBorderSoft: Color = Color(0x3D39714F) // --cst-accent-border-soft
            override val accentDefault: Color = Color(0xFF39714F) // --cst-accent-default
            override val accentSoft: Color = Color(0x1C39714F) // --cst-accent-soft
            override val accentSoftBg: Color = Color(0x1A39714F) // --cst-accent-soft-bg
            override val borderDefault: Color = Color(0x3D2A4934) // --cst-border-default
            override val borderStrong: Color = Color(0x612A4934) // --cst-border-strong
            override val borderSubtle: Color = Color(0x3D2A4934) // --cst-border-subtle
            override val contentInverse: Color = Color(0xFFFFFFFF) // --cst-content-inverse
            override val contentInverseMuted: Color = Color(0xB3FFFFFF) // --cst-content-inverse-muted
            override val contentInverseSecondary: Color = Color(0xCCFFFFFF) // --cst-content-inverse-secondary
            override val contentOnMarker: Color = Color(0xFFFFFFFF) // --cst-content-on-marker
            override val fieldBg: Color = Color(0xFFF6F7F3) // --cst-field-bg
            override val fieldBgDisabled: Color = Color(0xFFE4E6E2) // --cst-field-bg-disabled
            override val fieldBorder: Color = Color(0x3D2A4934) // --cst-field-border
            override val fieldBorderFocus: Color = Color(0xFF39714F) // --cst-field-border-focus
            override val fieldBorderHover: Color = Color(0x612A4934) // --cst-field-border-hover
            override val fieldRing: Color = Color(0x3839714F) // --cst-field-ring
            override val fieldRingDanger: Color = Color(0x3DA04B37) // --cst-field-ring-danger
            override val glowAccent: Color = Color(0x00000000) // --cst-glow-accent
            override val glowDanger: Color = Color(0x1F000000) // --cst-glow-danger
            override val interactiveActive: Color = Color(0x2B39714F) // --cst-interactive-active
            override val interactiveActiveFill: Color = Color(0x2B39714F) // --cst-interactive-active-fill
            override val interactiveHover: Color = Color(0x1F39714F) // --cst-interactive-hover
            override val interactiveHoverFill: Color = Color(0x1F39714F) // --cst-interactive-hover-fill
            override val markerGroup: Color = Color(0xFF4A703F) // --cst-marker-group
            override val markerGroupBg: Color = Color(0x1F4A703F) // --cst-marker-group-bg
            override val markerGroupBorder: Color = Color(0x384A703F) // --cst-marker-group-border
            override val markerGroupGlow: Color = Color(0x6B4A703F) // --cst-marker-group-glow
            override val markerMoment: Color = Color(0xFF8A6426) // --cst-marker-moment
            override val markerMomentBg: Color = Color(0x1F8A6426) // --cst-marker-moment-bg
            override val markerMomentBorder: Color = Color(0x388A6426) // --cst-marker-moment-border
            override val markerMomentGlow: Color = Color(0x6B8A6426) // --cst-marker-moment-glow
            override val markerUser: Color = Color(0xFF39714F) // --cst-marker-user
            override val markerUserBg: Color = Color(0x1F39714F) // --cst-marker-user-bg
            override val markerUserBorder: Color = Color(0x3839714F) // --cst-marker-user-border
            override val markerUserGlow: Color = Color(0x6B39714F) // --cst-marker-user-glow
            override val navigationActiveIndicator: Color = Color(0xFF39714F) // --cst-navigation-active-indicator
            override val navigationContent: Color = Color(0xFF354B3D) // --cst-navigation-content
            override val navigationContentActive: Color = Color(0xFF39714F) // --cst-navigation-content-active
            override val statusDanger: Color = Color(0xFFA04B37) // --cst-status-danger
            override val statusDangerBg: Color = Color(0x1FA04B37) // --cst-status-danger-bg
            override val statusDangerBorder: Color = Color(0x38A04B37) // --cst-status-danger-border
            override val statusOnline: Color = Color(0xFF39714F) // --cst-status-online
            override val statusOnlineBg: Color = Color(0x1F39714F) // --cst-status-online-bg
            override val statusOnlineBorder: Color = Color(0x3839714F) // --cst-status-online-border
            override val statusWarning: Color = Color(0xFF7E602B) // --cst-status-warning
            override val statusWarningBg: Color = Color(0x1F7E602B) // --cst-status-warning-bg
            override val statusWarningBorder: Color = Color(0x387E602B) // --cst-status-warning-border
            override val surfaceCardMenu: Color = Color(0xFFF6F7F3) // --cst-surface-card-menu
            override val surfaceChat: Color = Color(0xFFF2F3F2) // --cst-surface-chat
            override val surfaceGlassBadge: Color = Color(0x2EFFFFFF) // --cst-surface-glass-badge
            override val surfaceIsland: Color = Color(0xFFE4E6E2) // --cst-surface-island
            override val surfaceMediaScrimStrong: Color = Color(0x8C000000) // --cst-surface-media-scrim-strong
            override val surfaceNavGlass: Color = Color(0xE6F6F7F3) // --cst-surface-nav-glass
            override val surfaceNavigationActive: Color = Color(0xFFC5D4C8) // --cst-surface-navigation-active
            override val surfaceNavigationActiveHover: Color = Color(0xFFB6C9BB) // --cst-surface-navigation-active-hover
            override val surfaceNavigationHover: Color = Color(0xFFDBDDD9) // --cst-surface-navigation-hover
            override val surfaceOverlay: Color = Color(0x61141C14) // --cst-surface-overlay
            override val surfaceOverlayStrong: Color = Color(0x6B141C14) // --cst-surface-overlay-strong
            override val surfacePage: Color = Color(0xFFF6F7F3) // --cst-surface-page
            override val surfacePanel: Color = Color(0xFFF6F7F3) // --cst-surface-panel
            override val surfacePrimary: Color = Color(0xFFF6F7F3) // --cst-surface-primary
            override val surfaceRaised: Color = Color(0xFFF6F7F3) // --cst-surface-raised
            override val surfaceScrim: Color = Color(0x61141C14) // --cst-surface-scrim
            override val surfaceSubtle: Color = Color(0xFFE4E6E2) // --cst-surface-subtle
            override val surfaceToggleOff: Color = Color(0x38787880) // --cst-surface-toggle-off
            override val surfaceToggleThumb: Color = Color(0xFFF6F7F3) // --cst-surface-toggle-thumb
            override val textAccentOnSoft: Color = Color(0xFF326144) // --cst-text-accent-on-soft
            override val textOnAccent: Color = Color(0xFFFFFFFF) // --cst-text-on-accent
            override val textOnStatus: Color = Color(0xFFFFFFFF) // --cst-text-on-status
            override val textPrimary: Color = Color(0xFF152219) // --cst-text-primary
            override val textSecondary: Color = Color(0xFF354B3D) // --cst-text-secondary
            override val textStatusDangerOnSoft: Color = Color(0xFF844331) // --cst-text-status-danger-on-soft
            override val textStatusOnlineOnSoft: Color = Color(0xFF326144) // --cst-text-status-online-on-soft
            override val textStatusWarningOnSoft: Color = Color(0xFF695427) // --cst-text-status-warning-on-soft
        }
        /** Forest · dark */
        object ForestDark : SemanticPalette {
            override val accentBorder: Color = Color(0x5752B878) // --cst-accent-border
            override val accentBorderSoft: Color = Color(0x4D52B878) // --cst-accent-border-soft
            override val accentDefault: Color = Color(0xFF52B878) // --cst-accent-default
            override val accentSoft: Color = Color(0x1F52B878) // --cst-accent-soft
            override val accentSoftBg: Color = Color(0x1F52B878) // --cst-accent-soft-bg
            override val borderDefault: Color = Color(0x2EA9C7B2) // --cst-border-default
            override val borderStrong: Color = Color(0x4DA9C7B2) // --cst-border-strong
            override val borderSubtle: Color = Color(0x2EA9C7B2) // --cst-border-subtle
            override val contentInverse: Color = Color(0xFFFFFFFF) // --cst-content-inverse
            override val contentInverseMuted: Color = Color(0xB3FFFFFF) // --cst-content-inverse-muted
            override val contentInverseSecondary: Color = Color(0xCCFFFFFF) // --cst-content-inverse-secondary
            override val contentOnMarker: Color = Color(0xFF0C1911) // --cst-content-on-marker
            override val fieldBg: Color = Color(0xFF161A16) // --cst-field-bg
            override val fieldBgDisabled: Color = Color(0xFF272B27) // --cst-field-bg-disabled
            override val fieldBorder: Color = Color(0x2EA9C7B2) // --cst-field-border
            override val fieldBorderFocus: Color = Color(0xFF52B878) // --cst-field-border-focus
            override val fieldBorderHover: Color = Color(0x4DA9C7B2) // --cst-field-border-hover
            override val fieldRing: Color = Color(0x3852B878) // --cst-field-ring
            override val fieldRingDanger: Color = Color(0x3DD97C5A) // --cst-field-ring-danger
            override val glowAccent: Color = Color(0x00000000) // --cst-glow-accent
            override val glowDanger: Color = Color(0x1F000000) // --cst-glow-danger
            override val interactiveActive: Color = Color(0x3D52B878) // --cst-interactive-active
            override val interactiveActiveFill: Color = Color(0x3D52B878) // --cst-interactive-active-fill
            override val interactiveHover: Color = Color(0x2E52B878) // --cst-interactive-hover
            override val interactiveHoverFill: Color = Color(0x2E52B878) // --cst-interactive-hover-fill
            override val markerGroup: Color = Color(0xFF88B85A) // --cst-marker-group
            override val markerGroupBg: Color = Color(0x2488B85A) // --cst-marker-group-bg
            override val markerGroupBorder: Color = Color(0x4D88B85A) // --cst-marker-group-border
            override val markerGroupGlow: Color = Color(0x8088B85A) // --cst-marker-group-glow
            override val markerMoment: Color = Color(0xFFC4A75A) // --cst-marker-moment
            override val markerMomentBg: Color = Color(0x24C4A75A) // --cst-marker-moment-bg
            override val markerMomentBorder: Color = Color(0x4DC4A75A) // --cst-marker-moment-border
            override val markerMomentGlow: Color = Color(0x80C4A75A) // --cst-marker-moment-glow
            override val markerUser: Color = Color(0xFF52B878) // --cst-marker-user
            override val markerUserBg: Color = Color(0x2452B878) // --cst-marker-user-bg
            override val markerUserBorder: Color = Color(0x4D52B878) // --cst-marker-user-border
            override val markerUserGlow: Color = Color(0x8052B878) // --cst-marker-user-glow
            override val navigationActiveIndicator: Color = Color(0xFF52B878) // --cst-navigation-active-indicator
            override val navigationContent: Color = Color(0xFFB7CDBD) // --cst-navigation-content
            override val navigationContentActive: Color = Color(0xFF52B878) // --cst-navigation-content-active
            override val statusDanger: Color = Color(0xFFD97C5A) // --cst-status-danger
            override val statusDangerBg: Color = Color(0x24D97C5A) // --cst-status-danger-bg
            override val statusDangerBorder: Color = Color(0x47D97C5A) // --cst-status-danger-border
            override val statusOnline: Color = Color(0xFF52B878) // --cst-status-online
            override val statusOnlineBg: Color = Color(0x2452B878) // --cst-status-online-bg
            override val statusOnlineBorder: Color = Color(0x4752B878) // --cst-status-online-border
            override val statusWarning: Color = Color(0xFFC4A75A) // --cst-status-warning
            override val statusWarningBg: Color = Color(0x24C4A75A) // --cst-status-warning-bg
            override val statusWarningBorder: Color = Color(0x47C4A75A) // --cst-status-warning-border
            override val surfaceCardMenu: Color = Color(0xFF161A16) // --cst-surface-card-menu
            override val surfaceChat: Color = Color(0xFF0E100E) // --cst-surface-chat
            override val surfaceGlassBadge: Color = Color(0x2EFFFFFF) // --cst-surface-glass-badge
            override val surfaceIsland: Color = Color(0xFF232723) // --cst-surface-island
            override val surfaceMediaScrimStrong: Color = Color(0x8C000000) // --cst-surface-media-scrim-strong
            override val surfaceNavGlass: Color = Color(0xEB161A16) // --cst-surface-nav-glass
            override val surfaceNavigationActive: Color = Color(0xFF26432F) // --cst-surface-navigation-active
            override val surfaceNavigationActiveHover: Color = Color(0xFF2A5037) // --cst-surface-navigation-active-hover
            override val surfaceNavigationHover: Color = Color(0xFF2F342F) // --cst-surface-navigation-hover
            override val surfaceOverlay: Color = Color(0x6B000000) // --cst-surface-overlay
            override val surfaceOverlayStrong: Color = Color(0x8A000000) // --cst-surface-overlay-strong
            override val surfacePage: Color = Color(0xFF161A16) // --cst-surface-page
            override val surfacePanel: Color = Color(0xFF161A16) // --cst-surface-panel
            override val surfacePrimary: Color = Color(0xFF161A16) // --cst-surface-primary
            override val surfaceRaised: Color = Color(0xFF161A16) // --cst-surface-raised
            override val surfaceScrim: Color = Color(0x6B000000) // --cst-surface-scrim
            override val surfaceSubtle: Color = Color(0xFF272B27) // --cst-surface-subtle
            override val surfaceToggleOff: Color = Color(0x38787880) // --cst-surface-toggle-off
            override val surfaceToggleThumb: Color = Color(0xFF161A16) // --cst-surface-toggle-thumb
            override val textAccentOnSoft: Color = Color(0xFF70C38F) // --cst-text-accent-on-soft
            override val textOnAccent: Color = Color(0xFF0C1911) // --cst-text-on-accent
            override val textOnStatus: Color = Color(0xFF0C1911) // --cst-text-on-status
            override val textPrimary: Color = Color(0xFFE8F1E9) // --cst-text-primary
            override val textSecondary: Color = Color(0xFFB7CDBD) // --cst-text-secondary
            override val textStatusDangerOnSoft: Color = Color(0xFFDC9377) // --cst-text-status-danger-on-soft
            override val textStatusOnlineOnSoft: Color = Color(0xFF70C38F) // --cst-text-status-online-on-soft
            override val textStatusWarningOnSoft: Color = Color(0xFFCBB677) // --cst-text-status-warning-on-soft
        }
        /** Inkwash · light */
        object InkwashLight : SemanticPalette {
            override val accentBorder: Color = Color(0x47A63A2E) // --cst-accent-border
            override val accentBorderSoft: Color = Color(0x3DA63A2E) // --cst-accent-border-soft
            override val accentDefault: Color = Color(0xFFA63A2E) // --cst-accent-default
            override val accentSoft: Color = Color(0x1AA63A2E) // --cst-accent-soft
            override val accentSoftBg: Color = Color(0x1AA63A2E) // --cst-accent-soft-bg
            override val borderDefault: Color = Color(0x332B2A26) // --cst-border-default
            override val borderStrong: Color = Color(0x522B2A26) // --cst-border-strong
            override val borderSubtle: Color = Color(0x332B2A26) // --cst-border-subtle
            override val contentInverse: Color = Color(0xFFFFFFFF) // --cst-content-inverse
            override val contentInverseMuted: Color = Color(0xB3FFFFFF) // --cst-content-inverse-muted
            override val contentInverseSecondary: Color = Color(0xCCFFFFFF) // --cst-content-inverse-secondary
            override val contentOnMarker: Color = Color(0xFFFFFFFF) // --cst-content-on-marker
            override val fieldBg: Color = Color(0xFFFBF9F3) // --cst-field-bg
            override val fieldBgDisabled: Color = Color(0xFFEAE8E3) // --cst-field-bg-disabled
            override val fieldBorder: Color = Color(0x332B2A26) // --cst-field-border
            override val fieldBorderFocus: Color = Color(0xFFA63A2E) // --cst-field-border-focus
            override val fieldBorderHover: Color = Color(0x522B2A26) // --cst-field-border-hover
            override val fieldRing: Color = Color(0x38A63A2E) // --cst-field-ring
            override val fieldRingDanger: Color = Color(0x3D9E2140) // --cst-field-ring-danger
            override val glowAccent: Color = Color(0x00000000) // --cst-glow-accent
            override val glowDanger: Color = Color(0x1F000000) // --cst-glow-danger
            override val interactiveActive: Color = Color(0x2BA63A2E) // --cst-interactive-active
            override val interactiveActiveFill: Color = Color(0x2BA63A2E) // --cst-interactive-active-fill
            override val interactiveHover: Color = Color(0x1FA63A2E) // --cst-interactive-hover
            override val interactiveHoverFill: Color = Color(0x1FA63A2E) // --cst-interactive-hover-fill
            override val markerGroup: Color = Color(0xFF4A7048) // --cst-marker-group
            override val markerGroupBg: Color = Color(0x1F4A7048) // --cst-marker-group-bg
            override val markerGroupBorder: Color = Color(0x384A7048) // --cst-marker-group-border
            override val markerGroupGlow: Color = Color(0x6B4A7048) // --cst-marker-group-glow
            override val markerMoment: Color = Color(0xFF8A6426) // --cst-marker-moment
            override val markerMomentBg: Color = Color(0x1F8A6426) // --cst-marker-moment-bg
            override val markerMomentBorder: Color = Color(0x388A6426) // --cst-marker-moment-border
            override val markerMomentGlow: Color = Color(0x6B8A6426) // --cst-marker-moment-glow
            override val markerUser: Color = Color(0xFFA63A2E) // --cst-marker-user
            override val markerUserBg: Color = Color(0x1FA63A2E) // --cst-marker-user-bg
            override val markerUserBorder: Color = Color(0x38A63A2E) // --cst-marker-user-border
            override val markerUserGlow: Color = Color(0x6BA63A2E) // --cst-marker-user-glow
            override val navigationActiveIndicator: Color = Color(0xFFA63A2E) // --cst-navigation-active-indicator
            override val navigationContent: Color = Color(0xFF5D594F) // --cst-navigation-content
            override val navigationContentActive: Color = Color(0xFFA63A2E) // --cst-navigation-content-active
            override val statusDanger: Color = Color(0xFF9E2140) // --cst-status-danger
            override val statusDangerBg: Color = Color(0x1F9E2140) // --cst-status-danger-bg
            override val statusDangerBorder: Color = Color(0x389E2140) // --cst-status-danger-border
            override val statusOnline: Color = Color(0xFF3F7256) // --cst-status-online
            override val statusOnlineBg: Color = Color(0x1F3F7256) // --cst-status-online-bg
            override val statusOnlineBorder: Color = Color(0x383F7256) // --cst-status-online-border
            override val statusWarning: Color = Color(0xFF7F6422) // --cst-status-warning
            override val statusWarningBg: Color = Color(0x1F7F6422) // --cst-status-warning-bg
            override val statusWarningBorder: Color = Color(0x387F6422) // --cst-status-warning-border
            override val surfaceCardMenu: Color = Color(0xFFFBF9F3) // --cst-surface-card-menu
            override val surfaceChat: Color = Color(0xFFF1EDE3) // --cst-surface-chat
            override val surfaceGlassBadge: Color = Color(0x2EFFFFFF) // --cst-surface-glass-badge
            override val surfaceIsland: Color = Color(0xDBFBF9F3) // --cst-surface-island
            override val surfaceMediaScrimStrong: Color = Color(0x8C000000) // --cst-surface-media-scrim-strong
            override val surfaceNavGlass: Color = Color(0xE6FBF9F3) // --cst-surface-nav-glass
            override val surfaceNavigationActive: Color = Color(0xFFE0C2B9) // --cst-surface-navigation-active
            override val surfaceNavigationActiveHover: Color = Color(0xFFDAB3AA) // --cst-surface-navigation-active-hover
            override val surfaceNavigationHover: Color = Color(0xFFDDDAD2) // --cst-surface-navigation-hover
            override val surfaceOverlay: Color = Color(0x611A1814) // --cst-surface-overlay
            override val surfaceOverlayStrong: Color = Color(0x6B1A1814) // --cst-surface-overlay-strong
            override val surfacePage: Color = Color(0xFFF5F2EA) // --cst-surface-page
            override val surfacePanel: Color = Color(0xFFFBF9F3) // --cst-surface-panel
            override val surfacePrimary: Color = Color(0xFFFBF9F3) // --cst-surface-primary
            override val surfaceRaised: Color = Color(0xFFFBF9F3) // --cst-surface-raised
            override val surfaceScrim: Color = Color(0x611A1814) // --cst-surface-scrim
            override val surfaceSubtle: Color = Color(0xFFEAE8E3) // --cst-surface-subtle
            override val surfaceToggleOff: Color = Color(0x38787880) // --cst-surface-toggle-off
            override val surfaceToggleThumb: Color = Color(0xFFFBF9F3) // --cst-surface-toggle-thumb
            override val textAccentOnSoft: Color = Color(0xFF8D372C) // --cst-text-accent-on-soft
            override val textOnAccent: Color = Color(0xFFFFFFFF) // --cst-text-on-accent
            override val textOnStatus: Color = Color(0xFFFFFFFF) // --cst-text-on-status
            override val textPrimary: Color = Color(0xFF2B2A26) // --cst-text-primary
            override val textSecondary: Color = Color(0xFF5D594F) // --cst-text-secondary
            override val textStatusDangerOnSoft: Color = Color(0xFF87233B) // --cst-text-status-danger-on-soft
            override val textStatusOnlineOnSoft: Color = Color(0xFF3B644C) // --cst-text-status-online-on-soft
            override val textStatusWarningOnSoft: Color = Color(0xFF6E5823) // --cst-text-status-warning-on-soft
        }
        /** Inkwash · dark */
        object InkwashDark : SemanticPalette {
            override val accentBorder: Color = Color(0x4DDF7F6A) // --cst-accent-border
            override val accentBorderSoft: Color = Color(0x4DDF7F6A) // --cst-accent-border-soft
            override val accentDefault: Color = Color(0xFFDF7F6A) // --cst-accent-default
            override val accentSoft: Color = Color(0x1FDF7F6A) // --cst-accent-soft
            override val accentSoftBg: Color = Color(0x1FDF7F6A) // --cst-accent-soft-bg
            override val borderDefault: Color = Color(0x29EAE6DA) // --cst-border-default
            override val borderStrong: Color = Color(0x47EAE6DA) // --cst-border-strong
            override val borderSubtle: Color = Color(0x29EAE6DA) // --cst-border-subtle
            override val contentInverse: Color = Color(0xFFFFFFFF) // --cst-content-inverse
            override val contentInverseMuted: Color = Color(0xB3FFFFFF) // --cst-content-inverse-muted
            override val contentInverseSecondary: Color = Color(0xCCFFFFFF) // --cst-content-inverse-secondary
            override val contentOnMarker: Color = Color(0xFF1F1512) // --cst-content-on-marker
            override val fieldBg: Color = Color(0xFF212426) // --cst-field-bg
            override val fieldBgDisabled: Color = Color(0xFF313434) // --cst-field-bg-disabled
            override val fieldBorder: Color = Color(0x29EAE6DA) // --cst-field-border
            override val fieldBorderFocus: Color = Color(0xFFDF7F6A) // --cst-field-border-focus
            override val fieldBorderHover: Color = Color(0x47EAE6DA) // --cst-field-border-hover
            override val fieldRing: Color = Color(0x38DF7F6A) // --cst-field-ring
            override val fieldRingDanger: Color = Color(0x3DDE8496) // --cst-field-ring-danger
            override val glowAccent: Color = Color(0x00000000) // --cst-glow-accent
            override val glowDanger: Color = Color(0x1F000000) // --cst-glow-danger
            override val interactiveActive: Color = Color(0x3DDF7F6A) // --cst-interactive-active
            override val interactiveActiveFill: Color = Color(0x3DDF7F6A) // --cst-interactive-active-fill
            override val interactiveHover: Color = Color(0x2EDF7F6A) // --cst-interactive-hover
            override val interactiveHoverFill: Color = Color(0x2EDF7F6A) // --cst-interactive-hover-fill
            override val markerGroup: Color = Color(0xFF7FA88B) // --cst-marker-group
            override val markerGroupBg: Color = Color(0x247FA88B) // --cst-marker-group-bg
            override val markerGroupBorder: Color = Color(0x4D7FA88B) // --cst-marker-group-border
            override val markerGroupGlow: Color = Color(0x807FA88B) // --cst-marker-group-glow
            override val markerMoment: Color = Color(0xFFC89B5A) // --cst-marker-moment
            override val markerMomentBg: Color = Color(0x24C89B5A) // --cst-marker-moment-bg
            override val markerMomentBorder: Color = Color(0x4DC89B5A) // --cst-marker-moment-border
            override val markerMomentGlow: Color = Color(0x80C89B5A) // --cst-marker-moment-glow
            override val markerUser: Color = Color(0xFFDF7F6A) // --cst-marker-user
            override val markerUserBg: Color = Color(0x24DF7F6A) // --cst-marker-user-bg
            override val markerUserBorder: Color = Color(0x4DDF7F6A) // --cst-marker-user-border
            override val markerUserGlow: Color = Color(0x80DF7F6A) // --cst-marker-user-glow
            override val navigationActiveIndicator: Color = Color(0xFFDF7F6A) // --cst-navigation-active-indicator
            override val navigationContent: Color = Color(0xFFB7B3A8) // --cst-navigation-content
            override val navigationContentActive: Color = Color(0xFFDF7F6A) // --cst-navigation-content-active
            override val statusDanger: Color = Color(0xFFDE8496) // --cst-status-danger
            override val statusDangerBg: Color = Color(0x24DE8496) // --cst-status-danger-bg
            override val statusDangerBorder: Color = Color(0x47DE8496) // --cst-status-danger-border
            override val statusOnline: Color = Color(0xFF82B297) // --cst-status-online
            override val statusOnlineBg: Color = Color(0x2482B297) // --cst-status-online-bg
            override val statusOnlineBorder: Color = Color(0x4782B297) // --cst-status-online-border
            override val statusWarning: Color = Color(0xFFD6A54E) // --cst-status-warning
            override val statusWarningBg: Color = Color(0x24D6A54E) // --cst-status-warning-bg
            override val statusWarningBorder: Color = Color(0x47D6A54E) // --cst-status-warning-border
            override val surfaceCardMenu: Color = Color(0xFF212426) // --cst-surface-card-menu
            override val surfaceChat: Color = Color(0xFF1D2022) // --cst-surface-chat
            override val surfaceGlassBadge: Color = Color(0x2EFFFFFF) // --cst-surface-glass-badge
            override val surfaceIsland: Color = Color(0xFF2D3031) // --cst-surface-island
            override val surfaceMediaScrimStrong: Color = Color(0x8C000000) // --cst-surface-media-scrim-strong
            override val surfaceNavGlass: Color = Color(0xEB212426) // --cst-surface-nav-glass
            override val surfaceNavigationActive: Color = Color(0xFF4C3430) // --cst-surface-navigation-active
            override val surfaceNavigationActiveHover: Color = Color(0xFF5C3C37) // --cst-surface-navigation-active-hover
            override val surfaceNavigationHover: Color = Color(0xFF313233) // --cst-surface-navigation-hover
            override val surfaceOverlay: Color = Color(0x6B000000) // --cst-surface-overlay
            override val surfaceOverlayStrong: Color = Color(0x94000000) // --cst-surface-overlay-strong
            override val surfacePage: Color = Color(0xFF181A1C) // --cst-surface-page
            override val surfacePanel: Color = Color(0xFF212426) // --cst-surface-panel
            override val surfacePrimary: Color = Color(0xFF212426) // --cst-surface-primary
            override val surfaceRaised: Color = Color(0xFF212426) // --cst-surface-raised
            override val surfaceScrim: Color = Color(0x6B000000) // --cst-surface-scrim
            override val surfaceSubtle: Color = Color(0xFF313434) // --cst-surface-subtle
            override val surfaceToggleOff: Color = Color(0x38787880) // --cst-surface-toggle-off
            override val surfaceToggleThumb: Color = Color(0xFF212426) // --cst-surface-toggle-thumb
            override val textAccentOnSoft: Color = Color(0xFFE19480) // --cst-text-accent-on-soft
            override val textOnAccent: Color = Color(0xFF1F1512) // --cst-text-on-accent
            override val textOnStatus: Color = Color(0xFF1F1512) // --cst-text-on-status
            override val textPrimary: Color = Color(0xFFEAE6DA) // --cst-text-primary
            override val textSecondary: Color = Color(0xFFB7B3A8) // --cst-text-secondary
            override val textStatusDangerOnSoft: Color = Color(0xFFE098A4) // --cst-text-status-danger-on-soft
            override val textStatusOnlineOnSoft: Color = Color(0xFF97BCA4) // --cst-text-status-online-on-soft
            override val textStatusWarningOnSoft: Color = Color(0xFFDAB26A) // --cst-text-status-warning-on-soft
        }
    }

    object Ambient {
        /** Vibrant · light */
        object VibrantLight : AmbientPalette {
            override val dot: Color = Color(0x2E2B5EB2) // --cst-ambient-dot
            override val emblem: Color = Color(0x142B5EB2) // --cst-ambient-emblem
            override val glow1: Color = Color(0x0F2B5EB2) // --cst-ambient-glow-1
            override val glow2: Color = Color(0x0D0D6A62) // --cst-ambient-glow-2
            override val horizon: Color = Color(0x082B5EB2) // --cst-ambient-horizon
            override val ink1: Color = Color(0x092B5EB2) // --cst-ambient-ink-1
            override val ink2: Color = Color(0x052B5EB2) // --cst-ambient-ink-2
            override val ink3: Color = Color(0x042B5EB2) // --cst-ambient-ink-3
            override val line: Color = Color(0x122B5EB2) // --cst-ambient-line
            override val sky: Color = Color(0x082B5EB2) // --cst-ambient-sky
            override val sun: Color = Color(0x00000000) // --cst-ambient-sun
            override val vignette: Color = Color(0x0816233A) // --cst-ambient-vignette
        }
        /** Vibrant · dark */
        object VibrantDark : AmbientPalette {
            override val dot: Color = Color(0x385B9CF6) // --cst-ambient-dot
            override val emblem: Color = Color(0x1A5B9CF6) // --cst-ambient-emblem
            override val glow1: Color = Color(0x145B9CF6) // --cst-ambient-glow-1
            override val glow2: Color = Color(0x0F34D399) // --cst-ambient-glow-2
            override val horizon: Color = Color(0x0A5B9CF6) // --cst-ambient-horizon
            override val ink1: Color = Color(0x0B5B9CF6) // --cst-ambient-ink-1
            override val ink2: Color = Color(0x085B9CF6) // --cst-ambient-ink-2
            override val ink3: Color = Color(0x055B9CF6) // --cst-ambient-ink-3
            override val line: Color = Color(0x175B9CF6) // --cst-ambient-line
            override val sky: Color = Color(0x0A5B9CF6) // --cst-ambient-sky
            override val sun: Color = Color(0x00000000) // --cst-ambient-sun
            override val vignette: Color = Color(0x33000000) // --cst-ambient-vignette
        }
        /** Forest · light */
        object ForestLight : AmbientPalette {
            override val dot: Color = Color(0x2439714F) // --cst-ambient-dot
            override val emblem: Color = Color(0x1439714F) // --cst-ambient-emblem
            override val glow1: Color = Color(0x0D39714F) // --cst-ambient-glow-1
            override val glow2: Color = Color(0x0D7E602B) // --cst-ambient-glow-2
            override val horizon: Color = Color(0x0A7E602B) // --cst-ambient-horizon
            override val ink1: Color = Color(0x0939714F) // --cst-ambient-ink-1
            override val ink2: Color = Color(0x0639714F) // --cst-ambient-ink-2
            override val ink3: Color = Color(0x0439714F) // --cst-ambient-ink-3
            override val line: Color = Color(0x1239714F) // --cst-ambient-line
            override val sky: Color = Color(0x087E602B) // --cst-ambient-sky
            override val sun: Color = Color(0x0D7E602B) // --cst-ambient-sun
            override val vignette: Color = Color(0x08152219) // --cst-ambient-vignette
        }
        /** Forest · dark */
        object ForestDark : AmbientPalette {
            override val dot: Color = Color(0x2E52B878) // --cst-ambient-dot
            override val emblem: Color = Color(0x1A52B878) // --cst-ambient-emblem
            override val glow1: Color = Color(0x1252B878) // --cst-ambient-glow-1
            override val glow2: Color = Color(0x12C4A75A) // --cst-ambient-glow-2
            override val horizon: Color = Color(0x0DC4A75A) // --cst-ambient-horizon
            override val ink1: Color = Color(0x0B52B878) // --cst-ambient-ink-1
            override val ink2: Color = Color(0x0852B878) // --cst-ambient-ink-2
            override val ink3: Color = Color(0x0552B878) // --cst-ambient-ink-3
            override val line: Color = Color(0x1752B878) // --cst-ambient-line
            override val sky: Color = Color(0x0AC4A75A) // --cst-ambient-sky
            override val sun: Color = Color(0x0FC4A75A) // --cst-ambient-sun
            override val vignette: Color = Color(0x33000000) // --cst-ambient-vignette
        }
        /** Inkwash · light */
        object InkwashLight : AmbientPalette {
            override val dot: Color = Color(0x142B2A26) // --cst-ambient-dot
            override val emblem: Color = Color(0x14A63A2E) // --cst-ambient-emblem
            override val glow1: Color = Color(0x0DA63A2E) // --cst-ambient-glow-1
            override val glow2: Color = Color(0x082B2A26) // --cst-ambient-glow-2
            override val horizon: Color = Color(0x052B2A26) // --cst-ambient-horizon
            override val ink1: Color = Color(0x092B2A26) // --cst-ambient-ink-1
            override val ink2: Color = Color(0x052B2A26) // --cst-ambient-ink-2
            override val ink3: Color = Color(0x032B2A26) // --cst-ambient-ink-3
            override val line: Color = Color(0x0F2B2A26) // --cst-ambient-line
            override val sky: Color = Color(0x062B2A26) // --cst-ambient-sky
            override val sun: Color = Color(0x00000000) // --cst-ambient-sun
            override val vignette: Color = Color(0x082B2A26) // --cst-ambient-vignette
        }
        /** Inkwash · dark */
        object InkwashDark : AmbientPalette {
            override val dot: Color = Color(0x1FEAE6DA) // --cst-ambient-dot
            override val emblem: Color = Color(0x1ADF7F6A) // --cst-ambient-emblem
            override val glow1: Color = Color(0x12DF7F6A) // --cst-ambient-glow-1
            override val glow2: Color = Color(0x0AEAE6DA) // --cst-ambient-glow-2
            override val horizon: Color = Color(0x08EAE6DA) // --cst-ambient-horizon
            override val ink1: Color = Color(0x09EAE6DA) // --cst-ambient-ink-1
            override val ink2: Color = Color(0x05EAE6DA) // --cst-ambient-ink-2
            override val ink3: Color = Color(0x03EAE6DA) // --cst-ambient-ink-3
            override val line: Color = Color(0x14EAE6DA) // --cst-ambient-line
            override val sky: Color = Color(0x08EAE6DA) // --cst-ambient-sky
            override val sun: Color = Color(0x00000000) // --cst-ambient-sun
            override val vignette: Color = Color(0x33000000) // --cst-ambient-vignette
        }
    }

    /** 字号档（sp）：10/11/12/14/18/20/24/30/36，对齐 design-system §4.1 */
    object FontSize {
        const val hero: Float = 36f // --text-4xl
        const val lg: Float = 18f // --text-lg
        const val md: Float = 14f // --text-md
        const val sm: Float = 12f // --text-sm
        const val xl: Float = 20f // --text-xl
        const val xs: Float = 11f // --text-xs
        const val xxl: Float = 24f // --text-2xl
        const val xxs: Float = 10f // --text-xxs
        const val xxxl: Float = 30f // --text-3xl
    }

    /** 圆角档：12/16/20/24/9999(胶囊)/28(sheet) */
    object Radius {
        const val full: Float = 9999f // --cst-shape-full
        const val lg: Float = 20f // --cst-shape-lg
        const val md: Float = 16f // --cst-shape-md
        const val sheet: Float = 28f // --radius-sheet
        const val sm: Float = 12f // --cst-shape-sm
        const val xl: Float = 24f // --cst-shape-xl
    }

    /** 动效常量（ms）与曲线：对齐 design-system §10.1 */
    object Motion {
        const val ambient: Int = 24000 // --cst-motion-ambient
        const val ambientSlow: Int = 36000 // --cst-motion-ambient-slow
        const val contentFade: Int = 110 // --cst-content-fade-duration
        const val gentle: Int = 500 // --cst-motion-gentle
        const val instant: Int = 100 // --cst-motion-instant
        const val pageEnter: Int = 120 // --cst-page-enter-duration
        const val pulse: Int = 2400 // --cst-motion-pulse
        const val rapid: Int = 150 // --cst-motion-rapid
        const val reduced: Int = 1400 // --cst-motion-reduced
        const val relaxed: Int = 260 // --cst-motion-relaxed
        const val skeleton: Int = 1500 // --cst-motion-skeleton
        const val slow: Int = 400 // --cst-motion-slow
        const val snappy: Int = 180 // --cst-motion-snappy
        const val spinner: Int = 700 // --cst-motion-spinner
        const val spring: Int = 300 // --cst-motion-spring
        const val standard: Int = 220 // --cst-motion-standard
        val easeStandard: CubicBezierEasing = CubicBezierEasing(0.22f, 1f, 0.36f, 1f) // --cst-ease-standard
        val easeBounce: CubicBezierEasing = CubicBezierEasing(0.34f, 1.56f, 0.64f, 1f) // --cst-ease-bounce
    }
}
