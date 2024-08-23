# argument
entertaining chatbot that has a monty python argument with you

# Brainstorming:
* Bot will live on my website, at argument.hanakano.com
* Users will interact with the bot via voice
* English language only
* Bot will communicate in 3 simultaneous modalities:
    1. The bot's messages will appear as text on screen, in a scrollable conversation history
    2. The bot's most recent reply will be read aloud by text to speech
    3. A persistent artifact pane will show content as relevant to the conversation, such as:
        * Wikipedia articles
        * Images
        * YouTube videos
* Bot replies should come in real-time, with the smallest possible latency
* Users should be able to interrupt the bot
* The UI should be decorated to look like a therapist clinic
* When users interact with the different UI elements, the bot reacts:
    * checking sources
    * reviewing transcript
    * resetting the timer
* Bot can interrupt the user
* Would love to have a special case for fast contradiction. If user says "No it's not" the bot immediately goes "yes it is" no matter what
* One of the bot personalities becomes very juvenile when they are losing and resort to name calling and utter ridiculousness
* Can have a list of "starter topics" for new users. "Is a hot dog a sandwich?" "Which position is best to sleep in?" "Do humans deserve to live on Earth?" as a non-sequitur joke
* Bot can react to user actions on the webpage, such as clicking on images, scrolling through the article, or clicking on other UI elements
* There is a 5 minute timer on the conversation and the bot is aware of how much time is available
* After the 5 minutes is up, the bot refuses to argue
* We can leverage two different LLMs:
    * Groq llama 3 for fast planning and retrieval
    * Claude sonnet for synthesizing a good response
* Ideally everything would be streamed
    * streaming user audio to transcription
    * streaming transcribed text to a looping planning llm
    * not streaming the planning llm's instructions to the synthesizing llm
    * streaming text from the synthesizing llm to the speech llm
    * streaming audio from the speech llm back to the user
* Planning llm does polling on a 1 - 2 second cycle. Basically rewrites its plan and retrieves sources with every 2 seconds of user audio. Then the final plan is made by merging them all. Is this efficient?
* Serverless hosting could be nice because it scales to zero. VM hosting could be nice because there is a limit to how much usage. But since it relies on so many 3rd party services, cost should be considered.

# Tools to use
* [Groq llama 3 8B](https://groq.com/) - The latency bottleneck will be time to first token. Groq llama 3.1 8B balances super fast inference with reasoning capabilities. I plan on using this model to plan responses and retrieve sources during and after the user's voice message.
* [Claude 3.5 Sonnet](https://www.anthropic.com/news/claude-3-5-sonnet) - This could be experimental and too complicated, but I think it would be good to use a more sophisticated model to execute the plan and actually generate the response.
* [ElevenLabs TTS](https://elevenlabs.io/text-to-speech) - Elevenlabs has excellent AI voices and low latency.
* [Daily / WebRTC](https://www.daily.co/) - For maximum speed we need to establish a constant connection and make it less like API calls and more like a vide chat.
* [Deepgram Transcriptions](https://developers.deepgram.com/docs/getting-started-with-live-streaming-audio) - Deepgram has a lot of docs and starter apps for real time transcription and they also collaborate with Daily.
* It looks like deepgram also has streamed text to speech available too?

# Simplify for a POC
## AI Agent
* Start with only one personality
* Focus on having an argument using only what the model already knows + prompt engineering

## Backend
* Use a starter or pre built app instead of building your own UI
* Experiment to see if WebRTC _really_ is better than normal webhooks and REST API
* I really, _really_ want to use Golang but that adds unnecessary complexity

## Tools
* Do everything in NodeJS, that seems to be the common language that all libraries and SDKs support
* Groq Llama 3.1 8B as the planning and synthesis model
    * It's fast and cheap
    * Yes it might not have as solid reasoning capabilities but this is an entertainment app
* Daily for the pre built app and webRTC 
* Deepgram for streamed transcription and streamed text-to-speech


# Let's break the POC up into specific milestones and user stories

### Feature: Users can send messages to an LLM, and receive replies
#### Step 1: Sending and receiving text messages using websockets
Requires:
    1. Frontend that can accept text inputs
    2. Frontend that sends text messages to the server
    3. Server that forwards the messages to the LLM
    4. Server that streams the message from the LLM into the websocket
    5. Frontend that mutates a text element to display text as it comes in from the websocket
DONE! Thanks to some helpful LLMs I was able to build a simple react interface, and use a Go backend to stream text from Groq to the interface.
Some things I learned:
* Websockets are both more simple and more complicated than I thought. They're just chunks of data that come in a sequence, how cool! But also, that means that ou have to parse them as they come, and all the responsibility for that is on you. This is great when you want control, but also means that you have more code to maintain. Luckily, sticking to JSON makes this easier.
* Groq is fast! With their llama 3 8b "instant" model, even though I was streaming the output it would take less than 1 second to come through. This made it look like the content wasn't even being streamed, but instead jsut coming through in batch like a traditional REST API. I had to add an artificial wait time between chunks to make the streaming effect more obvious.
* A little bit of CSS goes a long way. Early on I had GPT upgrade my basic form into a proper chat interface, and that made everything so much easier.
* Go _can_ power a react app! And it was honestly quite simple. I'm still learning Go so I relied heavily on LLMs for this. But I found that they were able to take instruction well and product workable code. Will me not knowing Go very well come back and bite me later in the project? Probably! But right now I'm really enjoying it.

#### Step 2: Sending and receiving voice messages using websockets
Requires:
    1. Frontend that can accept audio input
    2. Frontend that streams audio to the server
    3. Server that streams audio to transcription service
    4. Server that sends completed user transcription to an LLM
    5. Server that sends LLM reply chunks to a text to speech service
    7. Server that streams audio from text to speech service to the frontend
    8. Frontend that plays audio from a stream

TIL that there is no such thing as streaming TTS. Instead, you have to chunk your text by the sentence and send each chunk to the TTS service.
TIL also that you can run a small whisper model _entirely in your web browser_. This is cool!
##### But how do you detect when to stop transcribing and send the text?
Simple solution: Don't bother with pause detection!
    * Transcribe text straight into the text input box, and then make the user click send
    * Make the user click a microphone button to start, and then click again to stop
    * Make the user hold down space bar to talk and submit the audio when it is let up
I'm sure there are more sophisticated methods out there but honestly I think the spacebar method is best. 
PLUS I already have react code written for that!

So for the spec, I'm thinking:
    1. Use deepgram for TTS and STT. They have a streaming STT API with a golang SDK
    2. Make the user press and hold space bar while they talk.
    3. (Optional but would be really cool) Stream the transcription into the UI somehow
    4. Break up llm response by the sentence, and send each sentence to deepgram.
    5. Try to stream directly from the TTS to the frontend, but you'll probably have to set up a buffer file somewhere in case the TTS is fast.
