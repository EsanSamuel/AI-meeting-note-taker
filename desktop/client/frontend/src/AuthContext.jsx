import { createContext, useContext, useEffect, useState } from 'react';
import { api } from './services/api';

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
    const [user, setUser] = useState(null);
    // 'loading' | 'authenticated' | 'unauthenticated'
    const [status, setStatus] = useState('loading');

    async function refreshSession() {
        try {
            const { user } = await api.auth.me();
            setUser(user);
            setStatus('authenticated');
        } catch {
            setUser(null);
            setStatus('unauthenticated');
        }
    }

    useEffect(() => { refreshSession(); }, []);

    async function login(email, password) {
        const { user } = await api.auth.login(email, password);
        setUser(user);
        setStatus('authenticated');
    }

    async function setupOrganization(payload) {
        await api.auth.setup(payload);
        // Setup only creates the org + admin user, it does not start a
        // session, so log in right after with the same credentials.
        await login(payload.email, payload.password);
    }

    async function acceptInvitation(payload) {
        // acceptInvitation doesn't return a session either — the caller is
        // expected to send the person to the login form afterwards.
        await api.auth.acceptInvitation(payload);
    }

    async function logout() {
        await api.auth.logout();
        setUser(null);
        setStatus('unauthenticated');
    }

    const value = { user, status, login, logout, setupOrganization, acceptInvitation, refreshSession };
    return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
    const ctx = useContext(AuthContext);
    if (!ctx) throw new Error('useAuth must be used within an AuthProvider');
    return ctx;
}