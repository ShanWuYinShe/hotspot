// ═══ CSTUI (ChaoSet UI) 独立组件库入口 ═══
// 全站通用的纯 UI 原语与设计系统标准组件，零业务绑定，可直接跨项目复用。

// Cards
export { default as Card } from './cards/Card.svelte';
export { default as MediaCard } from './cards/MediaCard.svelte';
export { default as ImageCardActions } from './cards/ImageCardActions.svelte';

// Buttons
export { default as Button } from './buttons/Button.svelte';
export { default as ActionIconButton } from './buttons/ActionIconButton.svelte';
export { default as FollowButton } from './buttons/FollowButton.svelte';
export { default as JoinButton } from './buttons/JoinButton.svelte';

// Data Display
export { default as Badge } from './data-display/Badge.svelte';
export { default as Avatar } from './data-display/Avatar.svelte';
export { default as OnlineDot } from './data-display/OnlineDot.svelte';
export { default as UnreadBadge } from './data-display/UnreadBadge.svelte';
export { default as MetricRow } from './data-display/MetricRow.svelte';

// Forms
export { default as Input } from './forms/Input.svelte';
export { default as Textarea } from './forms/Textarea.svelte';
export { default as SearchInput } from './forms/SearchInput.svelte';
export { default as Checkbox } from './forms/Checkbox.svelte';
export { default as Toggle } from './forms/Toggle.svelte';
export { default as Select } from './forms/Select.svelte';
export { default as ImagePicker } from './forms/ImagePicker.svelte';
export { default as TagPill } from './forms/TagPill.svelte';

// List
export { default as ListContainer } from './list/ListContainer.svelte';
export { default as ListItem } from './list/ListItem.svelte';
export { default as LoadMoreButton } from './list/LoadMoreButton.svelte';
export { default as EntityRow } from './list/EntityRow.svelte';

// Feedback
export { default as EmptyState } from './feedback/EmptyState.svelte';
export { default as InlineLoading } from './feedback/InlineLoading.svelte';
export { default as LoadError } from './feedback/LoadError.svelte';
export { default as ListEnd } from './feedback/ListEnd.svelte';

// Overlays
export { default as ConfirmDialog } from './overlays/ConfirmDialog.svelte';
export { default as SheetShell } from './overlays/SheetShell.svelte';
export { default as ReportDialog } from './overlays/ReportDialog.svelte';
export * as Dialog from './overlays/dialog';

// Navigation
export { default as TabBar } from './navigation/TabBar.svelte';

// Layout
export { default as PageHeader } from './layout/PageHeader.svelte';
export { default as DetailHeader } from './layout/DetailHeader.svelte';
export { default as StandardPageLayout } from './layout/StandardPageLayout.svelte';
export { default as SplitPageLayout } from './layout/SplitPageLayout.svelte';
