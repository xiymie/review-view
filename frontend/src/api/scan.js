import http from './http'

export const listScanSchedules   = ()         => http.get('/api/scan-schedules')
export const getScanSchedule     = (id)       => http.get(`/api/scan-schedules/${id}`)
export const createScanSchedule  = (data)     => http.post('/api/scan-schedules', data)
export const updateScanSchedule  = (id, data) => http.put(`/api/scan-schedules/${id}`, data)
export const deleteScanSchedule  = (id)       => http.delete(`/api/scan-schedules/${id}`)
export const triggerScanSchedule = (id)       => http.post(`/api/scan-schedules/${id}/trigger`)
export const listScanJobs        = (id)       => http.get(`/api/scan-schedules/${id}/jobs`)
export const getScanJob          = (id)       => http.get(`/api/scan-jobs/${id}`)
export const testScanNas         = (data)     => http.post('/api/scan/test-nas', data)
export const listScanModels      = ()         => http.get('/api/scan/models')
export const listScanCredentials = ()         => http.get('/api/scan/credentials')
