import {readable} from 'svelte/store'

import * as toast from './toast'

export function getMessages(room) {
  var list = []

  return readable(list, set => {
    const es = new EventSource(`/${room}/receive`)
    es.onerror = e => console.log('contract sse error', e.data)
    es.addEventListener('error', e => {
      toast.error(e.data)
    })
    es.addEventListener('message', e => {
      let message = e.data
      list = [message, ...list]
      set(list)
    })

    return () => {
      es.close()
    }
  })
}
