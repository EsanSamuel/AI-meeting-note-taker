import {useEffect, useMemo, useRef, useState} from 'react';
import './App.css';
import {api, getApiBaseUrl} from './services/api';

const navItems = [
    {id: 'overview', label: 'Overview', glyph: '01'},
    {id: 'meetings', label: 'Meetings', glyph: '02'},
    {id: 'actions', label: 'Action items', glyph: '03'},
    {id: 'settings', label: 'Settings', glyph: '04'},
];

function formatDuration(seconds = 0) {
    const minutes = Math.floor(seconds / 60);
    return `${minutes} min`;
}

function formatTime(seconds = 0) {
    const minutes = Math.floor(seconds / 60);
    const remaining = Math.floor(seconds % 60).toString().padStart(2, '0');
    return `${minutes}:${remaining}`;
}

function normalizeMeeting(meeting) {
    const transcript = (meeting.transcript || meeting.transcript_segments || []).map((segment) => ({
        ...segment,
        start: segment.start ?? segment.start_time ?? 0,
        end: segment.end ?? segment.end_time ?? 0,
        speaker: segment.speaker || segment.speaker_id || 'Unknown speaker',
    }));
    return {...meeting, transcript, decisions: meeting.decisions || [], action_items: meeting.action_items || []};
}

