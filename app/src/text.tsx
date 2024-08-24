import { useState, useEffect } from 'react';

interface Message {
  text: string;
  isUser: boolean;
}

interface UseTextStreamProps {
  socket: WebSocket;
  conversationId: string;
  chunkDelay?: number;
}

type IncomingChunk = {
  content: string;
  role: string;
  name: string;
};

export const useTextStream = ({
  socket,
  conversationId,
  chunkDelay = 20, // Default chunk delay of 20 ms
}: UseTextStreamProps) => {
  const [input, setInput] = useState('');
  const [messages, setMessages] = useState<Message[]>([]);
  const [currentBotMessage, setCurrentBotMessage] = useState('');
  const [currentUserMessage, setCurrentUserMessage] = useState('');
  const [incomingChunks, setIncomingChunks] = useState<IncomingChunk[]>([]);

  const handleSubmit = () => {
    if (input.trim()) {
      const newMessages = [...messages, { text: input, isUser: true }];
      setMessages(newMessages);

      // Send the message as a JSON string
      socket.send(
        JSON.stringify({
          conversationId: conversationId,
          text: input,
        })
      );

      setInput('');
      setCurrentBotMessage('');
      setCurrentUserMessage(''); // Clear user message after sending
      setIncomingChunks([]);
    }
  };

  useEffect(() => {
    socket.onmessage = (event) => {
      try {
        const parsedMessage = JSON.parse(event.data);
        console.log(parsedMessage);

        if (parsedMessage && typeof parsedMessage.content === 'string') {
          // Handle incoming chunks
          setIncomingChunks((prev) => [...prev, parsedMessage]);
        }
      } catch (error) {
        console.error('Error parsing message:', error);
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

        setMessages((prevMessages) => {
          const lastMessageIndex = prevMessages.length - 1;
          const lastMessage = prevMessages[lastMessageIndex];

          if (chunk.role === 'user') {
            console.log(`User message: ${chunk.content}`);

            if (lastMessage && lastMessage.isUser) {
              // Update the last user message if it exists
              const updatedMessages = [...prevMessages];
              updatedMessages[lastMessageIndex] = {
                ...lastMessage,
                text: lastMessage.text + chunk.content,
              };
              return updatedMessages;
            } else {
              // Add a new user message if one doesn't exist
              return [...prevMessages, { text: chunk.content, isUser: true }];
            }
          } else if (chunk.role === 'bot') {
            console.log(`Bot message: ${chunk.content}`);
            const updatedBotMessage = currentBotMessage + ' ' + chunk.content;

            setCurrentBotMessage(updatedBotMessage);

          }

          return prevMessages;
        });

        setIncomingChunks((prev) => prev.slice(1));
      }, chunkDelay);

      return () => clearTimeout(timer);
    }
  }, [incomingChunks, chunkDelay, currentBotMessage]);
  return {
    input,
    setInput,
    messages,
    currentBotMessage,
    currentUserMessage, // Include currentUserMessage in the return object
    handleSubmit,
  };
};
