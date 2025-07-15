import { defineNuxtPlugin } from '#app'

export default defineNuxtPlugin((nuxtApp) => {
  // 服务端渲染时的 Naive UI 配置
  if (process.server) {
    // 为 SSR 提供完整的 DOM API mock
    if (typeof global !== 'undefined' && !global.document) {
      const mockElement = {
        style: {},
        className: '',
        id: '',
        getAttribute: () => null,
        setAttribute: () => {},
        removeAttribute: () => {},
        appendChild: () => {},
        removeChild: () => {},
        insertBefore: () => {},
        querySelector: () => null,
        querySelectorAll: () => [],
        addEventListener: () => {},
        removeEventListener: () => {},
        dispatchEvent: () => true,
        getBoundingClientRect: () => ({
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          width: 0,
          height: 0
        }),
        scrollIntoView: () => {},
        focus: () => {},
        blur: () => {},
        click: () => {},
        offsetWidth: 0,
        offsetHeight: 0,
        offsetTop: 0,
        offsetLeft: 0,
        scrollTop: 0,
        scrollLeft: 0,
        clientWidth: 0,
        clientHeight: 0,
        children: [],
        childNodes: [],
        parentNode: null,
        firstChild: null,
        lastChild: null,
        nextSibling: null,
        previousSibling: null,
        nodeType: 1,
        nodeName: 'DIV',
        innerHTML: '',
        textContent: '',
        dataset: {}
      }

      // 创建完整的 document mock
      global.document = {
        head: {
          ...mockElement,
          appendChild: () => {},
          removeChild: () => {},
          insertBefore: () => {},
          querySelector: () => null,
          querySelectorAll: () => []
        },
        body: {
          ...mockElement,
          appendChild: () => {},
          removeChild: () => {},
          insertBefore: () => {},
          querySelector: () => null,
          querySelectorAll: () => []
        },
        documentElement: {
          ...mockElement,
          clientWidth: 1024,
          clientHeight: 768
        },
        createElement: (tagName: string) => ({
          ...mockElement,
          tagName: tagName.toUpperCase(),
          nodeName: tagName.toUpperCase()
        }),
        createTextNode: (text: string) => ({
          nodeType: 3,
          nodeName: '#text',
          textContent: text,
          data: text
        }),
        createDocumentFragment: () => ({
          ...mockElement,
          nodeType: 11,
          nodeName: '#document-fragment'
        }),
        getElementById: () => null,
        getElementsByTagName: () => [],
        getElementsByClassName: () => [],
        querySelector: () => null,
        querySelectorAll: () => [],
        addEventListener: () => {},
        removeEventListener: () => {},
        defaultView: null,
        readyState: 'complete'
      } as any

      // 创建 window mock
      if (!global.window) {
        global.window = {
          document: global.document,
          addEventListener: () => {},
          removeEventListener: () => {},
          getComputedStyle: () => ({
            getPropertyValue: () => '',
            top: '0px',
            left: '0px',
            position: 'static',
            zIndex: 'auto'
          }),
          requestAnimationFrame: (callback: Function) => setTimeout(callback, 16),
          cancelAnimationFrame: (id: number) => clearTimeout(id),
          innerWidth: 1024,
          innerHeight: 768,
          devicePixelRatio: 1,
          location: {
            href: 'http://localhost:3000',
            origin: 'http://localhost:3000',
            protocol: 'http:',
            host: 'localhost:3000',
            hostname: 'localhost',
            port: '3000',
            pathname: '/',
            search: '',
            hash: ''
          },
          history: {
            pushState: () => {},
            replaceState: () => {},
            back: () => {},
            forward: () => {},
            go: () => {}
          },
          navigator: {
            userAgent: 'Mozilla/5.0 (SSR)'
          },
          matchMedia: () => ({
            matches: false,
            addListener: () => {},
            removeListener: () => {}
          })
        } as any
      }
    }

    // Mock ResizeObserver for SSR
    if (!global.ResizeObserver) {
      global.ResizeObserver = class ResizeObserver {
        constructor(callback: Function) {}
        observe() {}
        unobserve() {}
        disconnect() {}
      } as any
    }

    // Mock IntersectionObserver for SSR
    if (!global.IntersectionObserver) {
      global.IntersectionObserver = class IntersectionObserver {
        constructor(callback: Function, options?: any) {}
        observe() {}
        unobserve() {}
        disconnect() {}
      } as any
    }
  }
})
