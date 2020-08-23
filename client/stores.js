import {readable} from 'svelte/store'

import * as toast from './toast'

export function getMessages(room) {
  var list = []

  return readable(list, set => {
    const es = new EventSource(`/${room}/receive`)
    es.onerror = e => console.log('sse error', e.data)
    es.addEventListener('error', e => {
      toast.error(e.data)
    })
    es.addEventListener('reset', e => {
      console.log('reset')
      list = []
      set(list)
    })
    es.addEventListener('message', e => {
      let message = e.data
      list = [parseMessage(message), ...list]
      set(list)
    })

    return () => {
      es.close()
    }
  })
}

function parseMessage(data) {
  try {
    let [user, message, time] = JSON.parse(data)
    return {user, message, time}
  } catch (e) {
    return {user: '', message: data, time: 0}
  }
}
