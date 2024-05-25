import axios from 'axios';
import qs from 'qs';

const http = axios.create({
    timeout: 10000,
    responseType: "json",
});

// Request interceptor.
http.interceptors.request.use((request) => {
    // Set content type for POST/PUT requests.
    if ((request.method === "post" || request.method === "put") && !request.headers["Content-Type"]) {
        request.headers["Content-Type"] = "application/x-www-form-urlencoded"
        request.data = qs.stringify(request.data)
    }
    return request
})

const login = (data) => http.post(`/api/login`, data);
const getTeams = () => http.get("/api/teams")
const getAgents = () => http.get("/api/agents")
const getAgentProfile = () => http.get("/api/profile")
const getTags = () => http.get("/api/tags")
const upsertTags = (uuid, data) => http.post(`/api/conversation/${uuid}/tags`, data);
const updateAssignee = (uuid, assignee_type, data) => http.put(`/api/conversation/${uuid}/assignee/${assignee_type}`, data);
const updateStatus = (uuid, data) => http.put(`/api/conversation/${uuid}/status`, data);
const updatePriority = (uuid, data) => http.put(`/api/conversation/${uuid}/priority`, data);
const getMessages = (uuid) => http.get(`/api/conversation/${uuid}/messages`);
const getConversation = (uuid) => http.get(`/api/conversation/${uuid}`);
const getConversations = () => http.get('/api/conversations');
const getCannedResponses = () => http.get('/api/canned_responses');

export default {
    login,
    getTags,
    getTeams,
    getAgents,
    getConversation,
    getConversations,
    getMessages,
    getAgentProfile,
    updateAssignee,
    updateStatus,
    updatePriority,
    upsertTags,
    getCannedResponses,
}
