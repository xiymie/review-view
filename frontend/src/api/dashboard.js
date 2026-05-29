import http from './http'

export const getDashboard = () => http.get('/api/dashboard')
