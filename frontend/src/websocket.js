import { useConversationStore } from './stores/conversation';
import { CONVERSATION_WS_ACTIONS } from './constants/conversation';

let socket;
let reconnectInterval = 1000;
let maxReconnectInterval = 30000;
let reconnectTimeout;
let isReconnecting = false;
let manualClose = false;
let convStore;

function initializeWebSocket () {
  socket = new WebSocket('/ws');
  socket.addEventListener('open', handleOpen)
  socket.addEventListener('message', handleMessage)
  socket.addEventListener('error', handleError)
  socket.addEventListener('close', handleClose)
}

function handleOpen () {
  console.log('WebSocket connection established')
  reconnectInterval = 1000;
  if (reconnectTimeout) {
    clearTimeout(reconnectTimeout)
    reconnectTimeout = null;
  }
}

function handleMessage (event) {
  try {
    if (event.data) {
      const data = JSON.parse(event.data)
      switch (data.type) {
        case 'new_message':
          convStore.updateConversationLastMessage(data.data)
          convStore.updateConversationMessageList(data.data)
          break;
        case 'message_prop_update':
          convStore.updateMessageProp(data.data)
          break;
        case 'new_conversation':
          convStore.addNewConversation(data.data)
          break;
        case 'conversation_prop_update':
          convStore.updateConversationProp(data.data)
          break;
        default:
          console.warn(`Unknown websocket event type: ${data.type}`)
      }
    }
  } catch (error) {
    console.error('Error handling WebSocket message:', error)
  }
}

function handleError (event) {
  console.error('WebSocket error observed:', event)
}

function handleClose () {
  if (!manualClose) {
    reconnect()
  }
}

function reconnect () {
  if (isReconnecting) return;
  isReconnecting = true;
  reconnectTimeout = setTimeout(() => {
    initializeWebSocket()
    reconnectInterval = Math.min(reconnectInterval * 2, maxReconnectInterval)
    isReconnecting = false;
  }, reconnectInterval)
}

function setupNetworkListeners () {
  window.addEventListener('online', () => {
    if (!isReconnecting && socket.readyState !== WebSocket.OPEN) {
      reconnectInterval = 1000;
      reconnect()
    }
  })
}

export function initWS () {
  convStore = useConversationStore()
  initializeWebSocket()
  setupNetworkListeners()
}

function waitForWebSocketOpen (callback) {
  if (socket) {
    if (socket.readyState === WebSocket.OPEN) {
      callback()
    } else {
      socket.addEventListener('open', function handler () {
        socket.removeEventListener('open', handler)
        callback()
      })
    }
  }
}

export function sendMessage (message) {
  waitForWebSocketOpen(() => {
    socket.send(JSON.stringify(message))
  })
}

export function subscribeConversationsList (type) {
  const message = {
    action: CONVERSATION_WS_ACTIONS.SUB_LIST,
    type: type,
  }
  waitForWebSocketOpen(() => {
    socket.send(JSON.stringify(message))
  })
}

export function setCurrentConversation (uuid) {
  const message = {
    action: CONVERSATION_WS_ACTIONS.SET_CURRENT,
    uuid: uuid,
  }
  waitForWebSocketOpen(() => {
    socket.send(JSON.stringify(message))
  })
}

export function unsetCurrentConversation () {
  const message = {
    action: CONVERSATION_WS_ACTIONS.UNSET_CURRENT
  }
  waitForWebSocketOpen(() => {
    socket.send(JSON.stringify(message))
  })
}

export function closeWebSocket () {
  manualClose = true;
  if (socket) {
    socket.close()
  }
}