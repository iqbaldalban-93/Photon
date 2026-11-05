<script setup lang="ts">
defineProps<{ size?: number }>()
</script>

<template>
  <svg
    :width="size ?? 28"
    :height="size ?? 28"
    viewBox="0 0 32 32"
    fill="none"
    aria-hidden="true"
  >
    <circle
      cx="16"
      cy="16"
      r="15.5"
      fill="var(--primary)"
    />
    <circle
      cx="16"
      cy="16"
      r="11.6"
      stroke="var(--primary-foreground)"
      stroke-width="1.5"
    />
    <g
      stroke="var(--primary-foreground)"
      stroke-width="1.5"
      stroke-linecap="round"
    >
      <path d="M20.50 13.40 L27.51 17.45" />
      <path d="M20.50 18.60 L20.50 26.69" />
      <path d="M16.00 21.20 L8.99 25.25" />
      <path d="M11.50 18.60 L4.49 14.55" />
      <path d="M11.50 13.40 L11.50 5.31" />
      <path d="M16.00 10.80 L23.01 6.75" />
    </g>
    <polygon
      points="16.00,10.80 20.50,13.40 20.50,18.60 16.00,21.20 11.50,18.60 11.50,13.40"
      fill="var(--primary-foreground)"
      stroke="var(--primary-foreground)"
      stroke-width="1.5"
      stroke-linejoin="round"
    />
  </svg>
</template>
