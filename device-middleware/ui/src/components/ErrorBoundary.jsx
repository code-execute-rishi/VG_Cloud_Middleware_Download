import React from 'react';

class ErrorBoundary extends React.Component {
    constructor(props) {
        super(props);
        this.state = { hasError: false, error: null, errorInfo: null };
    }

    static getDerivedStateFromError(error) {
        return { hasError: true };
    }

    componentDidCatch(error, errorInfo) {
        console.error("Uncaught error:", error, errorInfo);
        this.setState({ error, errorInfo });
    }

    render() {
        if (this.state.hasError) {
            return (
                <div style={{ padding: '2rem', color: '#dc2626', background: '#fee2e2', height: '100vh', overflow: 'auto' }}>
                    <h1>⚠️ Something went wrong.</h1>
                    <p>Please report this error to support.</p>
                    <details style={{ whiteSpace: 'pre-wrap', marginTop: '1rem', padding: '1rem', background: 'white', borderRadius: '8px', border: '1px solid #fecaca' }}>
                        <summary style={{ cursor: 'pointer', fontWeight: 'bold' }}>View Error Details</summary>
                        <div style={{ marginTop: '1rem', fontFamily: 'monospace', fontSize: '0.85rem' }}>
                            <strong>{this.state.error && this.state.error.toString()}</strong>
                            <br />
                            {this.state.errorInfo && this.state.errorInfo.componentStack}
                        </div>
                    </details>
                    <button
                        onClick={() => window.location.reload()}
                        style={{ marginTop: '2rem', padding: '10px 20px', background: '#dc2626', color: 'white', border: 'none', borderRadius: '6px', cursor: 'pointer' }}
                    >
                        Reload Application
                    </button>
                </div>
            );
        }

        return this.props.children;
    }
}

export default ErrorBoundary;
