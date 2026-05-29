import http from './http'

export const listSensitiveWords = () => http.get('/api/sensitive-words')
export const createSensitiveWord = (data) => http.post('/api/sensitive-words', data)
export const updateSensitiveWord = (id, data) => http.put(`/api/sensitive-words/${id}`, data)
export const deleteSensitiveWord = (id) => http.delete(`/api/sensitive-words/${id}`)
