<script setup lang="ts">
const props = defineProps<{
  title: string
  count: number
}>()

const emit = defineEmits<{
  increase: [step: number]
}>()

function handleClick() {
  emit('increase', 1)
}
</script>

<template>
  <div>
    <p>{{ props.title }}</p>
    <p>子组件中的数量：{{ props.count }}</p>
    <button @click="handleClick">增加</button>
  </div>
</template>
