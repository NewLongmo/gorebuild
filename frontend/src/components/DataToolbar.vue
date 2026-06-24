<template>
  <div class="data-toolbar">
    <a-input-search
      v-model:value="keyword"
      :placeholder="placeholder"
      allow-clear
      :enter-button="searchText"
      class="data-toolbar__search"
      @search="emitSearch"
    />
    <slot />
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';

const props = withDefaults(defineProps<{
  modelValue: string;
  placeholder: string;
  searchText?: string;
}>(), {
  searchText: '搜索',
});

const emit = defineEmits<{
  'update:modelValue': [value: string];
  search: [];
}>();

const keyword = ref(props.modelValue);

watch(
  () => props.modelValue,
  (value) => {
    keyword.value = value;
  },
);

watch(keyword, (value) => emit('update:modelValue', value));

function emitSearch() {
  emit('search');
}
</script>
