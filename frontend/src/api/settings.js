import http from './http'
export const getSettings    = ()     => http.get('/api/settings')
export const updateSettings = (data) => http.put('/api/settings', data)
