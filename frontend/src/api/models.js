import http from './http'
export const listModels   = ()         => http.get('/api/models')
export const getModel     = (id)       => http.get(`/api/models/${id}`)
export const createModel  = (data)     => http.post('/api/models', data)
export const updateModel  = (id, data) => http.put(`/api/models/${id}`, data)
export const deleteModel  = (id)       => http.delete(`/api/models/${id}`)
export const testModel    = (data)     => http.post('/api/models/test', data)
