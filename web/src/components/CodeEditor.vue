<template lang="pug">
  .code-editor(ref="hostEl" :class="{ 'is-readonly': readonly, 'is-fill': fill }")
</template>

<script setup>
  import { basicSetup } from 'codemirror';
  import { EditorView } from '@codemirror/view';
  import { EditorState } from '@codemirror/state';
  import { json } from '@codemirror/lang-json';

  const props = defineProps({
    modelValue: { type: String, default: '' },
    readonly: { type: Boolean, default: false },
    // fill：编辑器铺满父容器高度、由 CodeMirror 内部滚动；此模式下忽略 minHeight/maxHeight
    fill: { type: Boolean, default: false },
    minHeight: { type: String, default: '160px' },
    maxHeight: { type: String, default: '420px' },
  });

  const emit = defineEmits(['update:modelValue', 'focus', 'blur']);

  const hostEl = ref(null);
  let view = null;

  // 内容回写外部 v-model：仅在文档实际变化时触发，避免与外部写回形成回环
  const contentListener = EditorView.updateListener.of((u) => {
    if (u.docChanged) emit('update:modelValue', u.state.doc.toString());
  });

  function buildExtensions() {
    const exts = [
      basicSetup,
      json(),
      EditorView.lineWrapping,
      EditorView.contentAttributes.of({ 'aria-label': 'code editor' }),
      EditorView.domEventHandlers({
        focus: () => emit('focus'),
        blur: () => emit('blur'),
      }),
      contentListener,
      EditorView.theme(
        {
          '&': {
            fontSize: '12.5px',
            border: '1px solid var(--el-border-color)',
            borderRadius: '4px',
            height: props.fill ? '100%' : 'auto',
          },
          '.cm-scroller': {
            overflow: 'auto',
            fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace',
            ...(props.fill
              ? { height: '100%' }
              : { minHeight: props.minHeight, maxHeight: props.maxHeight }),
          },
          '.cm-gutters': { backgroundColor: 'var(--el-fill-color-light)', borderRight: '1px solid var(--el-border-color-lighter)' },
        },
        { base: false },
      ),
    ];
    if (props.readonly) {
      exts.push(EditorState.readOnly.of(true), EditorView.editable.of(false));
    }
    return exts;
  }

  function createView() {
    destroyView();
    view = new EditorView({
      state: EditorState.create({ doc: props.modelValue ?? '', extensions: buildExtensions() }),
      parent: hostEl.value,
    });
  }

  function destroyView() {
    if (view) {
      view.destroy();
      view = null;
    }
  }

  // 外部（表单 → JSON）写回：只在内容真正不同时整篇替换，且绝不主动夺焦点
  watch(
    () => props.modelValue,
    (val) => {
      if (!view) return;
      const text = val ?? '';
      if (view.state.doc.toString() === text) return;
      view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: text } });
    },
  );

  // 只读态切换时重建以正确应用 editable 扩展
  watch(
    () => props.readonly,
    () => createView(),
  );

  onMounted(createView);
  onBeforeUnmount(destroyView);
</script>

<style scoped>
  .code-editor :deep(.cm-editor) {
    background-color: var(--el-bg-color);
  }
  .code-editor :deep(.cm-editor.cm-focused) {
    outline: none;
  }
  .code-editor.is-readonly :deep(.cm-editor) {
    background-color: var(--el-fill-color-lighter);
  }
  .code-editor.is-fill {
    height: 100%;
  }
  .code-editor.is-fill :deep(.cm-editor) {
    height: 100%;
  }
</style>
