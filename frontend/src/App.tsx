import React, { useEffect, useState } from 'react';
import { format } from 'date-fns';
import { getPingResults } from './api';
import './App.css';

const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp);
    return format(date, 'yyyy-MM-dd HH:mm:ss'); // Пример формата
};

interface PingResult {
    host_name: string;
    ip: string;
    time: string;
    rtt: number; // ns
    success: boolean
}

const App: React.FC = () => {
    const [results, setResults] = useState<PingResult[]>([]);

    useEffect(() => {
        const fetchData = async () => {
            const result = await getPingResults();
            setResults(result.data.ping_results);
        };
        fetchData();
        setInterval(fetchData, 5000);
    }, []);

    return (
        <div>
            <h1>Docker Container Monitoring</h1>
            <table>
                <thead>
                    <tr>
                        <th>Host</th>
                        <th>IP</th>
                        <th>Rtt</th>
                        <th>Timestamp</th>
                    </tr>
                </thead>
                <tbody>
                    {results.map((result, index) => (
                        <tr key={index}>
                            <td>{result.host_name}</td>
                            <td>{result.ip}</td>
                            <td className="rtt">
                                {(result.rtt / 1e6).toLocaleString(undefined, {
                                    minimumFractionDigits: 3,
                                    maximumFractionDigits: 3,
                                })}
                                &nbsp;ms
                            </td>
                            <td className="timestamp">
                                {formatTimestamp(result.time)} {/* Форматируем timestamp */}
                            </td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
};

export default App;
