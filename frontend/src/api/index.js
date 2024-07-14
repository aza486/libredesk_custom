import axios from 'axios'
import qs from 'qs'

const http = axios.create({
  timeout: 10000,
  responseType: 'json'
})

// Request interceptor.
http.interceptors.request.use((request) => {
  // Set content type for POST/PUT requests if the content type is not set.
  if ((request.method === 'post' || request.method === 'put') && !request.headers['Content-Type']) {
    request.headers['Content-Type'] = 'application/x-www-form-urlencoded'
    request.data = qs.stringify(request.data)
  }
  return request
})

const login = (data) => http.post(`/api/login`, data)
const getUser = (id) => http.get(`/api/users/${id}`)
const getTeam = (id) => http.get(`/api/teams/${id}`)
const getTeams = () => http.get('/api/teams')
const getUsers = () => http.get('/api/users')
const getCurrentUser = () => http.get('/api/users/me')
const getTags = () => http.get('/api/tags')
const upsertTags = (uuid, data) => http.post(`/api/conversations/${uuid}/tags`, data)
const updateAssignee = (uuid, assignee_type, data) =>
  http.put(`/api/conversations/${uuid}/assignee/${assignee_type}`, data)
const updateStatus = (uuid, data) => http.put(`/api/conversations/${uuid}/status`, data)
const updatePriority = (uuid, data) => http.put(`/api/conversations/${uuid}/priority`, data)
const updateAssigneeLastSeen = (uuid) => http.put(`/api/conversations/${uuid}/last-seen`)
const getMessage = (uuid) => http.get(`/api/message/${uuid}`)
const retryMessage = (uuid) => http.get(`/api/message/${uuid}/retry`)
const getMessages = (uuid) => http.get(`/api/conversations/${uuid}/messages`)
const sendMessage = (uuid, data) => http.post(`/api/conversations/${uuid}/messages`, data)
const getConversation = (uuid) => http.get(`/api/conversations/${uuid}`)
const getConversationParticipants = (uuid) => http.get(`/api/conversations/${uuid}/participants`)
const getCannedResponses = () => http.get('/api/canned-responses')
const getAssignedConversations = (page, filter) =>
  http.get(`/api/conversations/assigned?page=${page}&filter=${filter}`)
const getTeamConversations = (page, filter) =>
  http.get(`/api/conversations/team?page=${page}&filter=${filter}`)
const getAllConversations = (page, filter) =>
  http.get(`/api/conversations/all?page=${page}&filter=${filter}`)
const uploadAttachment = (data) =>
  http.post('/api/attachment', data, {
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
const uploadFile = (data) =>
  http.post('/api/attachment', data, {
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
const getUserDashboardCounts = () => http.get('/api/dashboard/me/counts')
const getUserDashoardCharts = () => http.get('/api/dashboard/me/charts')
const getLanguage = (lang) => http.get(`/api/lang/${lang}`)
const createUser = (data) => http.post('/api/users', data)
const updateUser = (id, data) => http.put(`/api/users/${id}`, data)
const updateTeam = (id, data) => http.put(`/api/teams/${id}`, data)
const createInbox = (data) =>
  http.post('/api/inboxes', data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const getInboxes = () => http.get('/api/inboxes')
const getInbox = (id) => http.get(`/api/inboxes/${id}`)
const updateInbox = (id, data) =>
  http.put(`/api/inboxes/${id}`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const deleteInbox = (id) => http.delete(`/api/inboxes/${id}`)

export default {
  login,
  getInbox,
  getUser,
  getInboxes,
  getLanguage,
  getTags,
  getTeams,
  getTeam,
  getUsers,
  getConversation,
  getAssignedConversations,
  getTeamConversations,
  getAllConversations,
  getUserDashboardCounts,
  getConversationParticipants,
  getMessage,
  getMessages,
  getUserDashoardCharts,
  getCurrentUser,
  getCannedResponses,
  updateAssignee,
  updateStatus,
  updatePriority,
  upsertTags,
  uploadFile,
  updateAssigneeLastSeen,
  uploadAttachment,
  updateUser,
  sendMessage,
  retryMessage,
  createUser,
  createInbox,
  updateInbox,
  deleteInbox
}
