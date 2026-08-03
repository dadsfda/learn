<script setup lang="ts">
const array = ['tom', 'jerry', '张三', '李四']

const obj = {
  name: '张三',
  age: 21,
  addr: '湖南长沙',
}
</script>

<template>
  <div>
    列表渲染

    item in array
    <ul>
      <li v-for="item in array">{{ item }}</li>
    </ul>

    (item, index) in array
    <ul>
      <li v-for="(item, index) in array">{{ index }}.{{ item }}</li>
    </ul>

    item in obj
    <ul>
      <li v-for="item in obj">{{ item }}</li>
    </ul>

    (value, key) in obj
    <ul>
      <li v-for="(value, key) in obj">{{ key }}: {{ value }}</li>
    </ul>

    (value, key, index) in obj
    <ul>
      <li v-for="(value, key, index) in obj">{{ index }}.{{ key }}: {{ value }}</li>
    </ul>
  </div>
</template>
