import React, { useState, useEffect, useRef } from 'react';
import './App.css';

interface Message {
  text: string;
  isUser: boolean;
}

interface AppProps {
  socket: WebSocket;
}

const CHUNK_DELAY = 20; // 0.02 seconds delay between chunks

const App: React.FC<AppProps> = ({ socket }) => {
  const [input, setInput] = useState('');
  const [messages, setMessages] = useState<Message[]>([]);
  const [currentBotMessage, setCurrentBotMessage] = useState('');
  const [incomingChunks, setIncomingChunks] = useState<string[]>([]);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const [conversationId, setConversationId] = useState<string>('');

  useEffect(() => {
    // Generate a new conversation ID when the component mounts
    setConversationId("abc" + Math.round(Math.random() * 100));
  }, []);

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (input.trim()) {
      setMessages(prev => [...prev, { text: input, isUser: true }]);

      // Send the message as a JSON string
      socket.send(JSON.stringify({
        conversationId: conversationId,
        text: input
      }));

      setInput('');
      setCurrentBotMessage('');
      setIncomingChunks([]);
    }
  };

  useEffect(() => {
    socket.onmessage = (event) => {
      try {
        const parsedMessage = JSON.parse(event.data);
        if (parsedMessage && typeof parsedMessage.text === 'string') {
          setIncomingChunks(prev => [...prev, parsedMessage.text]);
        }
      } catch (error) {
        console.error("Error parsing message:", error);
      }
    };

    return () => {
      socket.close();
    };
  }, [socket]);

  useEffect(() => {
    if (incomingChunks.length > 0) {
      const timer = setTimeout(() => {
        const chunk = incomingChunks[0];
        setCurrentBotMessage(prev => prev + chunk);
        setIncomingChunks(prev => prev.slice(1));
      }, CHUNK_DELAY);

      return () => clearTimeout(timer);
    }
  }, [incomingChunks, currentBotMessage]);

  useEffect(() => {
    if (currentBotMessage && incomingChunks.length === 0) {
      const timer = setTimeout(() => {
        setMessages(prev => [...prev, {
          text: currentBotMessage,
          isUser: false
        }]);
        setCurrentBotMessage('');
      }, CHUNK_DELAY);
      return () => clearTimeout(timer);
    }
  }, [currentBotMessage, incomingChunks]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, currentBotMessage]);

  const formatMessage = (text: string) => {
    return text.split('\n').map((line, index) => (
      <React.Fragment key={index}>
        {line}
        <br />
      </React.Fragment>
    ));
  };

  return (
    <div className="chat-app">
      <h1>Chat App</h1>
      <p>Conversation ID: {conversationId}</p>
      <div className="chat-messages">
        {messages.map((message, index) => (
          <div key={index} className={`message ${message.isUser ?
            'user' : 'bot'}`}>
            {formatMessage(message.text)}
          </div>
        ))}
        {currentBotMessage && (
          <div className="message bot">
            {formatMessage(currentBotMessage)}
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>
      <form onSubmit={handleSubmit} className="chat-input">
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Type a message..."
        />
        <button type="submit">Send</button>
      </form>
    </div>
  );
};

export default App;
