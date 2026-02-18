import React from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { dracula } from 'react-syntax-highlighter/dist/esm/styles/prism';
import './MarkdownStyles.css';

// Try to use wails runtime if available, otherwise just window.open
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';

interface MarkdownMessageProps {
    content: string;
    compact?: boolean;
}

const MarkdownMessage: React.FC<MarkdownMessageProps> = ({ content, compact }) => {
    // In compact mode, we want a single line preview.
    // We disallow block elements which forces react-markdown to either skip them or unwrap them.
    // unwrapDisallowed=true means <p>text</p> becomes "text".
    const allowedElements = compact
        ? ['a', 'strong', 'em', 'code', 'del', 'span']
        : undefined;

    return (
        <div className={`markdown-content ${compact ? 'compact' : ''}`}>
            <ReactMarkdown
                remarkPlugins={[remarkGfm]}
                allowedElements={allowedElements}
                unwrapDisallowed={compact}
                components={{
                    code({ node, inline, className, children, ...props }: any) {
                        const match = /language-(\w+)/.exec(className || '');
                        // If compact, always render inline code style
                        if (compact) {
                            return (
                                <code className={className} style={{
                                    background: 'rgba(255, 255, 255, 0.1)',
                                    padding: '2px 4px',
                                    borderRadius: '4px',
                                    fontFamily: 'monospace',
                                    color: 'var(--ghost-purple)'
                                }} {...props}>
                                    {children}
                                </code>
                            );
                        }

                        return !inline && match ? (
                            <SyntaxHighlighter
                                style={dracula}
                                language={match[1]}
                                PreTag="div"
                                {...props}
                            >
                                {String(children).replace(/\n$/, '')}
                            </SyntaxHighlighter>
                        ) : (
                            <code className={className} {...props}>
                                {children}
                            </code>
                        );
                    },
                    a({ node, children, href, ...props }) {
                        return (
                            <a
                                href={href}
                                onClick={(e) => {
                                    e.preventDefault();
                                    if (href) BrowserOpenURL(href);
                                }}
                                {...props}
                            >
                                {children}
                            </a>
                        );
                    },
                    // Remap paragraph to span in compact mode to ensure it stays inline
                    p: compact ? ({ children }) => <span style={{ display: 'inline' }}>{children}</span> : 'p'
                }}
            >
                {content}
            </ReactMarkdown>
        </div>
    );
};

export default MarkdownMessage;
