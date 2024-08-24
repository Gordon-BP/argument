import React, { useState, useEffect, useRef } from 'react';
import './App.css';
import { useTextStream } from "./text";
import AudioRecorder from './audio';

interface AppProps {
  socket: WebSocket;
}

const App: React.FC<AppProps> = ({ socket }) => {
  const [conversationId, setConversationId] = useState<string>('');
  const [status, setStatus] = useState<string>('Press and hold Space Bar to record');
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const { input, setInput, messages, currentBotMessage, currentUserMessage, handleSubmit } = useTextStream({
    socket,
    conversationId,
  });

  useEffect(() => {
    setConversationId('abc' + Math.round(Math.random() * 100));
  }, []);

  useEffect(() => {
    // Smooth scrolling
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, currentBotMessage, currentUserMessage]);

  return (
    <div className="chat-app">
      <h1>Chat App</h1>
      <p>Conversation ID: {conversationId}</p>
      <div className="chat-messages">
        {messages.map((message, index) => (
          <div key={index} className={`message ${message.isUser ? 'user' : 'bot'}`}>
            {message.text}
          </div>
        ))}
        {currentUserMessage && (
          <div className='message user'>
            {currentUserMessage}
          </div>
        )}
        {currentBotMessage && (
          <div className="message bot">
            {currentBotMessage}
          </div>
        )}
      </div>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          handleSubmit();
        }}
        className="chat-input"
      >
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Type a message..."
        />
        <button type="submit">Send</button>
      </form>
      <AudioRecorder onUpdateStatus={setStatus}
        conversationId={conversationId} socket={socket} />
      <p>{status}</p>
    </div>
  );
};

export default App;
