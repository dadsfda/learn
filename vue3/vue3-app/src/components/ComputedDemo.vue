<script setup lang="ts">
import { computed, ref } from 'vue'

const firstNumber = ref(10)
const secondNumber = ref(20)

const sum = computed({
  get() {
    return firstNumber.value + secondNumber.value
  },

  set(newValue) {
    firstNumber.value = newValue - secondNumber.value
  },
})
</script>

<template>
  <div>
    可写计算属性

    <div>
      <label>
        第一个数：
        <input v-model.number="firstNumber" type="number" />
      </label>
    </div>

    <div>
      <label>
        第二个数：
        <input v-model.number="secondNumber" type="number" />
      </label>
    </div>

    <p>{{ firstNumber }} + {{ secondNumber }} = {{ sum }}</p>

    <div>
      <label>
        修改总和：
        <input v-model.number="sum" type="number" />
      </label>
    </div>
  </div>
</template>
