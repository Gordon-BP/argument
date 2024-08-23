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
#### Step 1: Sending and receiving text messages using REST API
Requires:
    1. Frontend that can accept text inputs
    2. Frontend that sends text messages to the server
    3. Server that forwards the messages to the LLM
    4. Server that streams the message from the LLM into the websocket
    5. Frontend that mutates a text element to display text as it comes in from the websocket
