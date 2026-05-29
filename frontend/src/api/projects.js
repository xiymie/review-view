import http from './http'

export const listProjects          = ()         => http.get('/api/projects')
export const getProject            = (id)       => http.get(`/api/projects/${id}`)
export const createProject         = (data)     => http.post('/api/projects', data)
export const updateProject         = (id, data) => http.put(`/api/projects/${id}`, data)
export const updateProjectSchedule = (id, data) => http.put(`/api/projects/${id}/schedule`, data)
export const deleteProject         = (id)       => http.delete(`/api/projects/${id}`)
export const triggerProject        = (id, data) => http.post(`/api/projects/${id}/trigger`, data)
export const initProject           = (id)       => http.post(`/api/projects/${id}/initialize`)
export const getCommits            = (id)       => http.get(`/api/projects/${id}/commits?limit=50`)
export const listModels            = ()         => http.get('/api/models')
export const listCredentials       = ()         => http.get('/api/credentials')
