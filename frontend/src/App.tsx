import { useState, useEffect, useRef } from 'react';

// Wails bindings
import { SendMessage, GetUsername, GetPeerCount, SetUsername } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';

interface ChatMessage {
    sender: string;
    content: string;
    timestamp: number;
}

// ──────────────────────────────────────────────────
//  ASCII Art (large "HUSH" banner)
// ──────────────────────────────────────────────────
const HUSH_ASCII = `
 ██╗  ██╗██╗   ██╗███████╗██╗  ██╗
 ██║  ██║██║   ██║██╔════╝██║  ██║
 ███████║██║   ██║███████╗███████║
 ██╔══██║██║   ██║╚════██║██╔══██║
 ██║  ██║╚██████╔╝███████║██║  ██║
 ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═╝
`;

// ──────────────────────────────────────────────────
//  Welcome Screen
// ──────────────────────────────────────────────────
function WelcomeScreen({ onEnter }: { onEnter: (name: string) => void }) {
    const [name, setName] = useState('');
    const inputRef = useRef<HTMLInputElement>(null);

    useEffect(() => {
        inputRef.current?.focus();
    }, []);

    const handleSubmit = () => {
        const trimmed = name.trim();
        if (!trimmed) return;
        onEnter(trimmed);
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            handleSubmit();
        }
    };

    return (
        <div
            style={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                justifyContent: 'center',
                height: '100vh',
                width: '100vw',
                background: 'var(--bg)',
                fontFamily: "'Menlo', 'Monaco', 'Courier New', monospace",
                color: 'var(--warm-white)',
                WebkitAppRegion: 'drag',
            } as any}
        >


            {/* ASCII Art */}
            <pre
                style={{
                    color: 'var(--ghost-purple)',
                    fontWeight: 'bold',
                    fontSize: '13px',
                    lineHeight: '1.1',
                    textAlign: 'center',
                    margin: 0,
                    userSelect: 'none',
                }}
            >
                {HUSH_ASCII}
            </pre>

            {/* Subtitle */}
            <div
                style={{
                    color: 'var(--dim-gray)',
                    fontStyle: 'italic',
                    fontSize: '13px',
                    marginTop: '4px',
                    marginBottom: '24px',
                    userSelect: 'none',
                }}
            >
                talk to anyone on your wifi — no servers, no trace
            </div>

            {/* Username input box */}
            <div
                style={{
                    border: '1px solid var(--ghost-purple)',
                    borderRadius: '6px',
                    padding: '6px 12px',
                    display: 'flex',
                    alignItems: 'center',
                    width: '280px',
                    WebkitAppRegion: 'no-drag',
                } as any}
            >
                <span style={{ color: 'var(--dim-gray)', marginRight: '8px', userSelect: 'none' }}>{'>'}</span>
                <input
                    ref={inputRef}
                    type="text"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    onKeyDown={handleKeyDown}
                    placeholder="enter your name..."
                    spellCheck={false}
                    autoFocus
                    style={{
                        width: '100%',
                        background: 'transparent',
                        border: 'none',
                        outline: 'none',
                        color: 'var(--warm-white)',
                        fontFamily: "'Menlo', 'Monaco', 'Courier New', monospace",
                        fontSize: '14px',
                        caretColor: 'var(--warm-white)',
                    }}
                />
            </div>

            {/* Hint */}
            <div
                style={{
                    color: 'var(--dim-gray)',
                    fontSize: '11px',
                    marginTop: '12px',
                    userSelect: 'none',
                }}
            >
                press enter to join
            </div>
        </div>
    );
}

