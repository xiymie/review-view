import http from './http'
export const listTasks  = ()   => http.get('/api/tasks')
export const getTask    = (id) => http.get(`/api/tasks/${id}`)
export const cancelTask = (id) => http.post(`/api/tasks/${id}/cancel`)
export const retryTask  = (id) => http.post(`/api/tasks/${id}/retry`)
export const deleteTask = (id) => http.delete(`/api/tasks/${id}`)
