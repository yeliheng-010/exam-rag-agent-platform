<template>
  <component
    :is="as"
    ref="rootElement"
    class="exam-rich-text"
    :class="{ 'exam-rich-text--inline': inline }"
    v-html="renderedHTML"
  />
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import 'katex/dist/katex.min.css'

import { createChatMarkdownRenderer, renderChatMarkdown } from '@/utils/chatMarkdownRenderer'
import {
  createSafeImage,
  hydrateProtectedFileImages,
  isValidImageURL,
  safeMarkdownToHTML,
  sanitizeMarkdownHTML,
} from '@/utils/security'

const props = withDefaults(defineProps<{
  content?: string | null
  inline?: boolean
  as?: 'div' | 'span'
}>(), {
  content: '',
  inline: false,
  as: 'div',
})

const rootElement = ref<HTMLElement | null>(null)
const renderer = createChatMarkdownRenderer({
  imageRenderer: ({ href, title, text }) => createSafeImage(href, text, title || ''),
  isValidImageUrl: isValidImageURL,
})

const renderedHTML = computed(() => renderChatMarkdown(props.content, {
  renderer,
  escapeMarkdown: safeMarkdownToHTML,
  sanitizeHtml: sanitizeMarkdownHTML,
  streaming: false,
}))

const hydrateImages = async () => {
  await nextTick()
  await hydrateProtectedFileImages(rootElement.value)
}

watch(renderedHTML, hydrateImages)
onMounted(hydrateImages)
</script>

<style lang="less" scoped>
.exam-rich-text {
  min-width: 0;
  overflow-wrap: anywhere;

  :deep(p) {
    margin: 0 0 8px;
  }

  :deep(p:last-child) {
    margin-bottom: 0;
  }

  :deep(.katex-display) {
    max-width: 100%;
    margin: 8px 0;
    overflow-x: auto;
    overflow-y: hidden;
  }

  :deep(img) {
    display: block;
    max-width: min(100%, 760px);
    height: auto;
    margin: 8px 0;
    object-fit: contain;
  }

  :deep(img[data-img-loading='1']) {
    width: 88px;
    height: 44px;
    border-radius: 4px;
    background: var(--td-bg-color-secondarycontainer);
  }
}

.exam-rich-text--inline {
  display: block;
  max-width: 100%;
  overflow-x: auto;
  overflow-y: hidden;

  :deep(p) {
    display: inline;
    margin: 0;
  }

  :deep(img) {
    display: inline-block;
    max-width: min(100%, 420px);
    max-height: 96px;
    margin: 2px 4px;
    vertical-align: middle;
  }
}
</style>