// ──────────────────────────────────────────────────
//  Message Item Component
// ──────────────────────────────────────────────────
function MessageItem({ msg, username, formatTime }: { msg: ChatMessage, username: string, formatTime: (ts: number) => string }) {
    const [expanded, setExpanded] = useState(false);
    const isMe = msg.sender === username;

    return (
        <div
            onClick={() => setExpanded(!expanded)}
            style={{
                cursor: 'pointer',
                padding: '2px 8px',
                userSelect: 'text',
            }}
        >
            {!expanded ? (
                // Compact View
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <div style={{ display: 'flex', overflow: 'hidden', alignItems: 'center', flex: 1 }}>
                        <span style={{
                            color: isMe ? 'var(--soft-green)' : 'var(--ghost-pink)',
                            fontWeight: 'bold',
                            whiteSpace: 'nowrap',
                            marginRight: '8px',
                        }}>
                            {isMe ? 'you' : msg.sender}
                        </span>
                        <span style={{ color: 'var(--warm-white)' }}>: </span>
                        <span style={{
                            color: 'var(--warm-white)',
                            whiteSpace: 'nowrap',
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            marginLeft: '4px',
                            flex: 1,
                        }}>
                            {msg.content}
                        </span>
                        <span style={{ color: 'var(--dim-gray)', marginLeft: '4px' }}>(...)</span>
                    </div>
                    <span style={{ color: 'var(--dim-gray)', flexShrink: 0, marginLeft: '16px' }}>
                        {formatTime(msg.timestamp)}
                    </span>
                </div>
            ) : (
                // Expanded View
                <div>
                    <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                        <span>
                            <span style={{ color: '#ff79c6', marginRight: '8px' }}>{'>'}</span>
                            <span style={{
                                color: isMe ? 'var(--soft-green)' : 'var(--ghost-pink)',
                                fontWeight: 'bold',
                            }}>
                                {isMe ? 'you' : msg.sender}
                            </span>
                            <span style={{ color: 'var(--warm-white)' }}>:</span>
                        </span>
                        <span style={{ color: 'var(--dim-gray)' }}>
                            {formatTime(msg.timestamp)}
                        </span>
                    </div>
                    <div style={{
                        color: 'var(--warm-white)',
                        whiteSpace: 'pre-wrap',
                        marginLeft: '22px', // Approx indent
                        marginTop: '2px',
                        wordBreak: 'break-word',
                    }}>
                        {msg.content}
                    </div>
                </div>
            )}
        </div>
    );
}

