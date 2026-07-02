import { reactive, ref, computed, watch } from 'vue'
import { defineStore } from 'pinia'
import i18n from '@/i18n'
import { useAuthStore } from '@/stores/auth'

type MenuChild = Record<string, any>

interface MenuItem {
  title: string
  titleKey?: string
  icon: string
  path: string
  childrenPath?: string
  minRole?: 'viewer' | 'contributor' | 'admin' | 'owner'
  children?: MenuChild[]
}

const createMenuChildren = () => reactive<MenuChild[]>([])

export const useMenuStore = defineStore('menuStore', () => {
  const menuArr = reactive<MenuItem[]>([
    {
      title: '',
      titleKey: 'menu.newChat',
      icon: 'prefixIcon',
      path: 'creatChat',
      childrenPath: 'chat',
      children: createMenuChildren()
    },
    { title: '', titleKey: 'menu.learning', icon: 'zhishiku', path: 'learning' },
    { title: '', titleKey: 'menu.classes', icon: 'organization', path: 'classes' },
    { title: '', titleKey: 'menu.questionBank', icon: 'zhishiku', path: 'question-banks', minRole: 'contributor' },
    { title: '', titleKey: 'menu.knowledgeBase', icon: 'zhishiku', path: 'knowledge-bases', minRole: 'contributor' },
    { title: '', titleKey: 'menu.agents', icon: 'agent', path: 'agents', minRole: 'contributor' },
    { title: '', titleKey: 'menu.analytics', icon: 'integration', path: 'analytics', minRole: 'admin' },
    { title: '', titleKey: 'menu.billing', icon: 'setting', path: 'billing', minRole: 'admin' },
    { title: '', titleKey: 'menu.review', icon: 'setting', path: 'review', minRole: 'contributor' },
    { title: '', titleKey: 'menu.examConfig', icon: 'setting', path: 'exam-config', minRole: 'admin' },
    { title: '', titleKey: 'menu.integrations', icon: 'integration', path: 'integrations', minRole: 'admin' },
    { title: '', titleKey: 'menu.organizations', icon: 'organization', path: 'organizations', minRole: 'admin' },
    { title: '', titleKey: 'menu.settings', icon: 'setting', path: 'settings' },
    { title: '', titleKey: 'menu.logout', icon: 'logout', path: 'logout' }
  ])

  const isFirstSession = ref(false)
  const firstQuery = ref('')
  const firstMentionedItems = ref<any[]>([])
  const firstModelId = ref('')
  const firstImageFiles = ref<any[]>([])
  const firstAttachmentFiles = ref<any[]>([])
  const prefillQuery = ref('')

  const applyMenuTranslations = () => {
    menuArr.forEach(item => {
      if (item.titleKey) {
        item.title = i18n.global.t(item.titleKey)
      }
    })
  }

  applyMenuTranslations()

  watch(
    () => i18n.global.locale.value,
    () => {
      applyMenuTranslations()
    }
  )

  const liteHiddenPaths = new Set(['logout', 'organizations'])

  // 侧栏只展示当前角色自然能使用的模块；后端 RBAC 才是权限最终来源。
  const visibleMenuArr = computed(() => {
    const authStore = useAuthStore()
    return menuArr.filter(item => {
      if (authStore.isLiteMode && liteHiddenPaths.has(item.path)) {
        return false
      }
      if (item.minRole && !authStore.hasRole(item.minRole)) {
        return false
      }
      return true
    })
  })

  const chatMenuIndex = menuArr.findIndex(item => item.path === 'creatChat')

  const clearMenuArr = () => {
    const chatMenu = menuArr[chatMenuIndex]
    if (chatMenu && chatMenu.children) {
      chatMenu.children = createMenuChildren()
    }
  }

  const updatemenuArr = (obj: any) => {
    const chatMenu = menuArr[chatMenuIndex]
    if (!chatMenu.children) {
      chatMenu.children = createMenuChildren()
    }
    const exists = chatMenu.children.some((item: MenuChild) => item.id === obj.id)
    if (!exists) {
      chatMenu.children.push(obj)
    }
  }

  const updataMenuChildren = (item: MenuChild) => {
    const chatMenu = menuArr[chatMenuIndex]
    if (!chatMenu.children) {
      chatMenu.children = createMenuChildren()
    }
    chatMenu.children.unshift(item)
  }

  const updatasessionTitle = (sessionId: string, title: string) => {
    const chatMenu = menuArr[chatMenuIndex]
    chatMenu.children?.forEach((item: MenuChild) => {
      if (item.id === sessionId) {
        item.title = title
        item.isNoTitle = false
      }
    })
  }

  const changeIsFirstSession = (payload: boolean) => {
    isFirstSession.value = payload
  }

  const changeFirstQuery = (payload: string, mentionedItems: any[] = [], modelId: string = '', imageFiles: any[] = [], attachmentFiles: any[] = []) => {
    firstQuery.value = payload
    firstMentionedItems.value = mentionedItems
    firstModelId.value = modelId
    firstImageFiles.value = imageFiles
    firstAttachmentFiles.value = attachmentFiles
  }

  const setPrefillQuery = (q: string) => {
    prefillQuery.value = q
  }

  const consumePrefillQuery = () => {
    const q = prefillQuery.value
    prefillQuery.value = ''
    return q
  }

  return {
    menuArr,
    visibleMenuArr,
    isFirstSession,
    firstQuery,
    firstMentionedItems,
    firstModelId,
    firstImageFiles,
    firstAttachmentFiles,
    prefillQuery,
    clearMenuArr,
    updatemenuArr,
    updataMenuChildren,
    updatasessionTitle,
    changeIsFirstSession,
    changeFirstQuery,
    setPrefillQuery,
    consumePrefillQuery
  }
})
