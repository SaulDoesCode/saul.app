const {dom, each, $} = rilti
const {html, article, div, nav, header, section, span, a} = dom

const Ideas = {
  'nominalism': `Awareness of universals' non existence. Every object is discrete, even a class instance because no 2 objects exist in the
    same exact place, time and way. Relationships, between objects or phenomena, is a mental abstraction helping us construct
    a rulebook, narrative of events and a sense of contextual significance and state in terms of what we are able to perceive,
    know and the extent to which we can conceptualize, integrate, understand and apply it applying and projecting ourselves
    at the world thereby. <br> Likewise no two problems (errors, design, creative) are the same, each varies ever so slightly. Therefore
    by being consciously aware, that what we perceive is directly influenced by the abstractions of our brains, learned knowledge
    and beliefs, we gain the opportunity to sift throw what we are and see more clearly what is there; climbing out the proverbial
    box. Note universals are useful as a means to understand the chaos of a reality, but it is not reality itself, observe
    for example the way facts change as either as new knowledge is gained changing the model or sociocultural pressures or change
    alters the general perception of how or if fact is fact.`,
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

section.ideas({$: 'body'},
  ({state}) => {
    const title = header(span('ideas '), state`: ${'active'}`)
    const display = article()
    const list = div.ideas({
      onclick: ({target}) => { 
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
  }
)