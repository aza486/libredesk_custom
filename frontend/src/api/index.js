import axios from 'axios'
import qs from 'qs'

const http = axios.create({
    timeout: 10000,
    responseType: "json",
})

// Request interceptor.
http.interceptors.request.use((request) => {
    // Set content type for POST/PUT requests.
    if ((request.method === "post" || request.method === "put") && !request.headers["Content-Type"]) {
        request.headers["Content-Type"] = "application/x-www-form-urlencoded"
        request.data = qs.stringify(request.data)
    }
    return request
})

const login = (data) => http.post(`/api/login`, data)
const getTeams = () => http.get("/api/teams")
const getUsers = () => http.get("/api/users")
const getCurrentUser = () => http.get("/api/users/me")
const getTags = () => http.get("/api/tags")
const upsertTags = (uuid, data) => http.post(`/api/conversation/${uuid}/tags`, data)
const updateAssignee = (uuid, assignee_type, data) => http.put(`/api/conversation/${uuid}/assignee/${assignee_type}`, data)
const updateStatus = (uuid, data) => http.put(`/api/conversation/${uuid}/status`, data)
const updatePriority = (uuid, data) => http.put(`/api/conversation/${uuid}/priority`, data)
const updateAssigneeLastSeen = (uuid) => http.put(`/api/conversation/${uuid}/last-seen`)
const getMessage = (uuid) => http.get(`/api/message/${uuid}`)
const retryMessage = (uuid) => http.get(`/api/message/${uuid}/retry`)
const getMessages = (uuid) => http.get(`/api/conversation/${uuid}/messages`)
const sendMessage = (uuid, data) => http.post(`/api/conversation/${uuid}/message`, data)
const getConversation = (uuid) => http.get(`/api/conversation/${uuid}`)
const getConversationParticipants = (uuid) => http.get(`/api/conversation/${uuid}/participants`)
const getCannedResponses = () => http.get('/api/canned-responses')
const getAssigneeStats = () => http.get('/api/conversations/assignee/stats')

const getAssignedConversations = (page, preDefinedFilter) => http.get(`/api/conversations/assigned?page=${page}&predefinedfilter=${preDefinedFilter}`)
const getUnassignedConversations = (page, preDefinedFilter) => http.get(`/api/conversations/unassigned?page=${page}&predefinedfilter=${preDefinedFilter}`)
const getAllConversations = (page, preDefinedFilter) => http.get(`/api/conversations/all?page=${page}&predefinedfilter=${preDefinedFilter}`)

const uploadAttachment = (data) => http.post('/api/attachment', data, {
    headers: {
        'Content-Type': 'multipart/form-data'
    }
})

export default {
    login,
    getTags,
    getTeams,
    getUsers,
    getConversation,
    getAssignedConversations,
    getUnassignedConversations,
    getAllConversations,
    getAssigneeStats,
    getConversationParticipants,
    getMessage,
    getMessages,
    sendMessage,
    getCurrentUser,
    updateAssignee,
    updateStatus,
    updatePriority,
    upsertTags,
    retryMessage,
    updateAssigneeLastSeen,
    getCannedResponses,
    uploadAttachment,
}
