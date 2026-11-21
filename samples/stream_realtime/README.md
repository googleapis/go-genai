# Gemini Client in Go

This Go package provides a client for interacting with the Gemini **Live API**, enabling real-time multimodal interactions including text, audio, and video streaming. This client is inspired by the functionalities demonstrated in the Gemini 2.0 - Multimodal live API: Streaming, showcasing the ability to stream bidirectional audio and handle text-based conversations.

This client allows you to build applications that can:

*   Send and receive text messages.
*   Stream audio input to the Gemini API and receive audio responses.
*   Stream video input (from camera or screen capture) to the Gemini API.

## Features

*   **Real-time Communication:** Leverages WebSockets for low-latency, bidirectional communication with the Gemini Live API.
*   **Text Input:** Send textual prompts to the Gemini API for conversational interactions.
*   **Audio Input & Output:**
    *   Record audio from your microphone and stream it to the API.
    *   Receive and play back audio responses from the API in real-time.
*   **Video Input (Camera & Screen Capture):**
    *   Capture frames from your camera or screen and stream them to the API for multimodal prompts.
    *   Supports configuration to switch between camera and screen capture modes.
*   **Automatic Reconnection:** Attempts to reconnect to the API if the WebSocket connection is interrupted.
*   **Configurable:** Allows for customization of the Gemini model to be used.
*   **Error Handling & Logging:** Provides robust error handling and logging using `logrus`.

## Prerequisites

*   **Go:** Version 1.18 or higher.
*   **PortAudio:**  Required for audio input/output. Installation instructions vary by operating system.
    *   **Linux (Debian/Ubuntu):** `sudo apt-get install libportaudio2 libportaudiocpp0 portaudio19-dev`
*   **FFmpeg:** Required for camera capture and image conversion.
    *   You can usually install this using your system's package manager (e.g., `sudo apt-get install ffmpeg` on Debian/Ubuntu, `brew install ffmpeg` on macOS).
*   **GEMINI_API_KEY:** You need a valid API key from Google AI Studio, which you can obtain by signing up for the Gemini API.

## Installation

1. **Clone the repository:**
    ```bash
    git clone <repository_url>
    cd <repository_directory>
    ```

2. **Build the application:**
    ```bash
    go build -o gemini-client .
    ```

## Configuration

1. **Set the API Key:**
    You need to set the `GEMINI_API_KEY` environment variable. You can do this by running the following command:

    ```bash
    export GEMINI_API_KEY="YOUR_API_KEY"
    ```

    Replace `YOUR_API_KEY` with your actual Gemini API key.

## Usage

The client can be run in different modes by setting the `MODE` environment variable. This mirrors the different interaction patterns showcased in the.

### Output Mode

```go
setupMsg := map[string]interface{}{
		"setup": map[string]interface{}{
			"model": "models/gemini-2.0-flash-exp",
			"generationConfig": map[string]interface{}{
				"responseModalities": []string{"TEXT"}, // "TEXT", "AUDIO"
			},
		},
	}
```
The client defaults to text-based interaction, similar to the basic chat demonstrated in the "Live API - Websockets Quickstart" notebook. You can type messages in the terminal and send them to the Gemini API.

```bash
./gemini-client
```

You will be prompted with `message>: ` to enter your text.

### Audio Mode

To enable real-time bidirectional audio input and output, similar to the "Gemini 2.0 - Multimodal live API: Streaming", run the client without setting a specific responseModalities.  
```go
    "responseModalities": []string{"AUDIO"}, // "TEXT", "AUDIO"
```
The client will automatically start recording audio from your microphone and playing back audio responses.

```bash
./gemini-client
```

### Camera Mode

To send camera frames to the API, enabling multimodal prompts as explored in the examples, set the `MODE` environment variable to `camera`.

```bash
export MODE="camera"
./gemini-client
```

**Note:** Ensure your camera is properly configured and accessible by the system. The code currently targets `/dev/video1`. You might need to adjust this based on your camera setup.
```go
func (c *Client) openVideoCapture() (*exec.Cmd, error) {
	cap := exec.Command("ffmpeg", "-f", "v4l2", "-i", "/dev/video1", "-f", "rawvideo", "-pix_fmt", "rgb24", "-vframes", "1", "-")
	return cap, nil
}
```

### Screen Capture Mode

To send screen captures to the API, providing visual context to your prompts, set the `MODE` environment variable to `screen`.

```bash
export MODE="screen"
./gemini-client
```

The client will periodically capture your screen and send it to the API.

## Example Interaction

1. **Run the client (e.g., in text mode):**
    ```bash
    ./gemini-client
    ```
2. **You will see the prompt:**
    ```
    message>:
    ```
3. **Type your message and press Enter:**
    ```
    message>: What is the weather like today?
    ```
4. **The response from the Gemini API will be printed in the terminal.**

If running in audio or video modes, the client will continuously stream audio or video data to the API, enabling more dynamic and interactive conversations.

Refer to the Gemini API documentation for the available tools and their configurations.

## Future Improvements

1. **Windows Support:** Add support for Windows, including audio input/output and camera/screen capture.
2. **MacOS Support:** Add support for camera and screen capture on macOS.
3. **Tools Integrations:** Add support for Google Search, etc
4. **Function Calling**: Add support for calling functions in the Gemini API

## Contributing

Contributions are welcome! Please feel free to submit pull requests with improvements or bug fixes.

## Ported from

A Go-based implementation inspired by the functionality of [live_api_starter.ipynb](https://github.com/google-gemini/cookbook/blob/main/gemini-2/websockets/live_api_starter.ipynb) from [https://github.com/google-gemini/cookbook](https://github.com/google-gemini/cookbook).