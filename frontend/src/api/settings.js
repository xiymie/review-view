import http from './http'
export const getSettings    = ()     => http.get('/api/settings')
export const updateSettings = (data) => http.put('/api/settings', data)
export const testEmail      = (data) => http.post('/api/settings/test-email', data)
