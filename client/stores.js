import {readable} from 'svelte/store'

import * as toast from './toast'

var list = []
var roomName = null

var setRoom, setMessages

export const room = readable(roomName, set => {
  setRoom = set
})

export const messages = readable(list, set => {
  setMessages = set
})

var es

export function joinRoom(roomName) {
  setRoom(roomName)

  if (es) {
    es.close()
  }

  es = new EventSource(`/${window.btoa(roomName)}/receive`)
  es.onerror = e => {
    console.log('sse error', e.data)
    list = []
    setMessages(list)
  }
  es.addEventListener('error', e => {
    toast.error(e.data)
  })
  es.addEventListener('reset', e => {
    list = []
    setMessages(list)
  })
  es.addEventListener('message', e => {
    let message = e.data
    list = [parseMessage(message), ...list]
    setMessages(list)
  })

  return () => {
    es.close()
  }
}

function parseMessage(data) {
  try {
    let [user, message, time] = JSON.parse(data)
    return {user, message, time}
  } catch (e) {
    return {user: '', message: data, time: 0}
  }
}
