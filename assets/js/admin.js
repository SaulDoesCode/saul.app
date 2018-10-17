/* global localStorage fetch */
try { localStorage.getItem('cookie-ok') && document.querySelector('.cookie-toast').remove() } catch(e) {}

{ 
  const infinify = (fn, ref = false) => new Proxy(fn, {get: (fn, key) => ref && key in fn ? fn[key] : fn.bind(null, key)})
  var emitter = ((host = {}, listeners = new Map()) => Object.assign(host, {
    listeners,
    emit: infinify((event, ...data) => {
      if (listeners.has(event)) {
        for (const h of listeners.get(event)) {
          h.apply(undefined, data)
        }
      }
    }),
    emitAsync: infinify((event, ...data) => setTimeout(() => {
      if (listeners.has(event)) {
        for (const h of listeners.get(event)) {
          setTimeout(h, 0, ...data)
        }
      }
    }, 0), false),
    on: infinify((event, handler) => {
      if (!listeners.has(event)) listeners.set(event, new Set())
      listeners.get(event).add(handler)
      const manager = () => host.off(event, handler)
      manager.off = manager
      manager.on = () => {
        manager()
        return host.on(event, handler)
      }
      manager.once = () => {
        manager()
        return host.once(event, handler)
      }
      return manager
    }),
    once: infinify((event, handler) => host.on(event, function h() {
      handler(...arguments)
      host.off(event, h)
    })),
    off: infinify((event, handler) => {
      if (listeners.has(event)) {
        const ls = listeners.get(event)
        ls.delete(handler)
        if (!ls.size) listeners.delete(event)
      }
    })
  }))()
}

rilti.run(() => {

const bakeWrit = window.bakeWrit = (writ = {
  title: 'First post',
  slug: 'first-post',
  markdown: '# Post Number 1',
  content: '<h1>Post Number 1</h1>',
  description: 'post 1',
  author: 'Saul',
  tags: ['status'],
  membersonly: true
}, fn, errfn) => {

  fetch('/writ', {
    method: 'POST',
    body: JSON.stringify(writ)
  }).then(res => res.json(), err => {
    console.error(`Writ problem:`, err)
    errfn && errfn(err)
  }).then(res => {
    if (res.err || res.error) {
      console.error(`Writ problem:`, res.err || res.error)
      errfn && errfn(res.err || res.error, res)
    } else {
      console.log(`Writ Success!:`, res)
      fn && fn(res)
    }
  })

}

const queryWrits = window.queryWrits = (query = {}, fn) => {
  if (!('editormode' in query)) query.editormode = true
  fetch('/writ/query', {
    method: 'POST',
    body: JSON.stringify(query)
  }).then(res => res.json(), err => {
    console.error(`Writ Query problem:`, err)
  }).then(res => {
    if (res.err || res.error) return console.error(`Writ Query Problem:`, res.err || res.error)
    console.log(`Writ Success!:`, res)
    fn && fn(res)
  })
}

const {dom, each, $} = rilti
const {aside, html, main, body, b, button, h1, h2, h3, label, input, pre, article, div, nav, header, section, span, a} = dom

const converter = new showdown.Converter({openLinksInNewWindow: true, tasklists: true})
var md2html = (md, plain) => plain ? converter.makeHtml(md) : html(converter.makeHtml(md))

const state = {
  get currentHash() { return location.hash.substr(1) },
  set area(v) {
    if (state.currentArea === v) return
    for (const area of areaSelector.$children) {
      if (area.txt === v) {
        if (state.areaSelection) state.areaSelection.class({active: false})
        emitter.emit.areaChange(
          location.hash = state.currentArea = (state.areaSelection = area.class({active: true})).txt
        )
      }
    }
  },
  get area() {return state.currentArea},
  get writ() {
    return state.currentWrit
  },
  set writ(v) {
    emitter.emit.writEdit(state.currentWrit = v)
  }
}

const saveWrit = writ => {
  bakeWrit(writ, res => {
    const q = {one: true}
    if (writ._key) q._key = writ._key
    else q.title = writ.title
    queryWrits(q, w => {
      Object.assign(writ)
      if (state.writ === writ) emitter.emit.writEdit(writ)
    })
  })
}

const areas = {}
const areaSelector = nav({
  $: 'body',
  onclick(e, el) {
    if (!e.target.matches('div.area')) return
    state.area = e.target.textContent
  }
}, 'editor stats users'.split(' ').map(area => div.area(area)))

areas.editor = section.editor(editor => [
  header(
    input.writ_title({
      type: 'text',
      placeholder: 'title',
      contentEditable: true,
      title: 'title'
    }, el => {
      const updateTitle = () => {
        if (!state.writ) return
        state.writ.title = el.value
      }
      el.on.input(updateTitle)
      el.on.keydown(updateTitle)
      el.on.blur(updateTitle)
      el.on.focus(updateTitle)
      emitter.on.writEdit(writ => {el.value = writ.title})
    }),
    div.saveWrit['icon-floppy'](el => {
      el.on.click(e => {
        saveWrit(state.writ)
      })
    }),
    div.togglepreview(el => {
      el.on.click(e => {
        emitter.emit.preview(editor.class.preview = !editor.class.preview)
      })
      const edit = 'editing'
      const view = 'preview'
      emitter.on.preview(toggle => {
        if (editor.class.preview !== toggle) editor.class.preview = toggle
        el.class({
          'icon-eye': toggle,
          'icon-cog-alt': !toggle
        })
        el.title = toggle ? view : edit
      })
      el.class('icon-cog-alt').title = edit
    })
  ),
  section.writersblock(wb => {
    emitter.on.preview(preview => wb.class({preview}))
    pre({
      $: wb,
      contentEditable: true
    }, el => {
      const updateMD = () => {
        if (!state.writ) return
        state.writ.markdown = el.innerText
      }
      el.on.input(updateMD)
      el.on.keydown(updateMD)
      el.on.blur(updateMD)
      el.on.focus(updateMD)

      emitter.on.writEdit(writ => {
        el.innerText = writ.markdown
      })
    }),
    div.view({$: wb}, viewer => {
      emitter.on.preview(preview => {
        if (state.writ) viewer.html = md2html(state.writ.markdown)
      })
    })
  }),
  aside.writselector(el => {
    const populate = writs => {
      for (const writ of writs) div({$: el}, writ.title)
    }
    state.writsReady ? populate(state.writs) : emitter.on.writsReady(populate)
  })
])

const display = main({$: 'body'})

emitter.on.areaChange(area => {
  display.html = areas[area]
})

rilti.run(() => {
  if (!state.currentArea) state.area = 'editor'
  queryWrits({}, writs => {
    emitter.emit.writsReady(state.writs = writs, state.writsReady = true)
    if (!state.writ) state.writ = state.writs[0]
  })
})
})