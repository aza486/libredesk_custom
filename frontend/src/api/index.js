import axios from 'axios'
import qs from 'qs'

const http = axios.create({
  timeout: 10000,
  responseType: 'json'
})

// Function to extract CSRF token from cookies
function getCSRFToken() {
  const name = 'csrf_token=';
  const cookies = document.cookie.split(';');
  for (let i = 0; i < cookies.length; i++) {
      let c = cookies[i].trim();
      if (c.indexOf(name) === 0) {
          return c.substring(name.length, c.length);
      }
  }
  return '';
}

// Request interceptor.
http.interceptors.request.use((request) => {
  // Add csrf token
  const token = getCSRFToken()
  if (token) {
    request.headers['X-CSRFTOKEN'] = token
  }

  // Set content type for POST/PUT requests if the content type is not set.
  if ((request.method === 'post' || request.method === 'put') && !request.headers['Content-Type']) {
    request.headers['Content-Type'] = 'application/x-www-form-urlencoded'
    request.data = qs.stringify(request.data)
  }
  return request
})

const getEmailNotificationSettings = () => http.get('/api/settings/notifications/email')
const updateEmailNotificationSettings = (data) => http.put('/api/settings/notifications/email', data)
const getPriorities = () => http.get('/api/priorities')
const getStatuses = () => http.get('/api/statuses')
const createStatus = (data) => http.post('/api/statuses', data)
const updateStatus = (id, data) => http.put(`/api/statuses/${id}`, data)
const deleteStatus = (id) => http.delete(`/api/statuses/${id}`)
const createTag = (data) => http.post('/api/tags', data)
const updateTag = (id, data) => http.put(`/api/tags/${id}`, data)
const deleteTag = (id) => http.delete(`/api/tags/${id}`)
const getTemplate = (id) => http.get(`/api/templates/${id}`)
const getTemplates = () => http.get('/api/templates')
const createTemplate = (data) =>
  http.post('/api/templates', data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const deleteTemplate = (id) => http.delete(`/api/templates/${id}`)
const updateTemplate = (id, data) =>
  http.put(`/api/templates/${id}`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const createOIDC = (data) =>
  http.post('/api/oidc', data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const getAllOIDC = () => http.get('/api/oidc')
const getOIDC = (id) => http.get(`/api/oidc/${id}`)
const updateOIDC = (id, data) =>
  http.put(`/api/oidc/${id}`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const deleteOIDC = (id) => http.delete(`/api/oidc/${id}`)
const updateSettings = (key, data) =>
  http.put(`/api/settings/${key}`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const getSettings = (key) => http.get(`/api/settings/${key}`)
const login = (data) => http.post(`/api/login`, data)
const getAutomationRules = (type) =>
  http.get(`/api/automation/rules`, {
    params: { type: type }
  })
const toggleAutomationRule = (id) => http.put(`/api/automation/rules/${id}/toggle`)
const getAutomationRule = (id) => http.get(`/api/automation/rules/${id}`)
const updateAutomationRule = (id, data) =>
  http.put(`/api/automation/rules/${id}`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const createAutomationRule = (data) =>
  http.post(`/api/automation/rules`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const getRoles = () => http.get('/api/roles')
const getRole = (id) => http.get(`/api/roles/${id}`)
const createRole = (data) =>
  http.post('/api/roles', data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const updateRole = (id, data) =>
  http.put(`/api/roles/${id}`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const deleteRole = (id) => http.delete(`/api/roles/${id}`)
const deleteAutomationRule = (id) => http.delete(`/api/automation/rules/${id}`)
const getUser = (id) => http.get(`/api/users/${id}`)
const getTeam = (id) => http.get(`/api/teams/${id}`)
const getTeams = () => http.get('/api/teams')
const getTeamsCompact = () => http.get('/api/teams/compact')
const getUsers = () => http.get('/api/users')
const getUsersCompact = () => http.get('/api/users/compact')
const updateCurrentUser = (data) =>
  http.put('/api/users/me', data, {
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
const deleteUserAvatar = () => http.delete('/api/users/me/avatar')
const getCurrentUser = () => http.get('/api/users/me')
const getTags = () => http.get('/api/tags')
const upsertTags = (uuid, data) => http.post(`/api/conversations/${uuid}/tags`, data)
const updateAssignee = (uuid, assignee_type, data) =>
  http.put(`/api/conversations/${uuid}/assignee/${assignee_type}`, data)
const updateConversationStatus = (uuid, data) => http.put(`/api/conversations/${uuid}/status`, data)
const updateConversationPriority = (uuid, data) => http.put(`/api/conversations/${uuid}/priority`, data)
const updateAssigneeLastSeen = (uuid) => http.put(`/api/conversations/${uuid}/last-seen`)
const getConversationMessage = (cuuid, uuid) => http.get(`/api/conversations/${cuuid}/messages/${uuid}`)
const retryMessage = (cuuid, uuid) => http.put(`/api/conversations/${cuuid}/messages/${uuid}/retry`)
const getConversationMessages = (uuid, page) =>
  http.get(`/api/conversations/${uuid}/messages`, {
    params: { page: page }
  })
const sendMessage = (uuid, data) =>
  http.post(`/api/conversations/${uuid}/messages`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const getConversation = (uuid) => http.get(`/api/conversations/${uuid}`)
const getConversationParticipants = (uuid) => http.get(`/api/conversations/${uuid}/participants`)
const getCannedResponses = () => http.get('/api/canned-responses')
const createCannedResponse = (data) => http.post('/api/canned-responses', data)
const updateCannedResponse = (id, data) => http.put(`/api/canned-responses/${id}`, data)
const deleteCannedResponse = (id) => http.delete(`/api/canned-responses/${id}`)
const getAssignedConversations = (params) =>
  http.get('/api/conversations/assigned', { params })
const getUnassignedConversations = (params) =>
  http.get('/api/conversations/unassigned', { params })
const getAllConversations = (params) =>
  http.get('/api/conversations/all', { params })
const uploadMedia = (data) =>
  http.post('/api/media', data, {
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
const getGlobalDashboardCounts = () => http.get('/api/dashboard/global/counts')
const getGlobalDashboardCharts = () => http.get('/api/dashboard/global/charts')
const getUserDashboardCounts = () => http.get(`/api/dashboard/me/counts`)
const getUserDashboardCharts = () => http.get(`/api/dashboard/me/charts`)
const getLanguage = (lang) => http.get(`/api/lang/${lang}`)
const createUser = (data) =>
  http.post('/api/users', data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const updateUser = (id, data) =>
  http.put(`/api/users/${id}`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const updateTeam = (id, data) => http.put(`/api/teams/${id}`, data)
const createTeam = (data) => http.post('/api/teams', data)
const createInbox = (data) =>
  http.post('/api/inboxes', data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const getInboxes = () => http.get('/api/inboxes')
const getInbox = (id) => http.get(`/api/inboxes/${id}`)
const toggleInbox = (id) => http.put(`/api/inboxes/${id}/toggle`)
const updateInbox = (id, data) =>
  http.put(`/api/inboxes/${id}`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const deleteInbox = (id) => http.delete(`/api/inboxes/${id}`)

export default {
  login,
  getTags,
  getTeam,
  getUser,
  getRoles,
  getRole,
  createRole,
  deleteRole,
  updateRole,
  getTeams,
  getUsers,
  getInbox,
  getInboxes,
  getLanguage,
  getConversation,
  getAutomationRule,
  getAutomationRules,
  getAssignedConversations,
  getUnassignedConversations,
  getAllConversations,
  getGlobalDashboardCharts,
  getGlobalDashboardCounts,
  getUserDashboardCounts,
  getUserDashboardCharts,
  getConversationParticipants,
  getConversationMessage,
  getConversationMessages,
  getCurrentUser,
  getCannedResponses,
  createCannedResponse,
  updateCannedResponse,
  deleteCannedResponse,
  updateCurrentUser,
  updateAssignee,
  updateConversationStatus,
  updateConversationPriority,
  upsertTags,
  uploadMedia,
  updateAutomationRule,
  updateAssigneeLastSeen,
  updateUser,
  createAutomationRule,
  toggleAutomationRule,
  deleteAutomationRule,
  sendMessage,
  retryMessage,
  createUser,
  createInbox,
  updateInbox,
  deleteInbox,
  toggleInbox,
  createTeam,
  updateTeam,
  getSettings,
  updateSettings,
  createOIDC,
  getAllOIDC,
  getOIDC,
  updateOIDC,
  deleteOIDC,
  getTemplate,
  getTemplates,
  createTemplate,
  updateTemplate,
  deleteTemplate,
  deleteUserAvatar,
  createTag,
  updateTag,
  deleteTag,
  getStatuses,
  getPriorities,
  createStatus,
  updateStatus,
  deleteStatus,
  getTeamsCompact,
  getUsersCompact,
  getEmailNotificationSettings,
  updateEmailNotificationSettings,
}
