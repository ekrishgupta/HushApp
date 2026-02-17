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
            {/* Ghost emoji */}
            <div style={{ fontSize: '48px', marginBottom: '8px' }}>👻</div>

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
                ghost chat — serverless, encrypted, local
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
        return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`;
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
                    messages.map((msg, i) => {
                        const isMe = msg.sender === username;
                        return (
                            <div key={`${msg.timestamp}-${i}`} style={{ whiteSpace: 'pre' }}>
                                <span style={{ color: 'var(--dim-gray)' }}>
                                    {'  '}{formatTime(msg.timestamp)}{'  '}
                                </span>
                                <span style={{
                                    color: isMe ? 'var(--soft-green)' : 'var(--ghost-pink)',
                                    fontWeight: 'bold',
                                }}>
                                    {isMe ? 'you' : msg.sender}
                                </span>
                                <span style={{ color: 'var(--warm-white)' }}>
                                    : {msg.content}
                                </span>
                            </div>
                        );
                    })
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
                        alignItems: 'center',
                    }}
                >
                    <input
                        ref={inputRef}
                        type="text"
                        value={inputText}
                        onChange={(e) => setInputText(e.target.value)}
                        onKeyDown={handleKeyDown}
                        placeholder="type a message..."
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
