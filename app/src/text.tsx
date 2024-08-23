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

export const useTextStream = ({
  socket,
  conversationId,
  chunkDelay = 20, // Default chunk delay of 20 ms
}: UseTextStreamProps) => {
  const [input, setInput] = useState('');
  const [messages, setMessages] = useState<Message[]>([]);
  const [currentBotMessage, setCurrentBotMessage] = useState('');
  const [incomingChunks, setIncomingChunks] = useState<string[]>([]);

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
      setIncomingChunks([]);
    }
  };

  useEffect(() => {
    socket.onmessage = (event) => {
      try {
        const parsedMessage = JSON.parse(event.data);
        if (parsedMessage && typeof parsedMessage.text === 'string') {
          setIncomingChunks((prev) => [...prev, parsedMessage.text]);
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
        setCurrentBotMessage((prev) => prev + chunk);
        setIncomingChunks((prev) => prev.slice(1));
      }, chunkDelay);

      return () => clearTimeout(timer);
    }
  }, [incomingChunks, currentBotMessage, chunkDelay]);

  useEffect(() => {
    if (currentBotMessage && incomingChunks.length === 0) {
      const timer = setTimeout(() => {
        const newMessages = [
          ...messages,
          {
            text: currentBotMessage,
            isUser: false,
          },
        ];
        setMessages(newMessages);
        setCurrentBotMessage('');
      }, chunkDelay);
      return () => clearTimeout(timer);
    }
  }, [currentBotMessage, incomingChunks, messages, chunkDelay]);



  return {
    input,
    setInput,
    messages,
    currentBotMessage,
    handleSubmit,
  };
};
