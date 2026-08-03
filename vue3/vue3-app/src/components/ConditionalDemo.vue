<script setup lang="ts">
import { ref } from 'vue'

const visible = ref(true)
const gender = false
const age = 12
</script>

<template>
  <button @click="visible = !visible">
    切换显示：{{ visible }}
  </button>
   <div>
    v-if：删除元素
    <span v-if="visible">我是 v-if</span>
  </div>

  <div>
    v-show：隐藏元素
    <span v-show="visible">我是 v-show</span>
  </div>
  <div>
    条件渲染
    <div>
      v-if v-else
      <span v-if="gender">男</span>
      <span v-else>女</span>

      <br />
      v-if v-else-if v-else
      <div>
        <span v-if="age >= 35">被优化</span>
        <span v-else-if="age >= 18">已成年</span>
        <span v-else>未成年</span>
      </div>

    </div>

    <div>
      v-show
      <div>
        <span v-show="gender">男</span>
      </div>
    </div>
  </div>
</template>
