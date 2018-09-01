const {dom, each, $} = rilti
const {html, b, button, label, input, article, div, nav, header, section, span, a} = dom

const checkUsername = async username => {
  console.time()
  const res = await fetch('/check-username/' + username)
  const data = await res.json()
  console.log(`The username ${username} is: `, data)
  console.timeEnd()
  return data
}

const authenticate = async (email, username) => {
  if (!(await checkUsername(username)).ok) console.log('returing user')
  console.log(`Awaiting Authentication for ${username}`)
  console.time()
  const res = await fetch('/auth', {
    method: 'POST',
    body: JSON.stringify({email, username})
  })
  const data = await res.json()
  console.log(`The verdict is: `, data)
  console.timeEnd()
}


const authform = section.auth({
  $: 'body',
  state: {
    email: localStorage.getItem('$email') || '',
    username: localStorage.getItem('$username') || ''
  }
}, ({state}) => [
  div.email(
    label('email'),
    input({
      type: 'email',
      name: 'email',
      autocomplete: 'email',
      required: true
    }, state.$email)
  ),
  div.username(
    label('username'),
    input({
      type: 'text',
      name: 'username',
      pattern: '[a-zA-Z0-9._-]{3,50}',
      autocomplete: 'username',
      required: true
    }, state.$username)
  ),
  button.submit({
    onclick (e) { authenticate(state.email, state.username) }
  }, 'Go!')
])
