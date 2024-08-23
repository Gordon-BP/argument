import React, { useState, useEffect } from 'react';

interface Message {
  text: string; // Message text to display
}

interface AppProps {
  socket: WebSocket;
}

const App: React.FC<AppProps> = ({ socket }) => {
  const [input, setInput] = useState('');
  const [fullMessage, setFullMessage] = useState(''); // State to hold the complete message text

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    console.log(`Sending message: ${input}`);
    socket.send(input); // Send the user input to the server
    setInput(''); // Clear the input field
  };

  useEffect(() => {
    socket.onmessage = (event) => {
      console.log(`Received from websocket: ${event.data}`);

      try {
        // Parse the JSON safely
        const parsedMessage = JSON.parse(event.data);

        // Check if the parsed object has a text property
        if (parsedMessage && typeof parsedMessage.text === 'string') {
          // Append the new text chunk to the full message
          setFullMessage((prevFullMessage) => prevFullMessage +
            parsedMessage.text); // Append to the growing message
        } else {
          console.error("Received message does not have the expected structure: ", parsedMessage);
        }
      } catch (error) {
        console.error("Error parsing message:", error); // Log the error if JSON parsing fails
      }
    };

    // Clean up function when component unmounts
    return () => {
      socket.close();
    };
  }, [socket]);

  const formatMessage = (message: string) => {
    return message.split('\n').map((line, index) => (
      <React.Fragment key={index}>
        {line}
        <br />
      </React.Fragment>
    ));
  };

  return (
    <div>
      <h1>Chat App</h1>
      <form onSubmit={handleSubmit}>
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Type a message..."
        />
        <button type="submit">Send</button>
      </form>
      <div style={{ whiteSpace: 'pre-wrap' }}> {/* CSS to preserve
      whitespace and line breaks */}
        {/* Render the formatted message here */}
        {formatMessage(fullMessage)}
      </div>
    </div>
  );
};

export default App;
