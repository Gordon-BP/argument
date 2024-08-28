import { useState, useEffect, useRef } from 'react';

interface Message {
  text: string;
  isUser: boolean;
}

interface UseTextStreamProps {
  socket: WebSocket;
  conversationId: string;
  chunkDelay?: number;
  audioElement: React.RefObject<HTMLAudioElement>
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
  audioElement,
}: UseTextStreamProps) => {
  const [input, setInput] = useState('');
  const [messages, setMessages] = useState<Message[]>([]);
  const [currentBotMessage, setCurrentBotMessage] = useState('');
  const [currentUserMessage, setCurrentUserMessage] = useState('');
  const [incomingChunks, setIncomingChunks] = useState<IncomingChunk[]>([]);
  const [messageIndex, setMessageIndex] = useState(0); // Add index to track message changes
  const audioQueue = useRef<Blob[]>([]); // Queue for audio blobs

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
      setCurrentBotMessage(''); // Clear bot message after sending
      setIncomingChunks([]);
      setMessageIndex(messageIndex + 1); // Increment index after sending
    }
  };

  const playNextAudio = () => {
    if (audioQueue.current.length > 0) {
      const nextAudioBlob = audioQueue.current.shift(); // Get the next audio blob
      if (nextAudioBlob && audioElement.current) {
        const audioUrl = URL.createObjectURL(nextAudioBlob);
        audioElement.current.src = audioUrl;
        audioElement.current.play();
      }
    }
  };

  useEffect(() => {
    socket.onmessage = (event) => {
      console.log(`Received ${typeof event.data} packet of size ${event.data.size || event.data.length}`);
      if (event.data instanceof Blob) {
        console.log("Received audio data");
        event.data.arrayBuffer().then(buffer => {
          console.log("ArrayBuffer content:", new Uint8Array(buffer));
        }).catch(error => {
          console.error("Error reading Blob data:", error);
        });
        audioQueue.current.push(event.data); // Add incoming audio blob to the queue
        if (audioElement.current?.paused) {
          playNextAudio(); // Play immediately if not playing anything else
        }
      } else {
        try {
          const parsedMessage = JSON.parse(event.data);
          if (parsedMessage && typeof parsedMessage.content === 'string') {
            // Handle incoming chunks
            setIncomingChunks((prev) => [...prev, parsedMessage]);
          }
        } catch (error) {
          console.error('Error parsing message:', error);
        }
      }
    };

    if (audioElement.current) {
      audioElement.current.onended = playNextAudio; // Set up event listener for when the current audio finishes
    }

    return () => {
      socket.close();
    };
  }, [socket]);

  useEffect(() => {
    if (incomingChunks.length > 0) {
      const timer = setTimeout(() => {
        const chunk = incomingChunks[0];
        const isUserMessage = chunk.role === 'user';

        setMessages((prevMessages) => {
          const lastMessageIndex = prevMessages.length - 1;
          const lastMessage = prevMessages[lastMessageIndex];

          if (
            (isUserMessage && lastMessage && lastMessage.isUser) ||
            (!isUserMessage && lastMessage && !lastMessage.isUser)
          ) {
            // Update the last message if it's from the same role
            const updatedMessages = [...prevMessages];
            updatedMessages[lastMessageIndex] = {
              ...lastMessage,
              text: lastMessage.text + chunk.content,
            };
            return updatedMessages;
          } else {
            // Role changed, create a new message
            const newMessage = {
              text: chunk.content,
              isUser: isUserMessage,
            };
            setMessageIndex((prev) => prev + 1); // Increment index for new message
            return [...prevMessages, newMessage];
          }
        });

        setIncomingChunks((prev) => prev.slice(1));
      }, chunkDelay);

      return () => clearTimeout(timer);
    }
  }, [incomingChunks, chunkDelay, messageIndex]);

  return {
    input,
    setInput,
    messages,
    currentBotMessage,
    currentUserMessage,
    handleSubmit,
  };
};
