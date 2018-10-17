{
const {dom, $} = rilti
const {div, button, img, h1, h2, h3, h4, input, label, span, section, aside, article, header, html} = dom

/*
const converter = new showdown.Converter({openLinksInNewWindow: true, tasklists: true})
const md2html = (md, plain) => plain ? converter.makeHtml(md) : html(converter.makeHtml(md))
*/

const isEmail = email => isEmail.re.test(String(email).toLowerCase())
isEmail.re = /^(([^<>()\[\]\\.,;:\s@"]+(\.[^<>()\[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/

const isUsername = username => isUsername.re.test(String(username))
isUsername.re = /^[a-zA-Z0-9._-]{3,50}$/

const checkUsername = async username => {
  console.time()
  const res = await fetch('/check-username/' + username)
  const data = await res.json()
  console.log(`The username ${username} is: `, data)
  console.timeEnd()
  return data
}

const authenticate = async (email, username) => {
  if (!isEmail(email)) {
    return
  }
  if (!isUsername(username)) {
    return
  }
  authenticate.busy = true
  if (!(await checkUsername(username)).ok) {
    console.log('returing user')
  }
  try {
    console.log(`Awaiting Authentication for ${username}`)
    console.time()
    const res = await fetch('/auth', {
      method: 'POST',
      body: JSON.stringify({email, username})
    })
    const data = await res.json()
    console.log(`The verdict is: `, data)
    console.timeEnd()
  } catch(e) {}
  authenticate.busy = false
}

const authbtn = section.authbtn.icon_lock({
  $: 'header.hero',
  props: {
    open: false
  },
  onclick(e, el) {
    const open = el.open = !el.open
    el.class({
      'icon-lock': !open,
      'icon-lock-open': open
    })
    open ? authform.appendTo('header.hero') : authform.remove()
  }
})

const authform = section.auth(
  div.inputs(
    input({
      type: "checkbox",
      name: "ignore_the_starman_enforcing_anti_spam",
      value: "1",
      attr: {
        style: "display:none !important",
        tabindex: "-1",
        autocomplete: "off"
      }
    }),
    div.username(
      label({attr: {for:'username'}}, 'username'),
      authenticate.username = input({
        id: "username",
        type: 'text',
        name: 'username',
        title: 'username',
        autocomplete: 'username',
        placeholder: ' ',
        pattern: '[a-zA-Z0-9._-]{3,50}',
        required: 'required'
      })
    ),
    div.email(
      label({attr: {for: 'email'}}, 'email'),
      authenticate.email = input({
        id: 'email',
        type: 'email',
        name: 'email',
        title: 'email',
        autocomplete: 'email',
        placeholder: ' ',
        required: 'required'
      })
    )
  ),
  authenticate.button = button.submit({
    onclick(e) {
      if (!authenticate.busy) {
        authenticate(authenticate.email.value.trim(), authenticate.username.value.trim())
        authform.remove()
      }
    }
  }, 'Go')
)

}

