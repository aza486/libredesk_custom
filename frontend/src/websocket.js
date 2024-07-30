import { useConversationStore } from './stores/conversation'

let socket
let reconnectInterval = 1000
let maxReconnectInterval = 30000
let reconnectTimeout
let isReconnecting = false
let manualClose = false

export function initWS () {
    let convStore = useConversationStore()

    // Initialize the WebSocket connection
    function initializeWebSocket () {
        socket = new WebSocket('ws://localhost:9009/api/ws')

        // Connection opened event
        socket.addEventListener('open', function () {
            reconnectInterval = 1000
            if (reconnectTimeout) {
                clearTimeout(reconnectTimeout);
                reconnectTimeout = null;
            }
        })

        // Listen for messages from the server
        socket.addEventListener('message', function (e) {
            if (e.data) {
                let event = JSON.parse(e.data)
                switch (event.type) {
                    case 'new_message': {
                        const message = event.data;
                        convStore.updateConversationLastMessage(message);
                        convStore.updateConversationMessageList(message);
                        break;
                    }
                    case 'message_prop_update': {
                        convStore.updateMessageProp(event.data);
                        break;
                    }
                    case 'new_conversation': {
                        convStore.addNewConversation(event.data);
                        break;
                    }
                    case 'conversation_prop_update': {
                        convStore.updateConversationProp(event.data);
                        break;
                    }
                    default: {
                        console.warn(`Unknown websocket event ${event.ev}`);
                    }
                }
            }
        });


        // Handle possible errors
        socket.addEventListener('error', function (event) {
            console.error('WebSocket error observed:', event)
        })

        // Handle the connection close event
        socket.addEventListener('close', function () {
            if (!manualClose) {
                reconnect()
            }
        })
    }


    // Start the initial WebSocket connection
    initializeWebSocket()

    // Reconnect logic
    function reconnect () {
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
            reconnectInterval = 1000
            reconnect()
        }
    })
}

function waitForWebSocketOpen (socket, callback) {
    if (socket.readyState === WebSocket.OPEN) {
        callback()
    } else {
        socket.addEventListener('open', function handler () {
            socket.removeEventListener('open', handler)
            callback()
        })
    }
}


export function sendMessage (message) {
    waitForWebSocketOpen(socket, () => {
        socket.send(JSON.stringify(message))
    })
}

export function subscribeConversations (type, filter) {
    let message = {
        action: 'conversations_sub',
        type: type,
        filter: filter
    }

    waitForWebSocketOpen(socket, () => {
        socket.send(JSON.stringify(message))
    })
}

export function subscribeConversation (uuid) {
    if (!uuid) {
        return
    }
    let message = {
        action: 'conversation_sub',
        uuid: uuid
    }

    waitForWebSocketOpen(socket, () => {
        socket.send(JSON.stringify(message))
    })
}

export function unsubscribeConversation (uuid) {
    if (!uuid) {
        return
    }
    let message = {
        action: 'conversation_unsub',
        uuid: uuid
    }

    waitForWebSocketOpen(socket, () => {
        socket.send(JSON.stringify(message))
    })
}
