import http from './http'
export const listCredentials  = ()         => http.get('/api/credentials')
export const getCredential    = (id)       => http.get(`/api/credentials/${id}`)
export const createCredential = (data)     => http.post('/api/credentials', data)
export const updateCredential = (id, data) => http.put(`/api/credentials/${id}`, data)
export const deleteCredential = (id)       => http.delete(`/api/credentials/${id}`)
