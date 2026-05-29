import http from './http'

export const listUsers = () => http.get('/api/users')
export const getUser = (id) => http.get(`/api/users/${id}`)
export const createUser = (data) => http.post('/api/users', data)
export const updateUser = (id, data) => http.put(`/api/users/${id}`, data)
export const deleteUser = (id) => http.delete(`/api/users/${id}`)
export const getMe = () => http.get('/api/users/me')
export const updateMe = (data) => http.put('/api/users/me', data)
