const DEFAULT_API_URL = 'http://localhost:8080';

export function getApiBaseUrl() {
    return localStorage.getItem('afterword.apiBaseUrl') || DEFAULT_API_URL;
}

export function getMeetingAudioUrl(meetingId) {
    return `${getApiBaseUrl().replace(/\/$/, '')}/api/v1/meetings/${meetingId}/audio`;
}

export async function request(path, options = {}) {
    const baseUrl = getApiBaseUrl().replace(/\/$/, '');
    const response = await fetch(`${baseUrl}${path}`, options);
    if (!response.ok) {
        let message = `Request failed: ${response.status}`;
        try {
            const body = await response.json();
            message = body.error || message;
        } catch {
            // Keep the status message when the backend does not return JSON.
        }
        throw new Error(message);
    }
    return response.status === 204 ? null : response.json();
}

function jsonRequest(method, body) {
    return { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) };
}

export const api = {
    health: () => request('/health'),
    recordings: {
        upload: (file) => {
            const form = new FormData();
            form.append('recording', file);
            return request('/api/v1/recordings', { method: 'POST', body: form });
        },
        generateAIResults: (meetingId) =>
            request(`/api/v1/recordings/${meetingId}/generate-ai-results`, { method: 'POST' }),
    },
    meetings: {
        list: () => request('/api/v1/meetings'),
        get: (id) => request(`/api/v1/meetings/${id}`),
        create: (meeting) => request('/api/v1/meetings', jsonRequest('POST', meeting)),
        update: (id, meeting) => request(`/api/v1/meetings/${id}`, jsonRequest('PUT', meeting)),
        remove: (id) => request(`/api/v1/meetings/${id}`, { method: 'DELETE' }),
        decisions: {
            list: (meetingId) => request(`/api/v1/meetings/${meetingId}/decisions`),
            create: (meetingId, decision) => request(`/api/v1/meetings/${meetingId}/decisions`, jsonRequest('POST', decision)),
            removeAll: (meetingId) => request(`/api/v1/meetings/${meetingId}/decisions`, { method: 'DELETE' }),
        },
        actionItems: {
            list: (meetingId) => request(`/api/v1/meetings/${meetingId}/action-items`),
            listIncomplete: (meetingId) => request(`/api/v1/meetings/${meetingId}/action-items/incomplete`),
            create: (meetingId, item) => request(`/api/v1/meetings/${meetingId}/action-items`, jsonRequest('POST', item)),
            removeAll: (meetingId) => request(`/api/v1/meetings/${meetingId}/action-items`, { method: 'DELETE' }),
        },
    },
    decisions: {
        get: (id) => request(`/api/v1/decisions/${id}`),
        remove: (id) => request(`/api/v1/decisions/${id}`, { method: 'DELETE' }),
    },
    actionItems: {
        get: (id) => request(`/api/v1/action-items/${id}`),
        update: (id, item) => request(`/api/v1/action-items/${id}`, jsonRequest('PUT', item)),
        complete: (id) => request(`/api/v1/action-items/${id}/complete`, { method: 'PATCH' }),
        incomplete: (id) => request(`/api/v1/action-items/${id}/incomplete`, { method: 'PATCH' }),
        remove: (id) => request(`/api/v1/action-items/${id}`, { method: 'DELETE' }),
    },
    transcript: {
        list: (meetingId) => request(`/api/v1/meetings/${meetingId}/transcript-segments`),
        create: (meetingId, segment) => request(`/api/v1/meetings/${meetingId}/transcript-segments`, jsonRequest('POST', segment)),
        removeAll: (meetingId) => request(`/api/v1/meetings/${meetingId}/transcript-segments`, { method: 'DELETE' }),
        get: (id) => request(`/api/v1/transcript-segments/${id}`),
        update: (id, segment) => request(`/api/v1/transcript-segments/${id}`, jsonRequest('PUT', segment)),
        remove: (id) => request(`/api/v1/transcript-segments/${id}`, { method: 'DELETE' }),
        renameSpeakers: (meetingId, speakers) => request(`/api/v1/meetings/${meetingId}/transcript-segments/update-speakers`, jsonRequest('PUT', { speakers })),
    },
};