<script lang="ts">
	import MapCanvas from '../MapCanvas.svelte';

	// 测试宿主：把 MapCanvas 的 bindable 值渲染进 DOM，供 jsdom 断言。
	let { center = $bindable(null), zoom = $bindable(14), gestureTransform = $bindable('') } = $props();
</script>

<div data-testid="zoom">{zoom}</div>
<div data-testid="gt">{gestureTransform === '' ? '(idle)' : gestureTransform}</div>
<div data-testid="center">{center ? center.lat.toFixed(6) + ',' + center.lng.toFixed(6) : 'null'}</div>
<div data-testid="canvas-host">
	<MapCanvas bind:center bind:zoom bind:gestureTransform onMapTap={() => {}} />
</div>
