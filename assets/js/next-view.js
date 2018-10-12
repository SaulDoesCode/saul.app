/* global localStorage fetch */
const {dom, each, $} = rilti
const {html, article, div, nav, header, section, span, a} = dom

const converter = new showdown.Converter({openLinksInNewWindow: true, tasklists: true})

const persp = section.perspective({$: 'main'})
const lexi = section.lexicon({$: 'main'})

fetch('https://rawgit.com/SaulDoesCode/perspective/master/README.md')
  .then(res => res.text()).then(perspective => {
    persp.html = converter.makeHtml(perspective)
    setTimeout(() => {
      try {
        $('a[href="./Lexicon.md"]', persp).attr({
          href: '#lexicondefinitionsforallthenonsense',
          target: null,
          rel: null
        })
      } catch (e) {}
    }, 250)
  })

fetch('https://rawgit.com/SaulDoesCode/perspective/master/Lexicon.md')
  .then(res => res.text()).then(lexicon => {
    lexi.html = converter.makeHtml(lexicon)
  })

/*
const Ideas = {
  'nominalism': `(name + ism -> nominalism) - Awareness of universals' non existence. We see concepts and labels not objects as such, even though every object is discrete and particular; even species or class instances.
    Because no two objects exist in the same manner, time or place. Relationships, between objects or phenomena are, therefore, a mental abstraction helping us construct
    a rulebook, narrative of events, or, a sense of contextual significance and state in terms of what we are able to perceive,
    know and the extent to which we can conceptualize, integrate, understand, and apply it thereby applying and projecting ourselves
    at the world thereby. <br> Likewise no two problems (errors, design, creative) are the same, each varies slightly. Therefore
    by being consciously aware, that what we perceive is directly influenced by the abstractions of our brains, learned knowledge
    and beliefs, we gain the opportunity to sift throw what we are and see more clearly what is there; climbing out the proverbial
    box. Note universals are useful as a means to understand the chaos of a reality, but it is not reality itself, observe
    for example the way facts change as either as new knowledge is gained changing the model, or, when sociocultural drift/mutation
    alters the general perception of how or if fact is fact. (paradigm shifts)`,
  'organic knowledge': `Not meaning knowledge as representation, familiarity, or, justified true belief, but, rather unity in composure, internal
    operation, and, outward function; biological knowledge. In this way organic knowledge might be viewed as the manifest will
    of systems programming, where the system is not self-aware, yet, handles and maintains internal state, operations, and, state
    mutations emanating from external sources or sub-systems to the effect and extent of contextual external efficacy.`,
  'state': `Points of variability representing conditions in reality (as perceived) or systems from within a larger conceptualization
    such as one’s subjective worldview. With the representational transfer thereof, state, could be used to fill the role of
    knowledge as an abstraction, alongside impressions, like undigested images or sound, which have no state (aside from what
    it is), but, from which state could be derived (interpretation). Thereby bypassing the expectations and problems of truth
    conditions and what belief and subjectivity entail. State is a conceptualization directly bound to the operation and outward
    functioning of organisms, systems, and, how they change. In this way state can operate entirely with in abstract systems
    without the explicit requirement of a real reality. State could be thought of as the minimum surface area of awareness necessary to perform or
    exert a transaction or manner of operation within a larger context.`
}

section.ideas({$: 'main'}, ({state}) => {
  const title = header(span('ideas '), state`: ${'active'}`)
  const display = article()
  const list = div.ideas({
    onclick ({target}) {
      if (target.matches('span.idea')) state.active = target.textContent
    }
  })

  state.bind('active', (name, old) => {
    const idea = Ideas[name]
    if (idea == null) {
      if (old != null) state.active = old
      return
    }
    display.html = idea
    if (state.activeIdea) state.activeIdea.class('active', false)
    state.activeIdea = state.ideas[name].class('active', true)
  })

  state.ideas = {}
  for (const name of Object.keys(Ideas)) {
    list.append(state.ideas[name] = span.idea(name))
    if (!state.active) state.active = name
  }

  return [title, list, display]
})
*/