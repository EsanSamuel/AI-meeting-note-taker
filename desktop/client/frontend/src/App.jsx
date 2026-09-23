import { useEffect, useMemo, useRef, useState } from 'react';
import './App.css';
import { api, getApiBaseUrl, getMeetingAudioUrl } from './services/api';

const navItems = [
    { id: 'overview', label: 'Overview', glyph: '⌂' },
    { id: 'meetings', label: 'Meetings', glyph: '▤' },
    { id: 'actions', label: 'Action items', glyph: '✓' },
    { id: 'settings', label: 'Settings', glyph: '⚙' },
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
    return { ...meeting, transcript, decisions: meeting.decisions || [], action_items: meeting.action_items || [] };
}

// Flags common virtual-loopback devices so the UI can point the user toward them
// and so startRecording can decide whether to disable echoCancellation.
function isLikelyVirtualDevice(label = '') {
    return /cable|blackhole|loopback|virtual/i.test(label);
}

// --- AI summary formatting helpers -----------------------------------------

function renderInlineMarkdown(text) {
    const parts = text.split(/(\*\*[^*]+\*\*)/g).filter(Boolean);
    return parts.map((part, index) =>
        part.startsWith('**') && part.endsWith('**')
            ? <strong key={index}>{part.slice(2, -2)}</strong>
            : <span key={index}>{part}</span>
    );
}