// ──────────────────────────────────────────────────
//  Chat Screen (exact TUI replica)
// ──────────────────────────────────────────────────
function ChatScreen({ username }: { username: string }) {
    const [messages, setMessages] = useState<ChatMessage[]>([]);
    const [inputText, setInputText] = useState('');
    const [peerCount, setPeerCount] = useState(0);
    const [showWarning, setShowWarning] = useState(false);
    const [lastSent, setLastSent] = useState(0);
    const viewportRef = useRef<HTMLDivElement>(null);
    const inputRef = useRef<HTMLInputElement>(null);

    useEffect(() => {
        GetPeerCount().then(setPeerCount);

        const interval = setInterval(() => {
            GetPeerCount().then(setPeerCount);
        }, 1000);

        EventsOn('new_message', (msg: ChatMessage) => {
            setMessages((prev) => [...prev, msg]);
        });

        inputRef.current?.focus();
        return () => clearInterval(interval);
    }, []);

    // Auto-scroll
    useEffect(() => {
        if (viewportRef.current) {
            viewportRef.current.scrollTop = viewportRef.current.scrollHeight;
        }
    }, [messages]);

    const formatTime = (ts: number) => {
        const d = new Date(ts * 1000);
        return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}:${d.getSeconds().toString().padStart(2, '0')}`;
    };

    const handleSend = () => {
        const content = inputText.trim();
        if (!content) return;

        if (Date.now() - lastSent < 1500) {
            setShowWarning(true);
            return;
        }

        setShowWarning(false);
        SendMessage(content);
        setInputText('');
        setLastSent(Date.now());
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            handleSend();
        } else {
            setShowWarning(false);
        }
    };

    // Input Logic
    const [rows, setRows] = useState(1);
    const [placeholderShown, setPlaceholderShown] = useState(true);

    const handleInputChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
        const val = e.target.value;
        setInputText(val);
        if (placeholderShown && val.length > 0) {
            setPlaceholderShown(false);
        }

        // Simple row calculation based on newlines
        const lineCount = val.split('\n').length;
        setRows(Math.min(Math.max(lineCount, 1), 5));
    };

    const handleSendKey = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            handleSend();
            setRows(1);
        }
    };

    // Override handleSend to clear rows too
    const handleGUI_Send = () => {
        handleSend();
        setRows(1);
    }

    const borderColor = showWarning ? 'var(--warning-red)' : 'var(--ghost-purple)';

    return (
        <div
            style={{
                display: 'flex',
                flexDirection: 'column',
                height: '100vh',
                width: '100vw',
                background: 'var(--bg)',
                fontFamily: "'Menlo', 'Monaco', 'Courier New', monospace",
                fontSize: '14px',
                lineHeight: '1.4',
                color: 'var(--warm-white)',
            }}
        >
            {/* Header */}
            <div style={{ padding: '4px 8px', WebkitAppRegion: 'drag' } as any}>
                <span style={{ color: 'var(--ghost-purple)', fontWeight: 'bold' }}>
                    👻 Hush — Ghost Chat
                </span>
            </div>

            {/* Status */}
            <div style={{ padding: '0 8px', color: 'var(--dim-gray)', fontStyle: 'italic' }}>
                {'  '}online as {username}{'  '}({peerCount} active peers)
            </div>

            {/* Divider */}
            <div style={{ padding: '2px 0', color: 'var(--dim-gray)', overflow: 'hidden', whiteSpace: 'nowrap' }}>
                {'─'.repeat(200)}
            </div>

            {/* Messages */}
            <div
                ref={viewportRef}
                style={{
                    flex: 1,
                    overflowY: 'auto',
                    padding: '2px 0',
                    minHeight: 0,
                }}
            >
                {messages.length === 0 ? (
                    <div style={{ color: 'var(--dim-gray)', fontStyle: 'italic', padding: '4px 8px' }}>
                        {'  '}waiting for ghosts to appear... 👻
                    </div>
                ) : (
                    messages.map((msg, i) => (
                        <MessageItem
                            key={`${msg.timestamp}-${i}`}
                            msg={msg}
                            username={username}
                            formatTime={formatTime}
                        />
                    ))
                )}
            </div>

            {/* Divider */}
            <div style={{ padding: '2px 0', color: 'var(--dim-gray)', overflow: 'hidden', whiteSpace: 'nowrap' }}>
                {'─'.repeat(200)}
            </div>

            {/* Warning */}
            <div style={{ height: '20px', padding: '0 8px' }}>
                {showWarning && (
                    <span style={{ color: 'var(--warning-red)', fontWeight: 'bold' }}>
                        {'  '}⚡ Slow down!
                    </span>
                )}
            </div>

            {/* Input */}
            <div style={{ padding: '0 8px 8px 8px' }}>
                <div
                    style={{
                        border: `1px solid ${borderColor}`,
                        borderRadius: '6px',
                        padding: '4px 8px',
                        display: 'flex',
                        alignItems: 'flex-start', // Top align for multiline
                    }}
                >
                    <span style={{ marginRight: '8px', userSelect: 'none', color: 'var(--dim-gray)' }}>
                        {rows > 1 ? '  ' : '>'}
                    </span>
                    <textarea
                        ref={inputRef as any}
                        value={inputText}
                        onChange={handleInputChange}
                        onKeyDown={handleSendKey} // Updated handler name
                        placeholder={placeholderShown ? "type a message..." : ""}
                        spellCheck={false}
                        autoFocus
                        rows={rows}
                        style={{
                            width: '100%',
                            background: 'transparent',
                            border: 'none',
                            outline: 'none',
                            color: 'var(--warm-white)',
                            fontFamily: "'Menlo', 'Monaco', 'Courier New', monospace",
                            fontSize: '14px',
                            caretColor: 'var(--warm-white)',
                            resize: 'none',
                            overflow: 'hidden', // Hide scrollbar usually, or auto?
                            padding: 0,
                            lineHeight: '1.4',
                        }}
                    />
                </div>
            </div>
        </div>
    );
}

// ──────────────────────────────────────────────────
//  App — routes between Welcome and Chat
// ──────────────────────────────────────────────────
function App() {
    const [screen, setScreen] = useState<'welcome' | 'chat'>('welcome');
    const [username, setUsernameState] = useState('');

    const handleEnter = (name: string) => {
        setUsernameState(name);
        SetUsername(name);
        setScreen('chat');
    };

    if (screen === 'welcome') {
        return <WelcomeScreen onEnter={handleEnter} />;
    }

    return <ChatScreen username={username} />;
}

export default App;
