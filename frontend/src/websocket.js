import { useConversationStore } from './stores/conversation'

let socket
let reconnectInterval = 1000 // Initial reconnection interval
let maxReconnectInterval = 30000 // Maximum reconnection interval
let reconnectTimeout
let isReconnecting = false
let manualClose = false

export function initWS() {
    let convStore = useConversationStore()

    // Initialize the WebSocket connection
    function initializeWebSocket() {
        socket = new WebSocket('ws://localhost:9009/api/ws')

        // Connection opened event
        socket.addEventListener('open', function () {
            console.log('WebSocket connection established')
            reconnectInterval = 1000 // Reset the reconnection interval
            if (reconnectTimeout) {
                clearTimeout(reconnectTimeout) // Clear any existing reconnection timeout
                reconnectTimeout = null
            }
        })

        // Listen for messages from the server
        socket.addEventListener('message', function (e) {
            console.log('Message from server:', e.data)
            if (e.data) {
                let event = JSON.parse(e.data)
                // TODO: move event type to consts.
                switch (event.typ) {
                    case 'new_msg':
                        convStore.updateConversationList(event.d)
                        convStore.updateMessageList(event.d)
                        break
                    case 'msg_prop_update':
                        convStore.updateMessageProp(event.d)
                        break
                    case 'new_conv':
                        convStore.addNewConversation(event.d)
                        break
                    case 'conv_prop_update':
                        convStore.updateConversationProp(event.d)
                        break
                    default:
                        console.log(`Unknown event ${event.ev}`)
                }
            }
        })

        // Handle possible errors
        socket.addEventListener('error', function (event) {
            console.error('WebSocket error observed:', event)
        })

        // Handle the connection close event
        socket.addEventListener('close', function (event) {
            console.log('WebSocket connection closed:', event)
            if (!manualClose) {
                reconnect()
            }
        })
    }

    // Start the initial WebSocket connection
    initializeWebSocket()

    // Reconnect logic
    function reconnect() {
        if (isReconnecting) return
        isReconnecting = true
        reconnectTimeout = setTimeout(() => {
            initializeWebSocket()
            reconnectInterval = Math.min(reconnectInterval * 2, maxReconnectInterval)
            isReconnecting = false
        }, reconnectInterval)
    }

    // Detect network status and handle reconnection
    window.addEventListener('online', () => {
        if (!isReconnecting && socket.readyState !== WebSocket.OPEN) {
            reconnectInterval = 1000 // Reset reconnection interval
            reconnect()
        }
    })
}

function waitForWebSocketOpen(socket, callback) {
    if (socket.readyState === WebSocket.OPEN) {
        callback()
    } else {
        socket.addEventListener('open', function handler() {
            socket.removeEventListener('open', handler)
            callback()
        })
    }
}

export function sendMessage(message) {
    waitForWebSocketOpen(socket, () => {
        socket.send(JSON.stringify(message))
    })
}

export function subscribeConversations(type, preDefinedFilter) {
    let message = {
        a: 'conversations_sub',
        t: type,
        pf: preDefinedFilter
    }

    waitForWebSocketOpen(socket, () => {
        socket.send(JSON.stringify(message))
    })
}

export function subscribeConversation(uuid) {
    if (!uuid) {
        return
    }
    let message = {
        a: 'conversation_sub',
        uuid: uuid
    }

    waitForWebSocketOpen(socket, () => {
        socket.send(JSON.stringify(message))
    })
}

export function unsubscribeConversation(uuid) {
    if (!uuid) {
        return
    }
    let message = {
        a: 'conversation_unsub',
        uuid: uuid
    }

    waitForWebSocketOpen(socket, () => {
        socket.send(JSON.stringify(message))
    })
}