function renderSummaryBlocks(body) {
    const lines = body.split('\n').map((line) => line.trim()).filter(Boolean);
    const blocks = [];
    let currentList = null;
    lines.forEach((line) => {
        if (line.startsWith('*')) {
            const content = line.replace(/^\*+\s*/, '');
            if (!currentList) { currentList = []; blocks.push({ type: 'list', items: currentList }); }
            currentList.push(content);
        } else {
            currentList = null;
            blocks.push({ type: 'para', text: line.replace(/^#+\s*/, '') });
        }
    });
    return blocks.map((block, index) => block.type === 'list'
        ? <ul className="summary-list" key={index}>{block.items.map((item, itemIndex) => <li key={itemIndex}>{renderInlineMarkdown(item)}</li>)}</ul>
        : <p key={index}>{renderInlineMarkdown(block.text)}</p>
    );
}

function parseSummarySections(summary) {
    if (!summary) return [];
    const chunks = summary.replace(/\r\n/g, '\n').split(/\n-{3,}\n/).map((chunk) => chunk.trim()).filter(Boolean);
    return chunks.map((chunk, index) => {
        const headerMatch = chunk.match(/^#{1,6}\s*\d*\.?\s*(.+?)\n([\s\S]*)$/);
        return headerMatch
            ? { id: index, title: headerMatch[1].trim(), body: headerMatch[2].trim() }
            : { id: index, title: null, body: chunk };
    });
}

function SummaryContent({ summary }) {
    const [expanded, setExpanded] = useState(false);
    const sections = parseSummarySections(summary);
    if (!sections.length) return null;
    return <>
        <div className={`summary-content ${expanded ? '' : 'is-collapsed'}`}>
            {sections.map((section) => (
                <div className="summary-section" key={section.id}>
                    {section.title && <h4>{section.title}</h4>}
                    {renderSummaryBlocks(section.body)}
                </div>
            ))}
        </div>
        <button className={`summary-toggle ${expanded ? 'is-expanded' : ''}`} onClick={() => setExpanded((current) => !current)}>
            {expanded ? 'Show less' : 'Show full summary'}
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                <path d="M6 9l6 6 6-6" />
            </svg>
        </button>
    </>;
}

// -----------------------------------------------------------------------------

function App() {
    const [view, setView] = useState('overview');
    const [meetings, setMeetings] = useState([]);
    const [selectedId, setSelectedId] = useState(null);
    const [query, setQuery] = useState('');
    const [apiOnline, setApiOnline] = useState(false);
    const [uploading, setUploading] = useState(false);
    const [notice, setNotice] = useState('');
    const [apiBaseUrl, setApiBaseUrl] = useState(getApiBaseUrl);
    const [workspaceName, setWorkspaceName] = useState(() => localStorage.getItem('afterword.workspaceName') || '');
    const [recording, setRecording] = useState(false);
    const [recordingSeconds, setRecordingSeconds] = useState(0);
    const [editingMeetingTitle, setEditingMeetingTitle] = useState(false);
    const [meetingTitleDraft, setMeetingTitleDraft] = useState('');
    const [audioDevices, setAudioDevices] = useState([]);
    const [microphoneDeviceId, setMicrophoneDeviceId] = useState(
        () => localStorage.getItem('afterword.microphoneDeviceId') || ''
    );
    const [generatingSummary, setGeneratingSummary] = useState(false);
    const fileInput = useRef(null);
    const recorderRef = useRef(null);
    const streamRef = useRef(null);
    const sourceStreamsRef = useRef([]);
    const audioContextRef = useRef(null);
    const recordingChunksRef = useRef([]);
    const recordingTimerRef = useRef(null);
    const audioMonitorTimerRef = useRef(null);
    const audioSignalDetectedRef = useRef(false);

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
            setMeetings((current) => current.map((item) => item.id === selectedId ? normalizeMeeting({ ...item, ...meeting, transcript, decisions, action_items: actionItems }) : item));
        }).catch(() => setNotice('Some meeting details could not be loaded.'));
    }, [apiOnline, selectedId]);

    // Enumerate audio input devices so the user can pick a virtual loopback
    // device (VB-Cable / BlackHole) instead of the physical mic, avoiding
    // the exclusive-access conflict with Meet.
    useEffect(() => {
        async function loadAudioDevices() {
            try {
                // Device labels are only populated after a permission grant, so
                // request a throwaway stream first if we don't have labels yet.
                const devices = await navigator.mediaDevices.enumerateDevices();
                const inputs = devices.filter((device) => device.kind === 'audioinput');
                if (inputs.length && !inputs[0].label) {
                    const temp = await navigator.mediaDevices.getUserMedia({ audio: true });
                    temp.getTracks().forEach((track) => track.stop());
                    const relabeled = await navigator.mediaDevices.enumerateDevices();
                    setAudioDevices(relabeled.filter((device) => device.kind === 'audioinput'));
                } else {
                    setAudioDevices(inputs);
                }
            } catch {
                // Permission not granted yet; leave the list empty until the user tries recording.
            }
        }
        loadAudioDevices();
        navigator.mediaDevices.addEventListener('devicechange', loadAudioDevices);
        return () => navigator.mediaDevices.removeEventListener('devicechange', loadAudioDevices);
    }, []);

    useEffect(() => () => {
        clearInterval(recordingTimerRef.current);
        clearInterval(audioMonitorTimerRef.current);
        streamRef.current?.getTracks().forEach((track) => track.stop());
        sourceStreamsRef.current.forEach((stream) => stream.getTracks().forEach((track) => track.stop()));
        audioContextRef.current?.close();
    }, []);

    const selectedMeeting = meetings.find((meeting) => meeting.id === selectedId) || null;
    const filteredMeetings = useMemo(() => meetings.filter((meeting) => meeting.title?.toLowerCase().includes(query.toLowerCase())), [meetings, query]);
    const actionItems = meetings.flatMap((meeting) => (meeting.action_items || []).map((item) => ({ ...item, meetingTitle: meeting.title, meetingId: meeting.id })));
    const openActions = actionItems.filter((item) => !item.completed);

    function openMeeting(id) {
        setSelectedId(id);
        setView('detail');
    }

    function beginMeetingTitleEdit(meeting) {
        setMeetingTitleDraft(meeting.title || '');
        setEditingMeetingTitle(true);
    }

    async function saveMeetingTitle(meeting) {
        const title = meetingTitleDraft.trim();
        if (!title) {
            setNotice('Meeting name cannot be empty.');
            return;
        }
        const previousTitle = meeting.title;
        setMeetings((current) => current.map((item) => item.id === meeting.id ? { ...item, title } : item));
        setEditingMeetingTitle(false);
        try {
            await api.meetings.update(meeting.id, {
                title,
                started_at: meeting.started_at,
                ended_at: meeting.ended_at,
                duration_seconds: meeting.duration_seconds || 0,
                audio_path: meeting.audio_path || '',
                video_path: meeting.video_path || '',
            });
            setNotice('Meeting name saved.');
        } catch (error) {
            setMeetings((current) => current.map((item) => item.id === meeting.id ? { ...item, title: previousTitle } : item));
            setNotice(error instanceof Error ? `Could not save meeting name: ${error.message}` : 'Could not save meeting name.');
        }
    }

    async function deleteMeeting(meeting) {
        if (!window.confirm(`Delete "${meeting.title || 'this meeting'}"? This also removes its transcript, decisions, and action items.`)) return;
        try {
            await api.meetings.remove(meeting.id);
            setMeetings((current) => current.filter((item) => item.id !== meeting.id));
            setSelectedId(null);
            setEditingMeetingTitle(false);
            setView('meetings');
            setNotice('Meeting deleted.');
        } catch (error) {
            setNotice(error instanceof Error ? `Could not delete meeting: ${error.message}` : 'Could not delete meeting.');
        }
    }

    async function toggleAction(item) {
        const completed = !item.completed;
        setMeetings((current) => current.map((meeting) => meeting.id !== item.meetingId ? meeting : { ...meeting, action_items: meeting.action_items.map((action) => action.id === item.id ? { ...action, completed } : action) }));
        if (apiOnline && !String(item.id).startsWith('a-')) {
            try { await (completed ? api.actionItems.complete(item.id) : api.actionItems.incomplete(item.id)); } catch { setNotice('Could not sync that update.'); }
        }
    }

    async function renameSpeakers(meetingId, speakers) {
        setMeetings((current) => current.map((meeting) => meeting.id !== meetingId ? meeting : {
            ...meeting,
            transcript: meeting.transcript.map((segment) => {
                const speakerKey = segment.speaker_id || segment.speaker;
                return speakers[speakerKey] ? { ...segment, speaker: speakers[speakerKey] } : segment;
            }),
        }));
        if (apiOnline && !String(meetingId).startsWith('demo-')) {
            try {
                await api.transcript.renameSpeakers(meetingId, speakers);
                setNotice('Speaker names saved.');
            } catch {
                try {
                    const transcript = await api.transcript.list(meetingId);
                    setMeetings((current) => current.map((meeting) => meeting.id === meetingId ? { ...meeting, transcript } : meeting));
                } catch {
                    // Keep the optimistic state if the recovery request also fails.
                }
                setNotice('Speaker names could not be saved.');
            }
        }
    }

    async function regenerateSummary(meeting) {
        if (!meeting) return;
        setGeneratingSummary(true);
        setNotice('Regenerating AI summary...');
        try {
            await api.recordings.generateAIResults(meeting.id);
            const [meetingDetails, decisions, actionItems] = await Promise.all([
                api.meetings.get(meeting.id),
                api.meetings.decisions.list(meeting.id),
                api.meetings.actionItems.list(meeting.id),
            ]);
            setMeetings((current) => current.map((item) => item.id === meeting.id
                ? normalizeMeeting({ ...item, ...meetingDetails, decisions, action_items: actionItems })
                : item));
            setNotice('AI summary generated.');
        } catch (error) {
            setNotice(error instanceof Error ? `Could not generate AI summary: ${error.message}` : 'Could not generate AI summary.');
        }
        setGeneratingSummary(false);
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
            const meeting = normalizeMeeting({ ...meetingDetails, transcript, decisions, action_items: actionItems, title: meetingDetails.title || result.filename || file.name });
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

    function saveMicrophoneDevice(deviceId) {
        setMicrophoneDeviceId(deviceId);
        if (deviceId) localStorage.setItem('afterword.microphoneDeviceId', deviceId);
        else localStorage.removeItem('afterword.microphoneDeviceId');
    }

    async function startRecording() {
        if (!navigator.mediaDevices?.getDisplayMedia || !window.MediaRecorder) {
            setNotice('This desktop environment does not support Google Meet audio capture.');
            return;
        }
        let microphoneStream = null;
        let microphoneUnavailable = false;
        try {
            // Request the mic FIRST, before Meet's tab is captured for sharing.
            // This avoids racing Meet for exclusive device access on Windows/driver setups
            // that lock the mic once a second consumer tries to open it mid-call.
            // If a virtual loopback device (VB-Cable/BlackHole) has been selected in
            // Settings, echoCancellation is disabled since there's no real acoustic
            // echo for the browser to cancel on a virtual device, and forcing it on
            // can distort or mute the signal.
            const selectedLabel = audioDevices.find((d) => d.deviceId === microphoneDeviceId)?.label;
            const usingVirtualDevice = isLikelyVirtualDevice(selectedLabel);
            try {
                microphoneStream = await navigator.mediaDevices.getUserMedia({
                    audio: {
                        ...(microphoneDeviceId ? { deviceId: { exact: microphoneDeviceId } } : {}),
                        echoCancellation: !usingVirtualDevice,
                        noiseSuppression: true,
                        autoGainControl: true,
                    },
                });
            } catch (error) {
                microphoneUnavailable = true;
                console.error('Mic getUserMedia failed:', error.name, error.message);
                setNotice(`Microphone unavailable: ${error.name || 'unknown error'} — ${error.message || 'device in use'}. You will need to release it from Meet or another app before recording your own voice.`);
            }

            setNotice(microphoneStream ? 'Microphone ready. Now select the Google Meet browser tab and enable Share audio.' : 'Select the Google Meet browser tab and enable Share audio. Tab sharing is required for Meet audio.');
            const displayStream = await navigator.mediaDevices.getDisplayMedia({
                video: { displaySurface: 'browser' },
                audio: { suppressLocalAudioPlayback: false },
                preferCurrentTab: false,
                selfBrowserSurface: 'exclude',
                surfaceSwitching: 'include',
                systemAudio: 'include',
            });
            const audioTracks = displayStream.getAudioTracks();
            if (!audioTracks.length) {
                displayStream.getTracks().forEach((track) => track.stop());
                microphoneStream?.getTracks().forEach((track) => track.stop());
                throw new Error('No Meet audio was shared. Select the Meet tab/window and enable Share audio.');
            }
            displayStream.getVideoTracks().forEach((track) => track.stop());

            sourceStreamsRef.current = microphoneStream ? [displayStream, microphoneStream] : [displayStream];
            audioSignalDetectedRef.current = false;

            const AudioContextClass = window.AudioContext || window.webkitAudioContext;
            if (!AudioContextClass) throw new Error('This desktop environment cannot mix Meet audio and microphone audio.');
            const audioContext = new AudioContextClass();
            const destination = audioContext.createMediaStreamDestination();
            const analyser = audioContext.createAnalyser();
            analyser.fftSize = 512;

            const meetSource = audioContext.createMediaStreamSource(new MediaStream(displayStream.getAudioTracks()));
            meetSource.connect(destination);
            meetSource.connect(analyser);

            if (microphoneStream) {
                const microphoneSource = audioContext.createMediaStreamSource(microphoneStream);
                microphoneSource.connect(destination);
                microphoneSource.connect(analyser);
            }

            const silentOutput = audioContext.createGain();
            silentOutput.gain.value = 0;
            analyser.connect(silentOutput);
            silentOutput.connect(audioContext.destination);
            await audioContext.resume();
            audioContextRef.current = audioContext;

            const stream = destination.stream;
            const samples = new Uint8Array(analyser.fftSize);
            audioMonitorTimerRef.current = setInterval(() => {
                analyser.getByteTimeDomainData(samples);
                let peak = 0;
                for (const sample of samples) peak = Math.max(peak, Math.abs(sample - 128));
                if (peak > 2) audioSignalDetectedRef.current = true;
            }, 250);

            const mimeType = ['audio/webm;codecs=opus', 'audio/webm', 'audio/ogg'].find((type) => MediaRecorder.isTypeSupported(type));
            const recorder = new MediaRecorder(stream, mimeType ? { mimeType } : undefined);
            recordingChunksRef.current = [];
            streamRef.current = stream;
            recorderRef.current = recorder;
            recorder.ondataavailable = (event) => {
                if (event.data.size) recordingChunksRef.current.push(event.data);
            };
            recorder.onstop = async () => {
                const blob = new Blob(recordingChunksRef.current, { type: recorder.mimeType || 'audio/webm' });
                stream.getTracks().forEach((track) => track.stop());
                sourceStreamsRef.current.forEach((source) => source.getTracks().forEach((track) => track.stop()));
                sourceStreamsRef.current = [];
                clearInterval(audioMonitorTimerRef.current);
                const hadAudioSignal = audioSignalDetectedRef.current;
                await audioContextRef.current?.close();
                audioContextRef.current = null;
                streamRef.current = null;
                recorderRef.current = null;
                clearInterval(recordingTimerRef.current);
                setRecording(false);
                setRecordingSeconds(0);
                if (blob.size < 1000 || !hadAudioSignal) {
                    setNotice(microphoneUnavailable ? 'Your microphone is in use by another app, and no remote participant audio was detected either. Release the mic and confirm Share audio is enabled.' : 'No audio signal was detected. Confirm Meet is playing audio and microphone access was allowed.');
                    return;
                }
                await processRecording(new File([blob], `meeting-${new Date().toISOString().replace(/[:.]/g, '-')}.webm`, { type: blob.type }));
            };
            recorder.start(1000);
            setRecordingSeconds(0);
            setRecording(true);
            recordingTimerRef.current = setInterval(() => setRecordingSeconds((seconds) => seconds + 1), 1000);
            setNotice(microphoneStream ? 'Recording Meet audio and your microphone. You can switch apps now; return here to stop.' : 'Recording Meet audio only. Your microphone is in use by another app.');
        } catch (error) {
            clearInterval(audioMonitorTimerRef.current);
            sourceStreamsRef.current.forEach((source) => source.getTracks().forEach((track) => track.stop()));
            sourceStreamsRef.current = [];
            microphoneStream?.getTracks().forEach((track) => track.stop());
            setNotice(error instanceof Error ? `Recording could not start: ${error.message}` : 'Recording could not start. Check microphone permissions.');
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

    function saveWorkspaceName(value) {
        const nextName = value.trim();
        if (nextName) localStorage.setItem('afterword.workspaceName', nextName);
        else localStorage.removeItem('afterword.workspaceName');
        setWorkspaceName(nextName);
    }

    return (
        <div className="app-shell">
            <aside className="sidebar">
                <div className="brand"><span className="brand-mark">A</span><span>afterword</span></div>
                <div className="workspace-label">PERSONAL WORKSPACE</div>
                <nav>{navItems.map((item) => <button className={`nav-item ${view === item.id || (view === 'detail' && item.id === 'meetings') ? 'active' : ''}`} key={item.id} onClick={() => setView(item.id)}><span className="nav-glyph">{item.glyph}</span>{item.label}</button>)}</nav>
                <div className="sidebar-bottom"><div className="status-line"><span className={`status-dot ${apiOnline ? 'online' : ''}`}></span>{apiOnline ? 'Service connected' : 'Backend unavailable'}</div><div className="profile"><span>{workspaceName ? workspaceName.slice(0, 2).toUpperCase() : 'LW'}</span><div><strong>{workspaceName || 'Local workspace'}</strong><small>{workspaceName ? 'Local profile' : 'No account required'}</small></div><b>...</b></div></div>
            </aside>
            <main className="main-content">
                <header className="topbar"><div className="breadcrumbs">Local workspace <span>/</span> {view === 'detail' ? selectedMeeting?.title : view === 'overview' ? 'Overview' : view === 'actions' ? 'Action items' : view === 'settings' ? 'Settings' : 'Meetings'}</div><div className="topbar-actions"><button className="icon-button" aria-label="Search">⌕</button><button className="keyboard-hint">Ctrl K</button></div></header>
                {notice && <div className="notice">{notice}<button onClick={() => setNotice('')}>Dismiss</button></div>}
                <div className="page-wrap">
                    {view === 'overview' && <Overview meetings={meetings} openActions={openActions} onOpenMeeting={openMeeting} onUpload={() => fileInput.current?.click()} onRecord={toggleRecording} recording={recording} recordingSeconds={recordingSeconds} workspaceName={workspaceName} />}
                    {view === 'meetings' && <MeetingLibrary meetings={filteredMeetings} query={query} setQuery={setQuery} onOpenMeeting={openMeeting} onUpload={() => fileInput.current?.click()} onRecord={toggleRecording} recording={recording} recordingSeconds={recordingSeconds} onRefresh={loadMeetingsFromDatabase} />}
                    {view === 'actions' && <ActionView items={actionItems} onToggle={toggleAction} onOpenMeeting={openMeeting} />}
                    {view === 'settings' && <Settings apiBaseUrl={apiBaseUrl} apiOnline={apiOnline} workspaceName={workspaceName} onSaveApiBaseUrl={saveApiBaseUrl} onSaveWorkspaceName={saveWorkspaceName} audioDevices={audioDevices} microphoneDeviceId={microphoneDeviceId} onSaveMicrophoneDevice={saveMicrophoneDevice} />}
                    {view === 'detail' && selectedMeeting && <>
                        <MeetingAudio meeting={selectedMeeting} />
                        <MeetingDetail
                            meeting={selectedMeeting}
                            onBack={() => setView('meetings')}
                            onToggle={toggleAction}
                            editingTitle={editingMeetingTitle}
                            titleDraft={meetingTitleDraft}
                            onBeginTitleEdit={() => beginMeetingTitleEdit(selectedMeeting)}
                            onTitleDraftChange={setMeetingTitleDraft}
                            onSaveTitle={() => saveMeetingTitle(selectedMeeting)}
                            onCancelTitleEdit={() => setEditingMeetingTitle(false)}
                            onGenerateSummary={() => regenerateSummary(selectedMeeting)}
                            generatingSummary={generatingSummary}
                            onDelete={() => deleteMeeting(selectedMeeting)}
                        />
                        <SpeakerEditor meeting={selectedMeeting} onSave={renameSpeakers} />
                    </>}
                </div>
                <input ref={fileInput} type="file" accept="audio/*,video/*,.webm,.mp4" hidden onChange={uploadRecording} />
                {uploading && <div className="processing"><span className="spinner"></span><div><strong>Analyzing your recording</strong><small>Transcription, speakers, and takeaways</small></div></div>}
            </main>
        </div>
    )
}

function PageIntro({ eyebrow, title, children }) { return <div className="page-intro"><div><div className="eyebrow">{eyebrow}</div><h1>{title}</h1></div>{children}</div>; }

function RecordingButton({ onRecord, recording, recordingSeconds }) {
    return <button className={`record-button ${recording ? 'is-recording' : ''}`} onClick={onRecord} aria-label={recording ? 'Stop recording' : 'Start recording'}>
        <span className="record-dot"></span>{recording ? `Stop ${formatTime(recordingSeconds)}` : 'Record Meet audio'}
    </button>;
}

function Overview({ meetings, openActions, onOpenMeeting, onUpload, onRecord, recording, recordingSeconds, workspaceName }) {
    const totalMinutes = Math.round(meetings.reduce((sum, meeting) => sum + (meeting.duration_seconds || 0), 0) / 60);
    return <>
        <PageIntro eyebrow="Monday, September 21, 2026" title={workspaceName ? `Good morning, ${workspaceName}.` : 'Good morning.'}><div className="intro-actions"><RecordingButton onRecord={onRecord} recording={recording} recordingSeconds={recordingSeconds} /><button className="primary-button" onClick={onUpload}><span>+</span> Import recording</button></div></PageIntro>
        <section className="hero-panel"><div><div className="eyebrow warm">YOUR MEETING MEMORY</div><h2>Make every conversation<br /><em>move things forward.</em></h2><p>Transcripts, decisions, and next steps in one quiet place.</p></div><div className="hero-stats"><div><strong>{meetings.length}</strong><span>Meetings captured</span></div><div><strong>{totalMinutes}<small>m</small></strong><span>Conversation indexed</span></div></div></section>
        <div className="section-heading"><div><div className="eyebrow">RECENTLY CAPTURED</div><h2>Your meetings</h2></div><button className="text-button" onClick={() => document.querySelector('.nav-item:nth-child(2)')?.click()}>View all <span>→</span></button></div>
        {meetings.length ? <div className="meeting-grid">{meetings.slice(0, 3).map((meeting, index) => <MeetingCard key={meeting.id} meeting={meeting} index={index} onClick={() => onOpenMeeting(meeting.id)} />)}</div> : <div className="empty-state large">No meetings yet. Start a recording or import one to create your first meeting.</div>}
        <section className="lower-grid"><div className="panel"><div className="panel-heading"><div><div className="eyebrow">NEEDS YOUR ATTENTION</div><h3>Open action items</h3></div><span className="count-badge">{openActions.length}</span></div>{openActions.slice(0, 3).map((item) => <ActionRow key={item.id} item={item} />)}{!openActions.length && <div className="empty-state">You are all caught up.</div>}</div><div className="quote-panel"><div className="quote-mark">“</div><p>Good notes do not just remember the conversation. They make the next conversation easier.</p><span>— Afterword principle</span></div></section>
    </>;
}

function MeetingCard({ meeting, index, onClick }) { return <button className={`meeting-card card-${index}`} onClick={onClick}><div className="card-top"><span className="waveform"><i></i><i></i><i></i><i></i><i></i><i></i><i></i></span><span>{formatDuration(meeting.duration_seconds)}</span></div><h3>{meeting.title}</h3><p>{meeting.summary || 'Transcript and meeting insights are ready to review.'}</p><div className="card-footer"><span>{meeting.date || new Date(meeting.started_at).toLocaleDateString()}</span><span className="arrow">↗</span></div></button>; }

function MeetingLibrary({ meetings, query, setQuery, onOpenMeeting, onUpload, onRecord, recording, recordingSeconds, onRefresh }) { return <><PageIntro eyebrow="MEETING LIBRARY" title="All meetings"><div className="intro-actions"><RecordingButton onRecord={onRecord} recording={recording} recordingSeconds={recordingSeconds} /><button className="primary-button" onClick={onUpload}><span>+</span> Import recording</button></div></PageIntro><div className="library-toolbar"><div className="search-field"><span>⌕</span><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search meetings" /><kbd>⌘ F</kbd></div><div className="library-tools"><span className="result-count">{meetings.length} conversations</span><button className="refresh-button" onClick={onRefresh}>Refresh database</button></div></div><div className="library-list">{meetings.map((meeting) => <button className="library-row" key={meeting.id} onClick={() => onOpenMeeting(meeting.id)}><span className="row-date">{meeting.date || new Date(meeting.started_at).toLocaleDateString()}</span><span className="row-title"><strong>{meeting.title}</strong><small>{meeting.summary || 'No summary available yet.'}</small></span><span className="row-duration">{formatDuration(meeting.duration_seconds)}</span><span className="row-arrow">→</span></button>)}</div>{!meetings.length && <div className="empty-state large">No meetings match that search.</div>}</>; }

function ActionRow({ item, onToggle }) { return <div className="action-row"><button className={`checkbox ${item.completed ? 'checked' : ''}`} onClick={() => onToggle(item)}>{item.completed ? '✓' : ''}</button><div><strong className={item.completed ? 'completed-text' : ''}>{item.task}</strong><small>{item.assignee || 'Unassigned'} · {item.meetingTitle || 'Meeting'}</small></div></div>; }

function ActionView({ items, onToggle, onOpenMeeting }) { return <><PageIntro eyebrow="FOLLOW THROUGH" title="Action items"><span className="subtle-label">{items.filter((item) => !item.completed).length} open</span></PageIntro><div className="action-layout"><div className="panel actions-panel"><div className="panel-heading"><div><div className="eyebrow">YOUR NEXT STEPS</div><h3>Everything in one list</h3></div></div>{items.map((item) => <div className="action-row clickable" key={item.id} onClick={() => onOpenMeeting(item.meetingId)}><button className={`checkbox ${item.completed ? 'checked' : ''}`} onClick={(event) => { event.stopPropagation(); onToggle(item); }}>{item.completed ? '✓' : ''}</button><div><strong className={item.completed ? 'completed-text' : ''}>{item.task}</strong><small>{item.assignee || 'Unassigned'} · {item.meetingTitle}</small></div><span className="row-arrow">→</span></div>)}</div><div className="side-note"><div className="eyebrow">A SMALL PROMPT</div><h3>Close the loop.</h3><p>The best action item is the one that disappears because it got done.</p></div></div></>; }

function SpeakerEditor({ meeting, onSave }) {
    const speakers = [...new Map((meeting.transcript || []).map((segment) => {
        const id = segment.speaker_id || segment.speaker;
        return [id, { id, label: segment.speaker || id }];
    })).values()];
    const [draft, setDraft] = useState({});
    useEffect(() => {
        setDraft(Object.fromEntries(speakers.map((speaker) => [speaker.id, speaker.label])));
    }, [meeting.id, meeting.transcript]);

    if (!speakers.length) return null;
    return <section className="panel speaker-panel"><div className="panel-heading"><div><div className="eyebrow">SPEAKER MANAGEMENT</div><h3>Rename speakers</h3></div><span className="subtle-label">IDs stay stable</span></div><p className="settings-copy">Names change how speakers are displayed; backend speaker identifiers remain unchanged.</p>{speakers.map((speaker) => <label className="speaker-field" key={speaker.id}><span>{speaker.id}</span><input value={draft[speaker.id] || ''} onChange={(event) => setDraft((current) => ({ ...current, [speaker.id]: event.target.value }))} /></label>)}<button className="primary-button" onClick={() => onSave(meeting.id, Object.fromEntries(Object.entries(draft).map(([id, name]) => [id, name.trim() || id])))}>Save speaker names</button></section>;
}

function MeetingAudio({ meeting }) {
    const audioRef = useRef(null);

    useEffect(() => {
        function handleTimestampClick(event) {
            const timestamp = event.target.closest('.timestamp');
            if (!timestamp || !audioRef.current) return;
            const parts = timestamp.textContent.trim().split(':').map(Number);
            const seconds = parts.length === 2 ? parts[0] * 60 + parts[1] : 0;
            audioRef.current.currentTime = seconds;
            audioRef.current.play().catch(() => { });
        }
        document.addEventListener('click', handleTimestampClick);
        return () => document.removeEventListener('click', handleTimestampClick);
    }, []);

    return <div className="audio-player"><div className="eyebrow">MEETING AUDIO</div><audio ref={audioRef} controls preload="metadata" src={getMeetingAudioUrl(meeting.id)} /><small>Click a transcript timestamp to jump to that moment.</small></div>;
}

function MeetingDetail({ meeting, onBack, onToggle, editingTitle, titleDraft, onBeginTitleEdit, onTitleDraftChange, onSaveTitle, onCancelTitleEdit, onGenerateSummary, generatingSummary, onDelete }) {
    return <>
        <button className="back-button" onClick={onBack}>← Back to meetings</button>
        <div className="detail-heading">
            <div>
                <div className="eyebrow">{meeting.date || 'CAPTURED MEETING'}</div>
                {editingTitle ? <div className="title-editor"><input className="meeting-title-input" value={titleDraft} onChange={(event) => onTitleDraftChange(event.target.value)} onKeyDown={(event) => { if (event.key === 'Enter') onSaveTitle(); if (event.key === 'Escape') onCancelTitleEdit(); }} autoFocus /><button className="title-save-button" onClick={onSaveTitle}>Save</button><button className="title-cancel-button" onClick={onCancelTitleEdit}>Cancel</button></div> : <div className="title-line"><h1>{meeting.title}</h1><button className="edit-title-button" onClick={onBeginTitleEdit} aria-label="Edit meeting name" title="Edit meeting name">Edit</button></div>}
                <p className="detail-meta">{formatDuration(meeting.duration_seconds)} <span>·</span> {meeting.transcript?.length || 0} transcript segments <span>·</span> {meeting.action_items?.length || 0} action items</p>
            </div>
            <div className="detail-header-actions">
                <button className="icon-danger-button" onClick={onDelete} aria-label="Delete meeting" title="Delete meeting">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                        <path d="M4 7h16" />
                        <path d="M9 7V4h6v3" />
                        <path d="M6 7l1 13h10l1-13" />
                        <path d="M10 11v6" />
                        <path d="M14 11v6" />
                    </svg>
                </button>
            </div>
        </div>
        <section className="summary-banner">
            <div className="summary-icon">✦</div>
            <div className="summary-banner-body">
                <div className="eyebrow">AI SUMMARY</div>
                {meeting.summary
                    ? <SummaryContent summary={meeting.summary} />
                    : <p>This meeting has not been summarized yet.</p>}
                {!meeting.summary && (
                    <button className="generate-summary-button" onClick={onGenerateSummary} disabled={generatingSummary}>
                        <span className="sparkle">✦</span>{generatingSummary ? 'Generating…' : 'Generate AI summary'}
                    </button>
                )}
            </div>
        </section>
        <div className="detail-grid">
            <div className="transcript-panel">
                <div className="panel-heading"><div><div className="eyebrow">CONVERSATION</div><h3>Transcript</h3></div><button className="text-button">Export <span>↓</span></button></div>
                {meeting.transcript?.length ? meeting.transcript.map((segment, index) => <div className="transcript-line" key={`${segment.start}-${index}`}><button className="timestamp">{formatTime(segment.start)}</button><div><strong>{segment.speaker?.replace('SPEAKER_', 'Speaker ')}</strong><p>{segment.text}</p></div></div>) : <div className="empty-state">Transcript segments will appear here after processing.</div>}
            </div>
            <aside className="detail-side">
                <div className="panel">
                    <div className="panel-heading"><div><div className="eyebrow">FOLLOW THROUGH</div><h3>Action items</h3></div><span className="count-badge">{meeting.action_items?.length || 0}</span></div>
                    {meeting.action_items?.map((item) => <ActionRow key={item.id} item={{ ...item, meetingId: meeting.id, meetingTitle: meeting.title }} onToggle={onToggle} />)}
                    {!meeting.action_items?.length && <div className="empty-state">No action items found.</div>}
                </div>
                <div className="panel decisions-panel">
                    <div className="panel-heading"><div><div className="eyebrow">AGREEMENTS</div><h3>Decisions</h3></div></div>
                    {meeting.decisions?.map((decision) => <div className="decision" key={decision.id || decision.timestamp_seconds}><span>✓</span><p>{decision.decision || decision.text}</p></div>)}
                    {!meeting.decisions?.length && <div className="empty-state">No decisions found.</div>}
                </div>
            </aside>
        </div>
    </>;
}

function Settings({ apiBaseUrl, apiOnline, workspaceName, onSaveApiBaseUrl, onSaveWorkspaceName, audioDevices, microphoneDeviceId, onSaveMicrophoneDevice }) {
    const [draftUrl, setDraftUrl] = useState(apiBaseUrl);
    const [draftName, setDraftName] = useState(workspaceName);
    return <>
        <PageIntro eyebrow="APPLICATION SETTINGS" title="Settings" />
        <div className="settings-grid">
            <section className="panel settings-panel"><div className="eyebrow">LOCAL PROFILE</div><h3>Workspace identity</h3><p className="settings-copy">Optional. This name stays on this computer and is only used for the greeting and local workspace label.</p><label className="settings-label" htmlFor="workspace-name">Your name</label><input id="workspace-name" className="settings-input" value={draftName} onChange={(event) => setDraftName(event.target.value)} placeholder="Leave blank for a private workspace" /><button className="primary-button" onClick={() => onSaveWorkspaceName(draftName)}>Save local name</button></section><section className="panel settings-panel"><div className="eyebrow">BACKEND</div><h3>Connection</h3><p className="settings-copy">The desktop app sends recordings and meeting queries to your configured local service.</p><label className="settings-label" htmlFor="api-url">API base URL</label><input id="api-url" className="settings-input" value={draftUrl} onChange={(event) => setDraftUrl(event.target.value)} /><div className="connection-state"><span className={`status-dot ${apiOnline ? 'online' : ''}`}></span>{apiOnline ? 'Connected' : 'Unavailable'}<span className="connection-url">{apiBaseUrl}</span></div><button className="primary-button" onClick={() => onSaveApiBaseUrl(draftUrl)}>Save and test connection</button></section>
            <section className="panel settings-panel">
                <div className="eyebrow">AUDIO INPUT</div>
                <h3>Microphone device</h3>
                <p className="settings-copy">
                    If Meet and this app conflict over your physical microphone, install a virtual
                    audio device (VB-Cable on Windows, BlackHole on macOS), set it as your Meet
                    input, and select it below — both apps can then read from it without contention.
                </p>
                <label className="settings-label" htmlFor="mic-device">Recording device</label>
                <select
                    id="mic-device"
                    className="settings-input"
                    value={microphoneDeviceId}
                    onChange={(event) => onSaveMicrophoneDevice(event.target.value)}
                >
                    <option value="">System default</option>
                    {audioDevices.map((device) => (
                        <option key={device.deviceId} value={device.deviceId}>
                            {device.label || `Microphone ${device.deviceId.slice(0, 6)}`}
                            {/(cable|blackhole|loopback|virtual)/i.test(device.label || '') ? ' (virtual — recommended)' : ''}
                        </option>
                    ))}
                </select>
            </section>
            <section className="panel settings-panel"><div className="eyebrow">PRIVACY</div><h3>Local-first processing</h3><p className="settings-copy">Audio is sent only to the backend URL above. This client does not add cloud AI services or store meeting data in browser storage.</p><div className="capability-list"><div><span className="status-dot online"></span><strong>Microphone recording</strong><small>Captured in the desktop webview</small></div><div><span className="status-dot online"></span><strong>Transcription and diarization</strong><small>Provided by the Go backend</small></div><div><span className="status-dot"></span><strong>Semantic search and Q&amp;A</strong><small>Backend routes are not currently exposed</small></div></div></section>
        </div>
    </>;
}

export default App