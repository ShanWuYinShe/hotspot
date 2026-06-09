<script lang="ts">
  let { children } = $props();
</script>

<aside class="flex flex-col gap-3 w-full">
  {@render children()}
</aside>
