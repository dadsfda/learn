<script setup lang="ts">
import { ref } from 'vue'
import CounterButton from './CounterButton.vue'

const title = ref('计数器')
const count = ref(0)

function handleIncrease(step: number) {
  count.value += step
}
</script>

<template>
  <div>
    <h3>Props 和 Emits</h3>

    <label>
      父组件中的标题：
      <input v-model="title" />
    </label>

    <CounterButton
      :title="title"
      :count="count"
      @increase="handleIncrease"
    />

    <p>父组件中的数量：{{ count }}</p>
  </div>
</template>
