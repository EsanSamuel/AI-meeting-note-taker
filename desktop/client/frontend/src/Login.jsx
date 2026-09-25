import { useState } from 'react';
import { useAuth } from './AuthContext';

export default function Login() {
    const { login, setupOrganization, acceptInvitation } = useAuth();
    const params = new URLSearchParams(window.location.search);
    const inviteToken = params.get('invite');
    const [mode, setMode] = useState(inviteToken ? 'accept' : 'login');
    const [form, setForm] = useState({ organizationName: '', domain: '', name: '', email: '', password: '' });
    const [error, setError] = useState('');
    const [notice, setNotice] = useState('');
    const [busy, setBusy] = useState(false);

    function update(field, value) {
        setForm((current) => ({ ...current, [field]: value }));
    }

    async function handleLogin(event) {
        event.preventDefault();
        setBusy(true); setError('');
        try {
            await login(form.email, form.password);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Login failed.');
        }
        setBusy(false);
    }

    async function handleSetup(event) {
        event.preventDefault();
        setBusy(true); setError('');
        try {
            await setupOrganization({
                organization_name: form.organizationName,
                domain: form.domain,
                name: form.name,
                email: form.email,
                password: form.password,
            });
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Could not create workspace.');
        }
        setBusy(false);
    }

    async function handleAccept(event) {
        event.preventDefault();
        setBusy(true); setError('');
        try {
            await acceptInvitation({ token: inviteToken, name: form.name, password: form.password });
            setNotice('Invitation accepted. Log in with your new password.');
            setMode('login');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Could not accept invitation.');
        }
        setBusy(false);
    }

    return (
        <div className="auth-shell">
            <div className="auth-card">
                <div className="brand"><span className="brand-mark">A</span><span>afterword</span></div>
                {mode !== 'accept' && (
                    <div className="auth-tabs">
                        <button type="button" className={mode === 'login' ? 'active' : ''} onClick={() => setMode('login')}>Log in</button>
                        <button type="button" className={mode === 'setup' ? 'active' : ''} onClick={() => setMode('setup')}>Create workspace</button>
                    </div>
                )}
                {notice && <div className="auth-notice">{notice}</div>}
                {error && <div className="auth-error">{error}</div>}

                {mode === 'login' && (
                    <form className="auth-form" onSubmit={handleLogin}>
                        <input className="settings-input" type="email" placeholder="Email" value={form.email} onChange={(e) => update('email', e.target.value)} required />
                        <input className="settings-input" type="password" placeholder="Password" value={form.password} onChange={(e) => update('password', e.target.value)} required />
                        <button className="primary-button" type="submit" disabled={busy}>{busy ? 'Signing in…' : 'Log in'}</button>
                    </form>
                )}

                {mode === 'setup' && (
                    <form className="auth-form" onSubmit={handleSetup}>
                        <input className="settings-input" placeholder="Organization name" value={form.organizationName} onChange={(e) => update('organizationName', e.target.value)} required />
                        <input className="settings-input" placeholder="Domain (optional)" value={form.domain} onChange={(e) => update('domain', e.target.value)} />
                        <input className="settings-input" placeholder="Your name" value={form.name} onChange={(e) => update('name', e.target.value)} required />
                        <input className="settings-input" type="email" placeholder="Email" value={form.email} onChange={(e) => update('email', e.target.value)} required />
                        <input className="settings-input" type="password" placeholder="Password" value={form.password} onChange={(e) => update('password', e.target.value)} required />
                        <button className="primary-button" type="submit" disabled={busy}>{busy ? 'Creating…' : 'Create workspace'}</button>
                    </form>
                )}

                {mode === 'accept' && (
                    <form className="auth-form" onSubmit={handleAccept}>
                        <p className="settings-copy">You were invited to join a workspace. Set your name and a password to accept.</p>
                        <input className="settings-input" placeholder="Your name" value={form.name} onChange={(e) => update('name', e.target.value)} required />
                        <input className="settings-input" type="password" placeholder="Password" value={form.password} onChange={(e) => update('password', e.target.value)} required />
                        <button className="primary-button" type="submit" disabled={busy}>{busy ? 'Joining…' : 'Accept invitation'}</button>
                    </form>
                )}
            </div>
        </div>
    );
}