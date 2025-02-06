import React, { useEffect, useState } from 'react';
import { getPingResults } from './api';

interface PingResult {
    ip: string;
    ping_time: string;
    success: boolean;
}

const App: React.FC = () => {
    const [results, setResults] = useState<PingResult[]>([]);

    useEffect(() => {
        getPingResults().then(response => {
            setResults(response.data);
        });
    }, []);

    return (
        <div>
            <h1>Docker Container Monitoring</h1>
            <table>
                <thead>
                    <tr>
                        <th>IP Address</th>
                        <th>Ping Time</th>
                        <th>Last Successful Ping</th>
                    </tr>
                </thead>
                <tbody>
                    {results.map((result, index) => (
                        <tr key={index}>
                            <td>{result.ip}</td>
                            <td>{result.ping_time}</td>
                            <td>{result.success ? 'Yes' : 'No'}</td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
};

export default App;
