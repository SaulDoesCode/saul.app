{ /* global rilti localStorage Prism */
  const {once, dom} = rilti

  const see = () => {
    const {ul, li, a, article, header, nav, main, div, span, h1, p, img, section, html} = dom

    section.intro({$: 'body'},
      header(
        html(`<span class="token keyword">const</span> <span class="token punctuation">{</span> coder<span class="token punctuation">,</span> thinker <span class="token punctuation">}</span> <span class="token operator">=</span> saul`)
      ),
      div.contact(
        span(span({title: 'email me'}, 'email'), span('me@saul.app'))
      )
    )

    main.content({$: 'body'},
      article.ideas(
        header('ideas'),
        section(
          div.idea(
            header('nominalism as a programming best practise'),
            p(`
              Instead of universals (generalizable objects, classes, patterns...)
              view each construct or object as an individual which may share
              similarities with say another instance of it's class but is
              fundamentally particular and unrelated.
              All entities and phenomenon are specific it is our brains which
              with its optimisations for perception and memory that tries to
              model the world and fit old models onto novel experiences;
              in reality however, it would seem there is no such thing as
              relationships or generalizable repeatable patterns instead
              these are perceptual human abstractions to make sense of a world
              driven by complex rules, which we perhaps cannot know
              but attempt to model through experience
              e.g. a child raised in an environment entirely
              devoid of strong gravity may learn to view the world as a
              series of habitable capsules in a void, notions of up or down
              would be as meaningless to them as mug in space; in contrast
              those with experience of living in a strong gravity field take
              for granted that objects remain where you left them and the
              strong muscles and bones maintained by fighting gravity, however
              a getting used to rooms with no floor where all four walls are
              packed and used like the kitchen diner table of a large family.
              The point is if we are willing and able to differentiate
              preconceived patterns from the phenomenal reality we're faced with
              because only then do we get a chance to think outside the box,
              the box being the spotlight of projection
              (like calling all rodent-like animals rats
               because that's the closest mental model we have,
              seeing words and labels instead of the particular reality)

            `)
          )
        )
      ),
      article.code(
        header('code'),
        section(
          a({
            href: 'https://github.com/SaulDoesCode/rilti.js',
            target: '_blank'
          }, 'rilti - framework')
        )
      )
    )
  }

  if (!localStorage.getItem('see')) {
    once.click('div.come-see', e => {
      document.body.textContent = ''
      document.body.className = 'transition-view'
      see()
      setTimeout(() => { document.body.className = '' }, 600)
      localStorage.setItem('see', true)
    })
  } else {
    document.body.textContent = ''
    see()
  }
}
