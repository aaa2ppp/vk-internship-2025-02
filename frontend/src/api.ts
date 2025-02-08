import axios from 'axios';

const api = axios.create({
    baseURL: '/api',
});

export const getPingResults = () => api.get('/ping-results');