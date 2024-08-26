import React, { useState, useEffect, useRef } from 'react';

interface AudioProps {
  socket: WebSocket;
  conversationId: string;
  onUpdateStatus: (status: string) => void;
}

const AudioRecorder: React.FC<AudioProps> = ({ socket, conversationId,
  onUpdateStatus }) => {
  const [audioRecorder, setAudioRecorder] = useState<MediaRecorder |
    null>(null);
  const [recording, setRecording] = useState<boolean>(false);

  useEffect(() => {
    async function getMicrophone() {
      try {
        const stream = await navigator.mediaDevices.getUserMedia({
          audio: true
        });
        const mediaRecorder = new MediaRecorder(stream);

        // Set up event listeners
        mediaRecorder.ondataavailable = (event: BlobEvent) => {
          if (event.data.size > 0) {
            socket.send(event.data); // Send each chunk as it becomes available
            console.log('Chunk sent:', event.data.size);
          }
        };

        mediaRecorder.onstop = async () => {
          await handleStopRecording();
        };

        setAudioRecorder(mediaRecorder);
      } catch (error) {
        console.error('Error accessing media devices.', error);
      }
    }

    getMicrophone();
  }, []);

  const handleStopRecording = async () => {
    setRecording(false);
    onUpdateStatus("Processing speech to text...");
  };

  const startRecording = () => {
    audioRecorder?.start(1000);
    setRecording(true);
    onUpdateStatus('Recording started...');
  };

  const stopRecording = () => {
    audioRecorder?.stop();
    socket.send(JSON.stringify({ type: 'audioEnd', conversationId }));
  };


  // Keyboard event handlers for starting and stopping the audio recorder.
  useEffect(() => {
    const downHandler = (event: KeyboardEvent) => {
      if (event.key === ' ' && audioRecorder && !recording) {
        startRecording();
      }
    };

    const upHandler = (event: KeyboardEvent) => {
      if (event.key === ' ' && audioRecorder && recording) {
        stopRecording();
      }
    };

    window.addEventListener('keydown', downHandler);
    window.addEventListener('keyup', upHandler);

    return () => {
      window.removeEventListener('keydown', downHandler);
      window.removeEventListener('keyup', upHandler);
    };
  }, [audioRecorder, recording]);

  return <div>
    {/* Additional UI can go here */}
  </div>;
};

export default AudioRecorder;
