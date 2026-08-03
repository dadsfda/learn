<script setup lang="ts">
const name = '枫枫'
const className = 'active'
const classObj = {
  menu: true,
  active: true,
}

const styleObj = {
  fontSize: '20px',
  color: 'blue',
}
</script>

<template>
  <div>
    动态绑定
    <div v-bind:name="name">动态绑定name</div>
    <div :name="name">动态绑定name：{{ name }}</div>
    <div v-bind:class="className">动态绑定class</div>
    <div :class="className">动态绑定class</div>
    <div :class="classObj">动态绑定class obj</div>
    <div :style="styleObj">动态绑定style</div>
  </div>
</template>

<style scoped>
.active {
  color: red;
}
</style>
