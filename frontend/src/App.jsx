import React, { useState, useEffect } from 'react';
import axios from 'axios';

export default function App() {
  const [file, setFile] = useState(null);
  const [scores, setScores] = useState([]);
  const [status, setStatus] = useState('Ready');
  const [error, setError] = useState('');

  const gatewayUrl = 'http://localhost:8080';

  const fetchScores = async () => {
    try {
      const response = await axios.get(`${gatewayUrl}/scores`);
      setScores(response.data.scores || []);
    } catch (err) {
      setError('Could not fetch scores.');
      console.error(err);
    }
  };

  useEffect(() => {
    fetchScores();
  }, []);

  const handleFileChange = (e) => {
    setFile(e.target.files[0]);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!file) {
      setError('Please select a file first.');
      return;
    }

    const formData = new FormData();
    formData.append('screenshot', file);

    setStatus('Uploading and processing...');
    setError('');

    try {
      await axios.post(`${gatewayUrl}/upload`, formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      });
      setStatus('Upload successful! Refreshing scores...');
      await fetchScores(); // Refresh scores after upload
      setStatus('Ready');
    } catch (err) {
      setStatus('Ready');
      setError('An error occurred during upload.');
      console.error(err);
    }
  };

  return (
    <div className="min-h-screen bg-gray-900 text-white font-sans p-8">
      <div className="container mx-auto max-w-4xl">
        <header className="text-center mb-12">
          <h1 className="text-4xl font-bold text-cyan-400">
            Voltaic Score Tracker
          </h1>
          <p className="text-gray-400 mt-2">Full-Stack Go Edition</p>
        </header>

        <div className="bg-gray-800 rounded-xl shadow-lg p-6 mb-8">
          <h2 className="text-2xl font-semibold mb-4">
            Process Scores from Image
          </h2>
          <form onSubmit={handleSubmit}>
            <input
              type="file"
              onChange={handleFileChange}
              className="w-full text-sm text-gray-400 file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-sm file:font-semibold file:bg-cyan-500 file:text-white hover:file:bg-cyan-600"
            />
            <button
              type="submit"
              className="mt-4 w-full bg-green-500 hover:bg-green-600 text-white font-bold py-2.5 px-4 rounded-lg transition-colors"
            >
              Process Image
            </button>
          </form>
          <p className="text-sm text-gray-400 mt-3">Status: {status}</p>
          {error && <p className="text-sm text-red-500 mt-2">{error}</p>}
        </div>

        <div className="bg-gray-800 rounded-xl shadow-lg overflow-hidden">
          <div className="p-6">
            <h2 className="text-2xl font-semibold text-white">Score History</h2>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm text-left text-gray-300">
              <thead className="text-xs text-cyan-300 uppercase bg-gray-700">
                <tr>
                  <th scope="col" className="px-6 py-3">
                    Scenario
                  </th>
                  <th scope="col" className="px-6 py-3">
                    Score
                  </th>
                  <th scope="col" className="px-6 py-3">
                    Processed At
                  </th>
                </tr>
              </thead>
              <tbody>
                {scores.map((score) => (
                  <tr
                    key={score.id}
                    className="bg-gray-800 border-b border-gray-700 hover:bg-gray-600"
                  >
                    <td className="px-6 py-4 font-medium text-white whitespace-nowrap">
                      {score.scenario}
                    </td>
                    <td className="px-6 py-4">{score.score}</td>
                    <td className="px-6 py-4">
                      {new Date(score.processed_at).toLocaleString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
}
