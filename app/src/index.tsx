import React from 'react';
import ReactDOM from 'react-dom';
import App from './App';

// Establish WebSocket connection to the Go server
const socket = new WebSocket('ws://localhost:8080/ws');

// Pass the WebSocket instance to the App component or use Context API
const AppWithSocket = () => <App socket={socket} />;

ReactDOM.render(
  <React.StrictMode>
    <AppWithSocket />
  </React.StrictMode>,
  document.getElementById('root')
);
