import axios from 'axios'
import qs from 'qs'

const http = axios.create({
  timeout: 10000,
  responseType: 'json'
})

function getCSRFToken () {
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

const resetPassword = (data) => http.post('/api/v1/users/reset-password', data)
const setPassword = (data) => http.post('/api/v1/users/set-password', data)
const deleteUser = (id) => http.delete(`/api/v1/users/${id}`)
const getEmailNotificationSettings = () => http.get('/api/v1/settings/notifications/email')
const updateEmailNotificationSettings = (data) => http.put('/api/v1/settings/notifications/email', data)
const getPriorities = () => http.get('/api/v1/priorities')
const getStatuses = () => http.get('/api/v1/statuses')
const createStatus = (data) => http.post('/api/v1/statuses', data)
const updateStatus = (id, data) => http.put(`/api/v1/statuses/${id}`, data)
const deleteStatus = (id) => http.delete(`/api/v1/statuses/${id}`)
const createTag = (data) => http.post('/api/v1/tags', data)
const updateTag = (id, data) => http.put(`/api/v1/tags/${id}`, data)
const deleteTag = (id) => http.delete(`/api/v1/tags/${id}`)
const getTemplate = (id) => http.get(`/api/v1/templates/${id}`)
const getTemplates = (type) => http.get('/api/v1/templates', { params: { type: type } })
const createTemplate = (data) =>
  http.post('/api/v1/templates', data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const deleteTemplate = (id) => http.delete(`/api/v1/templates/${id}`)
const updateTemplate = (id, data) =>
  http.put(`/api/v1/templates/${id}`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })

const getAllBusinessHours = () => http.get('/api/v1/business-hours')
const getBusinessHours = (id) => http.get(`/api/v1/business-hours/${id}`)
const createBusinessHours = (data) => http.post('/api/v1/business-hours', data, {
  headers: {
    'Content-Type': 'application/json'
  }
})
const updateBusinessHours = (id, data) =>
  http.put(`/api/v1/business-hours/${id}`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const deleteBusinessHours = (id) => http.delete(`/api/v1/business-hours/${id}`)

const getAllSLAs = () => http.get('/api/v1/sla')
const getSLA = (id) => http.get(`/api/v1/sla/${id}`)
const createSLA = (data) => http.post('/api/v1/sla', data)
const updateSLA = (id, data) => http.put(`/api/v1/sla/${id}`, data)
const deleteSLA = (id) => http.delete(`/api/v1/sla/${id}`)

const createOIDC = (data) =>
  http.post('/api/v1/oidc', data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const getAllOIDC = () => http.get('/api/v1/oidc')
const getOIDC = (id) => http.get(`/api/v1/oidc/${id}`)
const updateOIDC = (id, data) =>
  http.put(`/api/v1/oidc/${id}`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const deleteOIDC = (id) => http.delete(`/api/v1/oidc/${id}`)
const updateSettings = (key, data) =>
  http.put(`/api/v1/settings/${key}`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const getSettings = (key) => http.get(`/api/v1/settings/${key}`)
const login = (data) => http.post(`/api/v1/login`, data)
const getAutomationRules = (type) =>
  http.get(`/api/v1/automation/rules`, {
    params: { type: type }
  })
const toggleAutomationRule = (id) => http.put(`/api/v1/automation/rules/${id}/toggle`)
const getAutomationRule = (id) => http.get(`/api/v1/automation/rules/${id}`)
const updateAutomationRule = (id, data) =>
  http.put(`/api/v1/automation/rules/${id}`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const createAutomationRule = (data) =>
  http.post(`/api/v1/automation/rules`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const getRoles = () => http.get('/api/v1/roles')
const getRole = (id) => http.get(`/api/v1/roles/${id}`)
const createRole = (data) =>
  http.post('/api/v1/roles', data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const updateRole = (id, data) =>
  http.put(`/api/v1/roles/${id}`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const deleteRole = (id) => http.delete(`/api/v1/roles/${id}`)
const deleteAutomationRule = (id) => http.delete(`/api/v1/automation/rules/${id}`)
const getUser = (id) => http.get(`/api/v1/users/${id}`)

const getTeam = (id) => http.get(`/api/v1/teams/${id}`)
const getTeams = () => http.get('/api/v1/teams')
const updateTeam = (id, data) => http.put(`/api/v1/teams/${id}`, data)
const createTeam = (data) => http.post('/api/v1/teams', data)
const getTeamsCompact = () => http.get('/api/v1/teams/compact')
const deleteTeam = (id) => http.delete(`/api/v1/teams/${id}`)

const getUsers = () => http.get('/api/v1/users')
const getUsersCompact = () => http.get('/api/v1/users/compact')
const updateCurrentUser = (data) =>
  http.put('/api/v1/users/me', data, {
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
const deleteUserAvatar = () => http.delete('/api/v1/users/me/avatar')
const getCurrentUser = () => http.get('/api/v1/users/me')
const getCurrentUserTeams = () => http.get('/api/v1/users/me/teams')
const getTags = () => http.get('/api/v1/tags')
const upsertTags = (uuid, data) => http.post(`/api/v1/conversations/${uuid}/tags`, data)
const updateAssignee = (uuid, assignee_type, data) =>
  http.put(`/api/v1/conversations/${uuid}/assignee/${assignee_type}`, data)
const updateConversationStatus = (uuid, data) => http.put(`/api/v1/conversations/${uuid}/status`, data)
const updateConversationPriority = (uuid, data) => http.put(`/api/v1/conversations/${uuid}/priority`, data)
const updateAssigneeLastSeen = (uuid) => http.put(`/api/v1/conversations/${uuid}/last-seen`)
const getConversationMessage = (cuuid, uuid) => http.get(`/api/v1/conversations/${cuuid}/messages/${uuid}`)
const retryMessage = (cuuid, uuid) => http.put(`/api/v1/conversations/${cuuid}/messages/${uuid}/retry`)
const getConversationMessages = (uuid, params) => http.get(`/api/v1/conversations/${uuid}/messages`, { params })
const sendMessage = (uuid, data) =>
  http.post(`/api/v1/conversations/${uuid}/messages`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const getConversation = (uuid) => http.get(`/api/v1/conversations/${uuid}`)
const getConversationParticipants = (uuid) => http.get(`/api/v1/conversations/${uuid}/participants`)
const getCannedResponses = () => http.get('/api/v1/canned-responses')
const createCannedResponse = (data) => http.post('/api/v1/canned-responses', data)
const updateCannedResponse = (id, data) => http.put(`/api/v1/canned-responses/${id}`, data)
const deleteCannedResponse = (id) => http.delete(`/api/v1/canned-responses/${id}`)
const getTeamUnassignedConversations = (teamID, params) =>
  http.get(`/api/v1/teams/${teamID}/conversations/unassigned`, { params })
const getAssignedConversations = (params) => http.get('/api/v1/conversations/assigned', { params })
const getUnassignedConversations = (params) => http.get('/api/v1/conversations/unassigned', { params })
const getAllConversations = (params) => http.get('/api/v1/conversations/all', { params })
const getViewConversations = (id, params) => http.get(`/api/v1/views/${id}/conversations`, { params })
const uploadMedia = (data) =>
  http.post('/api/v1/media', data, {
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
const getOverviewCounts = () => http.get('/api/v1/reports/overview/counts')
const getOverviewCharts = () => http.get('/api/v1/reports/overview/charts')
const getLanguage = (lang) => http.get(`/api/v1/lang/${lang}`)
const createUser = (data) =>
  http.post('/api/v1/users', data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const updateUser = (id, data) =>
  http.put(`/api/v1/users/${id}`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const createInbox = (data) =>
  http.post('/api/v1/inboxes', data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const getInboxes = () => http.get('/api/v1/inboxes')
const getInbox = (id) => http.get(`/api/v1/inboxes/${id}`)
const toggleInbox = (id) => http.put(`/api/v1/inboxes/${id}/toggle`)
const updateInbox = (id, data) =>
  http.put(`/api/v1/inboxes/${id}`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const deleteInbox = (id) => http.delete(`/api/v1/inboxes/${id}`)
const getCurrentUserViews = () => http.get('/api/v1/views/me')
const createView = (data) =>
  http.post('/api/v1/views/me', data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const updateView = (id, data) =>
  http.put(`/api/v1/views/me/${id}`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const deleteView = (id) => http.delete(`/api/v1/views/me/${id}`)
const getAiPrompts = () => http.get('/api/v1/ai/prompts')
const aiCompletion = (data) => http.post('/api/v1/ai/completion', data)

export default {
  login,
  deleteUser,
  resetPassword,
  setPassword,
  getTags,
  getTeam,
  getUser,
  getRoles,
  getRole,
  createRole,
  deleteRole,
  updateRole,
  getTeams,
  deleteTeam,
  getUsers,
  getInbox,
  getInboxes,
  getLanguage,
  getConversation,
  getAutomationRule,
  getAutomationRules,
  getAllBusinessHours,
  getBusinessHours,
  createBusinessHours,
  updateBusinessHours,
  deleteBusinessHours,
  getAllSLAs,
  getSLA,
  createSLA,
  updateSLA,
  deleteSLA,
  getAssignedConversations,
  getUnassignedConversations,
  getAllConversations,
  getTeamUnassignedConversations,
  getViewConversations,
  getOverviewCharts,
  getOverviewCounts,
  getConversationParticipants,
  getConversationMessage,
  getConversationMessages,
  getCurrentUser,
  getCurrentUserTeams,
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
  getCurrentUserViews,
  createView,
  updateView,
  deleteView,
  getAiPrompts,
  aiCompletion
}
