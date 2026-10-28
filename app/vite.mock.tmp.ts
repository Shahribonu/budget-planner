import base from './vite.config'
export default { ...base, server: { proxy: { '/api': 'http://localhost:8091' } } }
