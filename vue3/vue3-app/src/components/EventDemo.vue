<script setup lang="ts">
import { ref } from "vue";

const msg = ref("")
function clickHandler(e: Event) {
  console.log('点击事件', e)
}

function clickHandler1(name: string, e: Event) {
  console.log('点击事件1', name, e)
}

function alertHandler(msg: string) {
  alert(msg)
}

function enterHandler(e: KeyboardEvent) {
  const target = e.target as HTMLInputElement
  const value = target.value
  console.log('enter键按下了', value)
}

function inputHandler(e: InputEvent) {
  console.log('键盘输入中', e.data)
}

function focusHandler() {
  console.log('触发焦点了')
}

function blurHandler() {
  console.log('失去焦点了')
}
function clickHandler2(e: PointerEvent) {
  console.log(e.target)
}
function clickHandler3(e: Event, name: string) {
  console.log(e.target, name)
}
</script>

<template>
  <div>
    事件绑定
    <div>点击事件 <button v-on:click="clickHandler">点我</button></div>
    <div>点击事件 <button @click="clickHandler">点我</button></div>
    <div>点击事件 <button @click="clickHandler1('枫枫', $event)">点我</button></div>

    <div>双击事件 <button @dblclick="alertHandler('双击')">双击</button></div>
    <div>右键 <button @click.right="alertHandler('右键')">右键</button></div>
    <div>中键 <button @click.middle="alertHandler('中键')">中键</button></div>

    <div>enter键 <input @keydown.enter="enterHandler" placeholder="enter键" /></div>
    <div>input输入 <input @input="inputHandler" placeholder="input输入" /></div>
    <div>input触发焦点 <input @focus="focusHandler" placeholder="input触发焦点" /></div>
    <div>input失去焦点 <input @blur="blurHandler" placeholder="input失去焦点" /></div>
  </div>
  <div>
    事件委托
    <a href="https://www.fengfengzhidao.com" @click.prevent="clickHandler2">枫枫知道</a>
  </div>
  <div>
    事件冒泡
    <div class="A" @click.self="clickHandler3($event, 'A')">
      A
      <div class="B" @click="clickHandler3($event, 'B')">
        B
        <button @click.stop="clickHandler3($event, 'Button')">button</button>
      </div>
    </div>

  </div>
  <div>
    双向数据绑定

    <div>
      <input v-model="msg" placeholder="输入内容">
    </div>
    <div>
      你输入的内容 {{ msg }}
    </div>
  </div>
</template>
<style scoped>
.A {
  background-color: #ee4a4a;
  padding: 20px;
}

.B {
  background-color: #288163;
  padding: 20px;
}
</style>