function App() {
    const [view, setView] = useState('overview');
    const [meetings, setMeetings] = useState([]);
    const [selectedId, setSelectedId] = useState(null);
    const [query, setQuery] = useState('');
    const [apiOnline, setApiOnline] = useState(false);
    const [uploading, setUploading] = useState(false);
    const [notice, setNotice] = useState('');
    const [apiBaseUrl, setApiBaseUrl] = useState(getApiBaseUrl);
    const [recording, setRecording] = useState(false);
    const [recordingSeconds, setRecordingSeconds] = useState(0);
    const fileInput = useRef(null);
    const recorderRef = useRef(null);
    const streamRef = useRef(null);
    const recordingChunksRef = useRef([]);
    const recordingTimerRef = useRef(null);

    async function loadMeetingsFromDatabase() {
        try {
            const items = await api.meetings.list();
            const databaseMeetings = Array.isArray(items) ? items.map(normalizeMeeting) : [];
            setApiOnline(true);
            setMeetings(databaseMeetings);
            setSelectedId((current) => current && databaseMeetings.some((meeting) => meeting.id === current) ? current : databaseMeetings[0]?.id || null);
            setNotice(databaseMeetings.length ? `Loaded ${databaseMeetings.length} meeting${databaseMeetings.length === 1 ? '' : 's'} from the database.` : 'The database returned no meetings.');
        } catch (error) {
            setApiOnline(false);
            setMeetings([]);
            setSelectedId(null);
            setNotice(error instanceof Error ? `GET /api/v1/meetings failed: ${error.message}` : 'GET /api/v1/meetings failed.');
            console.error('Failed to load meetings from GET /api/v1/meetings', error);
        }
    }

    useEffect(() => {
        let cancelled = false;
        async function loadDatabaseMeetings() {
            try {
                await api.health();
                if (cancelled) return;
                await loadMeetingsFromDatabase();
            } catch (error) {
                if (cancelled) return;
                setApiOnline(false);
                setMeetings([]);
                setSelectedId(null);
                setNotice(error instanceof Error ? `Could not load meetings: ${error.message}` : 'Could not load meetings from the backend.');
                console.error('Failed to load meetings from the backend', error);
            }
        }
        loadDatabaseMeetings();
        return () => { cancelled = true; };
    }, []);

    useEffect(() => {
        if (!apiOnline || !selectedId || String(selectedId).startsWith('demo-')) return;
        Promise.all([
            api.meetings.get(selectedId),
            api.transcript.list(selectedId),
            api.meetings.decisions.list(selectedId),
            api.meetings.actionItems.list(selectedId),
        ]).then(([meeting, transcript, decisions, actionItems]) => {
            setMeetings((current) => current.map((item) => item.id === selectedId ? normalizeMeeting({...item, ...meeting, transcript, decisions, action_items: actionItems}) : item));
        }).catch(() => setNotice('Some meeting details could not be loaded.'));
    }, [apiOnline, selectedId]);

    useEffect(() => () => {
        clearInterval(recordingTimerRef.current);
        streamRef.current?.getTracks().forEach((track) => track.stop());
    }, []);

    const selectedMeeting = meetings.find((meeting) => meeting.id === selectedId) || null;
    const filteredMeetings = useMemo(() => meetings.filter((meeting) => meeting.title?.toLowerCase().includes(query.toLowerCase())), [meetings, query]);
    const actionItems = meetings.flatMap((meeting) => (meeting.action_items || []).map((item) => ({...item, meetingTitle: meeting.title, meetingId: meeting.id})));
    const openActions = actionItems.filter((item) => !item.completed);

    function openMeeting(id) {
        setSelectedId(id);
        setView('detail');
    }

    async function toggleAction(item) {
        const completed = !item.completed;
        setMeetings((current) => current.map((meeting) => meeting.id !== item.meetingId ? meeting : {...meeting, action_items: meeting.action_items.map((action) => action.id === item.id ? {...action, completed} : action)}));
        if (apiOnline && !String(item.id).startsWith('a-')) {
            try { await (completed ? api.actionItems.complete(item.id) : api.actionItems.incomplete(item.id)); } catch { setNotice('Could not sync that update.'); }
        }
    }

    async function renameSpeakers(meetingId, speakers) {
        setMeetings((current) => current.map((meeting) => meeting.id !== meetingId ? meeting : {
            ...meeting,
            transcript: meeting.transcript.map((segment) => {
                const speakerKey = segment.speaker_id || segment.speaker;
                return speakers[speakerKey] ? {...segment, speaker: speakers[speakerKey]} : segment;
            }),
        }));
        if (apiOnline && !String(meetingId).startsWith('demo-')) {
            try {
                await api.transcript.renameSpeakers(meetingId, speakers);
                setNotice('Speaker names saved.');
            } catch {
                try {
                    const transcript = await api.transcript.list(meetingId);
                    setMeetings((current) => current.map((meeting) => meeting.id === meetingId ? {...meeting, transcript} : meeting));
                } catch {
                    // Keep the optimistic state if the recovery request also fails.
                }
                setNotice('Speaker names could not be saved.');
            }
        }
    }

    async function processRecording(file) {
        if (!file) return;
        setUploading(true);
        setNotice('Processing recording... transcription and speaker labels may take a few minutes.');
        try {
            const result = await api.recordings.upload(file);
            setApiOnline(true);
            const [meetingDetails, transcript, decisions, actionItems] = await Promise.all([
                api.meetings.get(result.meeting_id),
                api.transcript.list(result.meeting_id),
                api.meetings.decisions.list(result.meeting_id),
                api.meetings.actionItems.list(result.meeting_id),
            ]);
            setNotice('Recording processed successfully.');
            const meeting = normalizeMeeting({...meetingDetails, transcript, decisions, action_items: actionItems, title: meetingDetails.title || result.filename || file.name});
            setMeetings((current) => [meeting, ...current]);
            openMeeting(meeting.id);
        } catch (error) {
            setApiOnline(false);
            const downloadUrl = URL.createObjectURL(file);
            const link = document.createElement('a');
            link.href = downloadUrl;
            link.download = file.name;
            link.click();
            URL.revokeObjectURL(downloadUrl);
            setNotice(error instanceof Error ? `Backend upload failed, so the recording was saved locally only: ${error.message}` : 'Backend upload failed, so the recording was saved locally only.');
        }
        setUploading(false);
    }

    async function uploadRecording(event) {
        const file = event.target.files?.[0];
        await processRecording(file);
        event.target.value = '';
    }

    async function startRecording() {
        if (!navigator.mediaDevices?.getUserMedia || !window.MediaRecorder) {
            setNotice('This desktop environment does not support microphone recording.');
            return;
        }
        try {
            const stream = await navigator.mediaDevices.getUserMedia({audio: true});
            const mimeType = ['audio/webm;codecs=opus', 'audio/webm', 'audio/ogg'].find((type) => MediaRecorder.isTypeSupported(type));
            const recorder = new MediaRecorder(stream, mimeType ? {mimeType} : undefined);
            recordingChunksRef.current = [];
            streamRef.current = stream;
            recorderRef.current = recorder;
            recorder.ondataavailable = (event) => {
                if (event.data.size) recordingChunksRef.current.push(event.data);
            };
            recorder.onstop = async () => {
                const blob = new Blob(recordingChunksRef.current, {type: recorder.mimeType || 'audio/webm'});
                stream.getTracks().forEach((track) => track.stop());
                streamRef.current = null;
                recorderRef.current = null;
                clearInterval(recordingTimerRef.current);
                setRecording(false);
                setRecordingSeconds(0);
                await processRecording(new File([blob], `meeting-${new Date().toISOString().replace(/[:.]/g, '-')}.webm`, {type: blob.type}));
            };
            recorder.start(1000);
            setRecordingSeconds(0);
            setRecording(true);
            recordingTimerRef.current = setInterval(() => setRecordingSeconds((seconds) => seconds + 1), 1000);
            setNotice('Recording in progress. Speak naturally, then stop when the meeting ends.');
        } catch {
            setNotice('Microphone access was denied. Allow microphone access and try again.');
        }
    }

    function stopRecording() {
        if (recorderRef.current?.state === 'recording') recorderRef.current.stop();
    }

    function toggleRecording() {
        if (recording) stopRecording();
        else startRecording();
    }

    function saveApiBaseUrl(value) {
        const nextUrl = value.trim().replace(/\/$/, '') || getApiBaseUrl();
        localStorage.setItem('afterword.apiBaseUrl', nextUrl);
        setApiBaseUrl(nextUrl);
        setApiOnline(false);
        api.health().then(() => setApiOnline(true)).catch(() => setNotice(`Backend unavailable at ${nextUrl}.`));
    }

    return (
        <div className="app-shell">
            <aside className="sidebar">
                <div className="brand"><span className="brand-mark">A</span><span>afterword</span></div>
                <div className="workspace-label">PERSONAL WORKSPACE</div>
                <nav>{navItems.map((item) => <button className={`nav-item ${view === item.id || (view === 'detail' && item.id === 'meetings') ? 'active' : ''}`} key={item.id} onClick={() => setView(item.id)}><span className="nav-glyph">{item.glyph}</span>{item.label}</button>)}</nav>
                <div className="sidebar-bottom"><div className="status-line"><span className={`status-dot ${apiOnline ? 'online' : ''}`}></span>{apiOnline ? 'Service connected' : 'Preview mode'}</div><div className="profile"><span>ES</span><div><strong>Esan Samuel</strong><small>Local account</small></div><b>...</b></div></div>
            </aside>
            <main className="main-content">
                <header className="topbar"><div className="breadcrumbs">Workspace <span>/</span> {view === 'detail' ? selectedMeeting?.title : view === 'overview' ? 'Overview' : view === 'actions' ? 'Action items' : view === 'settings' ? 'Settings' : 'Meetings'}</div><div className="topbar-actions"><button className="icon-button" aria-label="Search">⌕</button><button className="keyboard-hint">Ctrl K</button><div className="avatar">ES</div></div></header>
                {notice && <div className="notice">{notice}<button onClick={() => setNotice('')}>Dismiss</button></div>}
                <div className="page-wrap">
                    {view === 'overview' && <Overview meetings={meetings} openActions={openActions} onOpenMeeting={openMeeting} onUpload={() => fileInput.current?.click()} onRecord={toggleRecording} recording={recording} recordingSeconds={recordingSeconds} />}
                    {view === 'meetings' && <MeetingLibrary meetings={filteredMeetings} query={query} setQuery={setQuery} onOpenMeeting={openMeeting} onUpload={() => fileInput.current?.click()} onRecord={toggleRecording} recording={recording} recordingSeconds={recordingSeconds} onRefresh={loadMeetingsFromDatabase} />}
                    {view === 'actions' && <ActionView items={actionItems} onToggle={toggleAction} onOpenMeeting={openMeeting} />}
                    {view === 'settings' && <Settings apiBaseUrl={apiBaseUrl} apiOnline={apiOnline} onSaveApiBaseUrl={saveApiBaseUrl} />}
                    {view === 'detail' && selectedMeeting && <><MeetingDetail meeting={selectedMeeting} onBack={() => setView('meetings')} onToggle={toggleAction} /><SpeakerEditor meeting={selectedMeeting} onSave={renameSpeakers} /></>}
                </div>
                <input ref={fileInput} type="file" accept="audio/*,video/*,.webm,.mp4" hidden onChange={uploadRecording} />
                {uploading && <div className="processing"><span className="spinner"></span><div><strong>Analyzing your recording</strong><small>Transcription, speakers, and takeaways</small></div></div>}
            </main>
        </div>
    )
}

