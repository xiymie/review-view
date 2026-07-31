import http from './http'

export const listReviewSkills = () => http.get('/api/review-skills')
export const getReviewSkill = (id) => http.get(`/api/review-skills/${id}`)
export const createReviewSkill = (data) => http.post('/api/review-skills', data)
export const updateReviewSkill = (id, data) => http.put(`/api/review-skills/${id}`, data)
export const toggleReviewSkill = (id, enabled) => http.patch(`/api/review-skills/${id}/toggle`, { enabled })
export const deleteReviewSkill = (id) => http.delete(`/api/review-skills/${id}`)
