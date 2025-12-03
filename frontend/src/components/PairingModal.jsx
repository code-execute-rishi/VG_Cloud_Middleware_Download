import React, { useState } from 'react';
import './PairingModal.css';
import { API_BASE_URL } from '../api/config';

const PairingModal = ({ onAuthenticated }) => {
    const [code, setCode] = useState('');
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);

    const handleSubmit = async (e) => {
        e.preventDefault();
        setError('');
        setLoading(true);

        try {
            const response = await fetch(`${API_BASE_URL}/api/claim-device`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ pair_code: code }),
            });

            const data = await response.json();

            if (!response.ok) {
                throw new Error(data.message || 'Authentication failed');
            }

            if (data.token) {
                onAuthenticated(data.token, data.url);
            } else {
                throw new Error('No token received');
            }
        } catch (err) {
            setError(err.message || 'Failed to connect to GCS');
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="pairing-modal-overlay">
            <div className="pairing-modal-container">
                <h2 className="pairing-header">System Access</h2>
                <form onSubmit={handleSubmit}>
                    <input
                        type="text"
                        className="pairing-input"
                        placeholder="ENTER ACCESS CODE"
                        value={code}
                        onChange={(e) => setCode(e.target.value)}
                        maxLength={9}
                        autoFocus
                    />
                    <button
                        type="submit"
                        className="pairing-button"
                        disabled={loading}
                    >
                        {loading ? 'AUTHENTICATING...' : 'INITIALIZE UPLINK'}
                    </button>
                    {error && <div className="pairing-error">ERROR: {error}</div>}
                </form>
            </div>
        </div>
    );
};

export default PairingModal;