function PageIntro({eyebrow, title, children}) { return <div className="page-intro"><div><div className="eyebrow">{eyebrow}</div><h1>{title}</h1></div>{children}</div>; }

function RecordingButton({onRecord, recording, recordingSeconds}) {
    return <button className={`record-button ${recording ? 'is-recording' : ''}`} onClick={onRecord} aria-label={recording ? 'Stop recording' : 'Start recording'}>
        <span className="record-dot"></span>{recording ? `Stop ${formatTime(recordingSeconds)}` : 'Start recording'}
    </button>;
}

function Overview({meetings, openActions, onOpenMeeting, onUpload, onRecord, recording, recordingSeconds}) {
    const totalMinutes = Math.round(meetings.reduce((sum, meeting) => sum + (meeting.duration_seconds || 0), 0) / 60);
    return <>
        <PageIntro eyebrow="Monday, September 21, 2026" title="Good morning, Esan."><div className="intro-actions"><RecordingButton onRecord={onRecord} recording={recording} recordingSeconds={recordingSeconds} /><button className="primary-button" onClick={onUpload}><span>+</span> Import recording</button></div></PageIntro>
        <section className="hero-panel"><div><div className="eyebrow warm">YOUR MEETING MEMORY</div><h2>Make every conversation<br /><em>move things forward.</em></h2><p>Transcripts, decisions, and next steps in one quiet place.</p></div><div className="hero-stats"><div><strong>{meetings.length}</strong><span>Meetings captured</span></div><div><strong>{totalMinutes}<small>m</small></strong><span>Conversation indexed</span></div></div></section>
        <div className="section-heading"><div><div className="eyebrow">RECENTLY CAPTURED</div><h2>Your meetings</h2></div><button className="text-button" onClick={() => document.querySelector('.nav-item:nth-child(2)')?.click()}>View all <span>→</span></button></div>
        {meetings.length ? <div className="meeting-grid">{meetings.slice(0, 3).map((meeting, index) => <MeetingCard key={meeting.id} meeting={meeting} index={index} onClick={() => onOpenMeeting(meeting.id)} />)}</div> : <div className="empty-state large">No meetings yet. Start a recording or import one to create your first meeting.</div>}
        <section className="lower-grid"><div className="panel"><div className="panel-heading"><div><div className="eyebrow">NEEDS YOUR ATTENTION</div><h3>Open action items</h3></div><span className="count-badge">{openActions.length}</span></div>{openActions.slice(0, 3).map((item) => <ActionRow key={item.id} item={item} />)}{!openActions.length && <div className="empty-state">You are all caught up.</div>}</div><div className="quote-panel"><div className="quote-mark">“</div><p>Good notes do not just remember the conversation. They make the next conversation easier.</p><span>— Afterword principle</span></div></section>
    </>;
}

