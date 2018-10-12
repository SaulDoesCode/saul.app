{
const {div, h1, span, section, aside, article, header, html} = rilti.dom

const converter = new showdown.Converter({openLinksInNewWindow: true, tasklists: true})

const lexicon = {
  'closure': 'a closing of openness, like a model but more ontological',
  'extrate': 'the direct unknowable externality of a subject'
}
for (const key in lexicon) lexicon[key] = html(converter.makeHtml(lexicon[key]))

section.lexicon({
  $: 'body',
  methods: {
    concept ({state}, key) {
      if (key in lexicon) state.content = lexicon[state.active = key]
    }
  },
  cycle: { mount: l => l.concept(Object.keys(lexicon)[0]) }
}, ({state, concept}) => [
  aside({
    onclick: e => concept(e.target.textContent)
  },
    Object.keys(lexicon).map(key => div(key))
  ),
  h1(state.bind.text('active')),
  article(el => { state.$content(c => el.html = c) })
])


}

