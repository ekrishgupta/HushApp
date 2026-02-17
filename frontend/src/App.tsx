import { useState, useEffect, useRef } from 'react';
import { Send, Users } from 'lucide-react';
import { clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

// Import Wails runtime methods
import { SendMessage, GetUsername, GetPeerCount } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';

interface ChatMessage {
    sender: string;
    content: string;
    timestamp: number;
}

function cn(...inputs: (string | undefined | null | false)[]) {
    return twMerge(clsx(inputs));
}

function App() {
    const [messages, setMessages] = useState<ChatMessage[]>([]);
    const [inputText, setInputText] = useState('');
    const [username, setUsername] = useState('');
    const [peerCount, setPeerCount] = useState(0);
    const messagesEndRef = useRef<HTMLDivElement>(null);

    const scrollToBottom = () => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    };

    useEffect(() => {
        // Initial setup
        GetUsername().then(setUsername);
        GetPeerCount().then(setPeerCount);

        // Poll peer count every 2s
        const interval = setInterval(() => {
            GetPeerCount().then(setPeerCount);
        }, 2000);

        // Listen for new messages
        const unsubscribe = EventsOn('new_message', (msg: ChatMessage) => {
            setMessages((prev) => [...prev, msg]);
            setTimeout(scrollToBottom, 100);
        });

        return () => {
            clearInterval(interval);
        };
    }, []);

    const handleSend = () => {
        if (!inputText.trim()) return;
        SendMessage(inputText);
        setInputText('');
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            handleSend();
        }
    };

    return (
        <div className="flex flex-col h-screen bg-slate-900 text-slate-100 font-sans">
            {/* Header */}
            <header className="flex items-center justify-between px-6 py-4 bg-slate-800/50 backdrop-blur-md border-b border-slate-700/50 draggable">
                <div className="flex items-center gap-3">
                    <div className="w-3 h-3 rounded-full bg-red-400/80" />
                    <div className="w-3 h-3 rounded-full bg-amber-400/80" />
                    <div className="w-3 h-3 rounded-full bg-emerald-400/80" />
                    <h1 className="ml-4 text-lg font-semibold tracking-tight text-white/90">HushApp</h1>
                </div>

                <div className="flex items-center gap-4 text-sm font-medium text-slate-400">
                    <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-slate-800/50 border border-slate-700/50">
                        <Users className="w-4 h-4 text-emerald-400" />
                        <span className="text-emerald-400">{peerCount} online</span>
                    </div>
                    <div className="flex items-center gap-2">
                        <div className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
                        <span>{username}</span>
                    </div>
                </div>
            </header>

            {/* Message List */}
            <main className="flex-1 overflow-y-auto p-6 space-y-4 scrollbar-thin scrollbar-thumb-slate-700 scrollbar-track-transparent">
                {messages.length === 0 && (
                    <div className="flex flex-col items-center justify-center h-full text-slate-500 space-y-2 opacity-60">
                        <Users className="w-12 h-12 mb-2" />
                        <p>Waiting for ghosts to appear...</p>
                    </div>
                )}

                {messages.map((msg, i) => {
                    const isMe = msg.sender === username;
                    // Determine if previous message was from same sender to group
                    const isSequence = i > 0 && messages[i - 1].sender === msg.sender;

                    return (
                        <div
                            key={`${msg.timestamp}-${i}`}
                            className={cn(
                                "flex flex-col max-w-[80%]",
                                isMe ? "self-end items-end" : "self-start items-start",
                                isSequence ? "mt-1" : "mt-4"
                            )}
                        >
                            {!isMe && !isSequence && (
                                <span className="text-xs font-medium text-slate-400 mb-1 ml-1">{msg.sender}</span>
                            )}

                            <div
                                className={cn(
                                    "px-4 py-2.5 rounded-2xl text-sm leading-relaxed shadow-sm transition-all duration-200",
                                    isMe
                                        ? "bg-indigo-600 text-white rounded-br-none hover:bg-indigo-500"
                                        : "bg-slate-700 text-slate-100 rounded-bl-none hover:bg-slate-600"
                                )}
                            >
                                {msg.content}
                            </div>

                            {!isSequence && (
                                <span className="text-[10px] text-slate-500 mt-1 px-1 opacity-0 group-hover:opacity-100 transition-opacity">
                                    {new Date(msg.timestamp * 1000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                                </span>
                            )}
                        </div>
                    );
                })}
                <div ref={messagesEndRef} />
            </main>

            {/* Input Area */}
            <footer className="p-4 bg-slate-800/30 backdrop-blur-md border-t border-slate-700/50">
                <div className="relative group">
                    <input
                        type="text"
                        value={inputText}
                        onChange={(e) => setInputText(e.target.value)}
                        onKeyDown={handleKeyDown}
                        placeholder="Type a message..."
                        className="w-full pl-4 pr-12 py-3.5 bg-slate-900/50 border border-slate-700 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/50 focus:border-indigo-500/50 text-slate-200 placeholder-slate-500 transition-all shadow-inner"
                        autoFocus
                    />
                    <button
                        onClick={handleSend}
                        disabled={!inputText.trim()}
                        className="absolute right-2 top-1/2 -translate-y-1/2 p-2 text-slate-400 hover:text-indigo-400 disabled:opacity-50 disabled:hover:text-slate-400 transition-colors"
                    >
                        <Send className="w-5 h-5" />
                    </button>
                </div>
            </footer>
        </div>
    );
}

export default App;