function MeetingCard({meeting, index, onClick}) { return <button className={`meeting-card card-${index}`} onClick={onClick}><div className="card-top"><span className="waveform"><i></i><i></i><i></i><i></i><i></i><i></i><i></i></span><span>{formatDuration(meeting.duration_seconds)}</span></div><h3>{meeting.title}</h3><p>{meeting.summary || 'Transcript and meeting insights are ready to review.'}</p><div className="card-footer"><span>{meeting.date || new Date(meeting.started_at).toLocaleDateString()}</span><span className="arrow">↗</span></div></button>; }

function MeetingLibrary({meetings, query, setQuery, onOpenMeeting, onUpload, onRecord, recording, recordingSeconds, onRefresh}) { return <><PageIntro eyebrow="MEETING LIBRARY" title="All meetings"><div className="intro-actions"><RecordingButton onRecord={onRecord} recording={recording} recordingSeconds={recordingSeconds} /><button className="primary-button" onClick={onUpload}><span>+</span> Import recording</button></div></PageIntro><div className="library-toolbar"><div className="search-field"><span>⌕</span><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search meetings" /><kbd>⌘ F</kbd></div><div className="library-tools"><span className="result-count">{meetings.length} conversations</span><button className="refresh-button" onClick={onRefresh}>Refresh database</button></div></div><div className="library-list">{meetings.map((meeting) => <button className="library-row" key={meeting.id} onClick={() => onOpenMeeting(meeting.id)}><span className="row-date">{meeting.date || new Date(meeting.started_at).toLocaleDateString()}</span><span className="row-title"><strong>{meeting.title}</strong><small>{meeting.summary || 'No summary available yet.'}</small></span><span className="row-duration">{formatDuration(meeting.duration_seconds)}</span><span className="row-arrow">→</span></button>)}</div>{!meetings.length && <div className="empty-state large">No meetings match that search.</div>}</>; }

function ActionRow({item, onToggle}) { return <div className="action-row"><button className={`checkbox ${item.completed ? 'checked' : ''}`} onClick={() => onToggle(item)}>{item.completed ? '✓' : ''}</button><div><strong className={item.completed ? 'completed-text' : ''}>{item.task}</strong><small>{item.assignee || 'Unassigned'} · {item.meetingTitle || 'Meeting'}</small></div></div>; }

function ActionView({items, onToggle, onOpenMeeting}) { return <><PageIntro eyebrow="FOLLOW THROUGH" title="Action items"><span className="subtle-label">{items.filter((item) => !item.completed).length} open</span></PageIntro><div className="action-layout"><div className="panel actions-panel"><div className="panel-heading"><div><div className="eyebrow">YOUR NEXT STEPS</div><h3>Everything in one list</h3></div></div>{items.map((item) => <div className="action-row clickable" key={item.id} onClick={() => onOpenMeeting(item.meetingId)}><button className={`checkbox ${item.completed ? 'checked' : ''}`} onClick={(event) => { event.stopPropagation(); onToggle(item); }}>{item.completed ? '✓' : ''}</button><div><strong className={item.completed ? 'completed-text' : ''}>{item.task}</strong><small>{item.assignee || 'Unassigned'} · {item.meetingTitle}</small></div><span className="row-arrow">→</span></div>)}</div><div className="side-note"><div className="eyebrow">A SMALL PROMPT</div><h3>Close the loop.</h3><p>The best action item is the one that disappears because it got done.</p></div></div></>; }

function SpeakerEditor({meeting, onSave}) {
    const speakers = [...new Map((meeting.transcript || []).map((segment) => {
        const id = segment.speaker_id || segment.speaker;
        return [id, {id, label: segment.speaker || id}];
    })).values()];
    const [draft, setDraft] = useState({});

    useEffect(() => {
        setDraft(Object.fromEntries(speakers.map((speaker) => [speaker.id, speaker.label])));
    }, [meeting.id, meeting.transcript]);

    if (!speakers.length) return null;
    return <section className="panel speaker-panel"><div className="panel-heading"><div><div className="eyebrow">SPEAKER MANAGEMENT</div><h3>Rename speakers</h3></div><span className="subtle-label">IDs stay stable</span></div><p className="settings-copy">Names change how speakers are displayed; backend speaker identifiers remain unchanged.</p>{speakers.map((speaker) => <label className="speaker-field" key={speaker.id}><span>{speaker.id}</span><input value={draft[speaker.id] || ''} onChange={(event) => setDraft((current) => ({...current, [speaker.id]: event.target.value}))} /></label>)}<button className="primary-button" onClick={() => onSave(meeting.id, Object.fromEntries(Object.entries(draft).map(([id, name]) => [id, name.trim() || id])))}>Save speaker names</button></section>;
}

function MeetingDetail({meeting, onBack, onToggle}) { return <><button className="back-button" onClick={onBack}>← Back to meetings</button><div className="detail-heading"><div><div className="eyebrow">{meeting.date || 'CAPTURED MEETING'}</div><h1>{meeting.title}</h1><p className="detail-meta">{formatDuration(meeting.duration_seconds)} <span>·</span> {meeting.transcript?.length || 0} transcript segments <span>·</span> {meeting.action_items?.length || 0} action items</p></div><button className="secondary-button">•••</button></div><section className="summary-banner"><div className="summary-icon">✦</div><div><div className="eyebrow">AI SUMMARY</div><p>{meeting.summary || 'This meeting has not been summarized yet.'}</p></div></section><div className="detail-grid"><div className="transcript-panel"><div className="panel-heading"><div><div className="eyebrow">CONVERSATION</div><h3>Transcript</h3></div><button className="text-button">Export <span>↓</span></button></div>{meeting.transcript?.length ? meeting.transcript.map((segment, index) => <div className="transcript-line" key={`${segment.start}-${index}`}><button className="timestamp">{formatTime(segment.start)}</button><div><strong>{segment.speaker?.replace('SPEAKER_', 'Speaker ')}</strong><p>{segment.text}</p></div></div>) : <div className="empty-state">Transcript segments will appear here after processing.</div>}</div><aside className="detail-side"><div className="panel"><div className="panel-heading"><div><div className="eyebrow">FOLLOW THROUGH</div><h3>Action items</h3></div><span className="count-badge">{meeting.action_items?.length || 0}</span></div>{meeting.action_items?.map((item) => <ActionRow key={item.id} item={{...item, meetingId: meeting.id, meetingTitle: meeting.title}} onToggle={onToggle} />)}{!meeting.action_items?.length && <div className="empty-state">No action items found.</div>}</div><div className="panel decisions-panel"><div className="panel-heading"><div><div className="eyebrow">AGREEMENTS</div><h3>Decisions</h3></div></div>{meeting.decisions?.map((decision) => <div className="decision" key={decision.id || decision.timestamp_seconds}><span>✓</span><p>{decision.decision || decision.text}</p></div>)}{!meeting.decisions?.length && <div className="empty-state">No decisions found.</div>}</div></aside></div></>; }

function Settings({apiBaseUrl, apiOnline, onSaveApiBaseUrl}) {
    const [draftUrl, setDraftUrl] = useState(apiBaseUrl);
    return <>
        <PageIntro eyebrow="APPLICATION SETTINGS" title="Settings" />
        <div className="settings-grid">
            <section className="panel settings-panel"><div className="eyebrow">BACKEND</div><h3>Connection</h3><p className="settings-copy">The desktop app sends recordings and meeting queries to your configured local service.</p><label className="settings-label" htmlFor="api-url">API base URL</label><input id="api-url" className="settings-input" value={draftUrl} onChange={(event) => setDraftUrl(event.target.value)} /><div className="connection-state"><span className={`status-dot ${apiOnline ? 'online' : ''}`}></span>{apiOnline ? 'Connected' : 'Unavailable'}<span className="connection-url">{apiBaseUrl}</span></div><button className="primary-button" onClick={() => onSaveApiBaseUrl(draftUrl)}>Save and test connection</button></section>
            <section className="panel settings-panel"><div className="eyebrow">PRIVACY</div><h3>Local-first processing</h3><p className="settings-copy">Audio is sent only to the backend URL above. This client does not add cloud AI services or store meeting data in browser storage.</p><div className="capability-list"><div><span className="status-dot online"></span><strong>Microphone recording</strong><small>Captured in the desktop webview</small></div><div><span className="status-dot online"></span><strong>Transcription and diarization</strong><small>Provided by the Go backend</small></div><div><span className="status-dot"></span><strong>Semantic search and Q&amp;A</strong><small>Backend routes are not currently exposed</small></div></div></section>
        </div>
    </>;
}

export default App
