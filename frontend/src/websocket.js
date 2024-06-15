import { useConversationStore } from "./stores/conversation";

export function initWS () {
    let convStore = useConversationStore()

    // Create a new WebSocket connection to the specified WebSocket URL
    const socket = new WebSocket('ws://localhost:9009/api/ws');

    // Connection opened event
    socket.addEventListener('open', function (event) {
        // Send a message to the server once the connection is opened
        socket.send('Hello, server!!');
    });

    // Listen for messages from the server
    socket.addEventListener('message', function (e) {
        console.log('Message from server !', e.data);
        if (e.data) {
            let event = JSON.parse(e.data)
            switch (event.ev) {
                case "new_msg":
                    convStore.updateConversationList(event.d)
                    convStore.updateMessageList(event.d)
                    break
                case "msg_status_update":
                    convStore.updateMessageStatus(event.d.uuid, event.d.status)
                    break
                default:
                    console.log(`Unknown event ${event.ev}`);
            }
        }
    });

    // Handle possible errors
    socket.addEventListener('error', function (event) {
        console.error('WebSocket error observed:', event);
        console.log('WebSocket readyState:', socket.readyState); // Log the state of the WebSocket
    });


    // Handle the connection close event
    socket.addEventListener('close', function (event) {
        console.log('WebSocket connection closed!:', event);
    });

    socket.onerror = function (event) {
        console.error("WebSocket error:", event);
    };
    socket.onclose = function (event) {
        console.log("WebSocket connection closed:", event);
    };
}