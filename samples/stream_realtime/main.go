package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	imageColor "image/color"
	"image/jpeg"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gordonklaus/portaudio"
	"github.com/gorilla/websocket"
	"github.com/hajimehoshi/oto"
	"github.com/kbinani/screenshot"
	"github.com/nfnt/resize"
	"github.com/sirupsen/logrus"
)

const (
	receiveSampleRate = 16000            // Output sample rate
	sendSampleRate    = 24000            // Input sample rate
	chunkSize         = 1024             // Input chunk size (number of int16 samples)
	channel           = 1                // Number of channels
	turnCompleteDelay = 10 * time.Second // Delay before sending turn_complete
	imageMaxWidth     = 1024
	imageMaxHeight    = 1024
	reconnectDelay    = 2 * time.Second // Delay between reconnect attempts
)

var APIKey string

func getAPIKey() (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}
	return apiKey, nil
}

type Client struct {
	conn             *websocket.Conn
	model            string
	tools            []map[string]interface{}
	audioPlayer      *oto.Player
	ctx              context.Context
	cancel           context.CancelFunc
	stream           *portaudio.Stream
	turnCompleteSent bool
	intBuf           []int16
	videoOutQueue    chan map[string]interface{}
	mu               sync.Mutex
	audioInQueue     chan []byte
	reconnectMutex   sync.Mutex // Mutex for reconnect logic
	reconnecting     bool       // Flag to indicate if a reconnect is in progress
	closeOnce        sync.Once
	closed           int32
	doneChan         chan struct{}
	wg               sync.WaitGroup
	paMutex          sync.Mutex
	voqMutex         sync.Mutex
}

func NewClient(model string, tools []map[string]interface{}) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	doneChan := make(chan struct{})
	intBuf := make([]int16, chunkSize)
	return &Client{
		model:            model,
		tools:            tools,
		ctx:              ctx,
		cancel:           cancel,
		turnCompleteSent: false,
		intBuf:           intBuf,
		videoOutQueue:    make(chan map[string]interface{}),
		audioInQueue:     make(chan []byte, 1000),
		doneChan:         doneChan,
		wg:               sync.WaitGroup{},
	}
}

func (c *Client) Connect(apiKey string) error {
	logrus.Infoln("Connecting to Gemini API...")
	c.reconnectMutex.Lock()
	if c.reconnecting {
		c.reconnectMutex.Unlock()
		return fmt.Errorf("reconnect already in progress")
	}
	c.reconnecting = true
	c.reconnectMutex.Unlock()

	defer func() {
		c.reconnectMutex.Lock()
		c.reconnecting = false
		c.reconnectMutex.Unlock()
	}()

	url := fmt.Sprintf("wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1alpha.GenerativeService.BidiGenerateContent?key=%s", apiKey)
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.DialContext(c.ctx, url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	c.conn = conn
	return nil
}

func (c *Client) Setup() error {
	setupMsg := map[string]interface{}{
		"setup": map[string]interface{}{
			"model": "models/gemini-2.0-flash-exp",
			"generationConfig": map[string]interface{}{
				"responseModalities": []string{"TEXT"},
			},
		},
	}
	err := c.SendMessage(setupMsg)
	if err != nil {
		return fmt.Errorf("write setup: %w", err)
	}

	_, _, err = c.conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read after setup: %w", err)
	}
	return nil
}

func (c *Client) SendMessage(msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	for retries := 0; retries < 3; retries++ {
		c.mu.Lock()
		err = c.conn.WriteMessage(websocket.TextMessage, data)
		c.mu.Unlock()
		if err == nil {
			return nil
		}

		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			logrus.Infoln("Connection closed, attempting to reconnect...")
			if reconnectErr := c.reconnect(); reconnectErr != nil {
				return fmt.Errorf("failed to reconnect: %w", reconnectErr)
			}

			continue
		}
		return err
	}
	return fmt.Errorf("failed to send message after retries: %w", err)
}

func (c *Client) reconnect() error {
	time.Sleep(reconnectDelay)
	err := c.Connect(APIKey)
	if err != nil {
		return err
	}
	if err := c.Setup(); err != nil {
		return err
	}
	return nil

}

// this is used to send text input
func (c *Client) SendTextInput(text string) error {
	message := map[string]interface{}{
		"client_content": map[string]interface{}{
			"turns": []map[string]interface{}{
				{
					"role": "user",
					"parts": []map[string]interface{}{
						{"text": text},
					},
				},
			},
			"turn_complete": true,
		},
	}
	return c.SendMessage(message)
}

func (c *Client) StartAudio() error {
	player, err := oto.NewContext(int(sendSampleRate), channel, 2, 8192)
	if err != nil {
		return fmt.Errorf("failed to create audio player: %w", err)
	}
	c.audioPlayer = player.NewPlayer()
	return nil
}

func (c *Client) SendAudio(audioData []byte) error {
	inputMsg := map[string]interface{}{
		"realtime_input": map[string]interface{}{
			"media_chunks": []map[string]interface{}{
				{
					"data":      base64.StdEncoding.EncodeToString(audioData),
					"mime_type": "audio/pcm",
				},
			},
		},
	}

	return c.SendMessage(inputMsg)
}

func (c *Client) SendVideo(videoData map[string]interface{}) error {
	inputMsg := map[string]interface{}{
		"realtime_input": map[string]interface{}{
			"media_chunks": []map[string]interface{}{videoData},
		},
	}
	return c.SendMessage(inputMsg)
}

func (c *Client) resizeImage(img *image.RGBA) *image.RGBA {
	imgResized := resize.Thumbnail(imageMaxWidth, imageMaxHeight, img, resize.Lanczos3).(*image.RGBA)
	return imgResized
}

func (c *Client) encodeImageToJPEG(img *image.RGBA) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := jpeg.Encode(buf, img, nil)
	if err != nil {
		return nil, fmt.Errorf("encodeImageToJPEG failed %w", err)
	}
	return buf.Bytes(), nil
}

// ProcessScreenFrames reads frames from the screen and sends them to the server
// until doneChan is closed.
// and in the end subtracts the wait group
// TODO use atomic.LoadInt32(&c.closed) instead of defalt select
func (c *Client) processScreenFrames() error {
	for {
		select {
		case <-c.doneChan:
			logrus.Debugln("doneChan closed processScreenFrames")
			return nil
		default:
			frame, err := c._getScreen()
			if err != nil {
				logrus.Errorln("Failed to process screen frame:", err)
				continue
			}
			select {
			case <-c.doneChan:
				logrus.Debugln("doneChan closed processScreenFrames inside select")
				return nil
			case c.videoOutQueue <- frame:
				time.Sleep(1 * time.Second)
			default:
				time.Sleep(10 * time.Millisecond)
			}
		}
	}
}

// _getScreen captures the screen and returns the image as a map
func (c *Client) _getScreen() (map[string]interface{}, error) {
	n := screenshot.NumActiveDisplays()
	if n <= 0 {
		return nil, fmt.Errorf("no monitors detected")
	}
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, fmt.Errorf("failed to capture screen: %w", err)
	}
	imgResized := c.resizeImage(img)
	imageBytes, err := c.encodeImageToJPEG(imgResized)
	if err != nil {
		return nil, fmt.Errorf("failed to encode image to JPEG: %w", err)
	}

	return map[string]interface{}{
		"mime_type": "image/jpeg",
		"data":      base64.StdEncoding.EncodeToString(imageBytes),
	}, nil
}

// _getFrame captures a frame from the camera and returns the image as a map
func (c *Client) _getFrame() (map[string]interface{}, error) {
	cap, stderr, err := c.openVideoCapture()
	if err != nil {
		return nil, fmt.Errorf("failed to open video capture %w", err)
	}
	defer func() {
		if cap.Process != nil {
			cap.Process.Release()
		}
	}()
	data, err := cap.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read frame %w, %s", err, stderr.String())
	}
	frame_rgb := c.convertToRGB(data)
	img, err := c.createImageFromBytes(frame_rgb)
	if err != nil {
		return nil, fmt.Errorf("failed to create image from bytes %w", err)
	}
	img = c.resizeImage(img)
	imageBytes, err := c.encodeImageToJPEG(img)
	if err != nil {
		return nil, fmt.Errorf("failed to encode image to JPEG %w", err)
	}
	return map[string]interface{}{
		"mime_type": "image/jpeg",
		"data":      base64.StdEncoding.EncodeToString(imageBytes),
	}, nil
}

func (c *Client) openVideoCapture() (*exec.Cmd, *bytes.Buffer, error) {
	cap := exec.Command("ffmpeg", "-f", "v4l2", "-i", "/dev/video1", "-f", "rawvideo", "-pix_fmt", "rgb24", "-vframes", "1", "-")
	var stderr bytes.Buffer
	cap.Stderr = &stderr
	return cap, &stderr, nil
}

func (c *Client) convertToRGB(frame []byte) []byte {
	var out bytes.Buffer
	cmd := exec.Command("ffmpeg", "-f", "rawvideo", "-pix_fmt", "bgr24", "-video_size", "640x480", "-i", "-", "-f", "rawvideo", "-pix_fmt", "rgb24", "-")
	cmd.Stdin = bytes.NewReader(frame)
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		logrus.Errorln("Error converting image to rgb:", err)
	}
	return out.Bytes()
}

func (c *Client) createImageFromBytes(frame_rgb []byte) (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, 640, 480))
	for i := 0; i < len(frame_rgb); i += 3 {
		x := (i / 3) % 640
		y := (i / 3) / 640
		img.Set(x, y, imageColor.RGBA{R: frame_rgb[i], G: frame_rgb[i+1], B: frame_rgb[i+2], A: 0xff})
	}
	return img, nil
}

func (c *Client) StartRecording() error {
	portaudio.Initialize()
	stream, err := portaudio.OpenDefaultStream(channel, 0, float64(sendSampleRate), len(c.intBuf), c.intBuf)
	if err != nil {
		return fmt.Errorf("failed to open default stream: %w", err)
	}
	c.stream = stream
	if err := c.stream.Start(); err != nil {
		return fmt.Errorf("failed to start recording: %w", err)
	}
	return nil
}

// ProcessCameraFrames reads frames from the camera and sends them to the server
// until doneChan is closed.
// and in the end subtracts the wait group
// TODO use atomic.LoadInt32(&c.closed) instead of defalt select
func (c *Client) processCameraFrames() error {
	for {
		select {
		case <-c.doneChan:
			logrus.Debugln("doneChan closed processCameraFrames")
			return nil
		default:
			frame, err := c._getFrame()
			if err != nil {
				logrus.Errorln("Failed to process camera frame:", err)
				return err
			}
			select {
			case <-c.doneChan:
				logrus.Debugln("doneChan closed processCameraFrames inside select")
				return nil
			case c.videoOutQueue <- frame:
				time.Sleep(1 * time.Second)
			default:
				time.Sleep(10 * time.Millisecond)
			}
		}
	}
}

// ProcessAudioInput reads audio from the microphone and sends it to the server
// until doneChan is closed.
// and in the end subtracts the wait group
func (c *Client) ProcessAudioInput() error {
	for {
		select {
		case <-c.doneChan:
			logrus.Debugln("doneChan closed ProcessAudioInput")
			return nil
		default:
			if atomic.LoadInt32(&c.closed) == 1 {
				logrus.Debugln("Client is closing, exiting ProcessAudioInput")
				return nil
			}
			c.paMutex.Lock()
			err := c.stream.Read()
			c.paMutex.Unlock()

			if err != nil {
				if err == io.EOF {
					return nil
				} else if portaudioErr, ok := err.(portaudio.UnanticipatedHostError); ok {
					if portaudioErr.Code == 0 {
						logrus.Debugln("channel closed")
						return nil
					}
				}
				return fmt.Errorf("failed to read audio: %w", err)
			}
			if err := c.SendAudio(c.int16ToLittleEndianByte(c.intBuf)); err != nil {
				// return fmt.Errorf("failed to send audio: %w", err)
				// I am not returning here because if if there is error to send to websocket
				// I will return and stop the input microphone processing
				// so i will just log the error and continue
				logrus.Infof("failed to send audio: %v", err)
				continue
			}
		}
	}
}

func (c *Client) int16ToLittleEndianByte(f []int16) []byte {
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.LittleEndian, f)
	if err != nil {
		logrus.Errorf("binary.Write failed. Err %v\n", err)
	}
	return buf.Bytes()
}

// ProcessAudio reads audio from the server and plays it until doneChan is closed.
// and in the end subtracts the wait group
func (c *Client) ProcessAudio() error {
	time.Sleep(1 * time.Second)

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseInternalServerErr) {
				if reconnectErr := c.reconnect(); reconnectErr != nil {
					return fmt.Errorf("failed to reconnect: %w", reconnectErr)
				}
				continue
			}
			return fmt.Errorf("failed to read message: %w", err)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(message, &response); err != nil {
			logrus.Errorln("Error unmarshaling:", err)
			continue
		}

		serverContent, ok := response["serverContent"].(map[string]interface{})
		if !ok {
			logrus.Debugln("No serverContent found in response:", response)
			continue
		}

		modelTurn, ok := serverContent["modelTurn"].(map[string]interface{})
		if !ok {
			turnComplete, ok := serverContent["turnComplete"].(bool)
			if ok && turnComplete {
				for len(c.audioInQueue) > 0 {
					<-c.audioInQueue
				}
				continue
			}
			logrus.Debugln("No modelTurn found in response:", serverContent)
			continue
		}
		parts, ok := modelTurn["parts"].([]interface{})
		if !ok || len(parts) == 0 {
			logrus.Debugln("No parts found in modelTurn:", modelTurn)
			continue
		}

		for _, part := range parts {
			partMap, ok := part.(map[string]interface{})
			if !ok {
				continue
			}
			if text, ok := partMap["text"].(string); ok {
				fmt.Print(text)
			} else if inlineData, ok := partMap["inlineData"].(map[string]interface{}); ok {
				if audioData, ok := inlineData["data"].(string); ok {
					decoded, err := base64.StdEncoding.DecodeString(audioData)
					if err != nil {
						logrus.Errorln("Error decoding audio:", err)
						continue
					}
					select {
					case <-c.doneChan:
						logrus.Debugln("doneChan closed ProcessAudio")
						return nil
					case c.audioInQueue <- decoded:
					default:
					}
				}
			}
		}
		fmt.Println()
	}
}

// ProcessAudioOutput reads audio from the `audioInQueue` and plays it until doneChan is closed.
// and in the end subtracts the wait group
func (c *Client) ProcessAudioOutput() {
	for {
		select {
		case <-c.doneChan:
			logrus.Debugln("doneChan closed ProcessAudioOutput")
			return
		case audio := <-c.audioInQueue:
			// time.Sleep(100 * time.Millisecond)
			if _, err := c.audioPlayer.Write(audio); err != nil {
				logrus.Errorln("Error playing audio:", err)
			}
		}
	}
}

// ProcessVideoSend reads video from the `videoOutQueue` and sends it to the server until doneChan is closed.
// and in the end subtracts the wait group
func (c *Client) ProcessVideoSend() {
	for {
		select {
		case <-c.doneChan:
			logrus.Debugln("doneChan closed ProcessVideoSend")
			return
		case video := <-c.videoOutQueue:
			if err := c.SendVideo(video); err != nil {
				logrus.Errorln("Error sending video:", err)
			}
		}

	}
}

func (c *Client) Close() {
	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		c.closeOnce.Do(func() {

			close(c.doneChan)

			close(c.audioInQueue)
			c.cancel()
			if c.conn != nil {
				err := c.conn.Close()
				if err != nil {
					logrus.Errorln("Error closing connection:", err)
				}
			}
			close(c.videoOutQueue)
			c.paMutex.Lock()
			if c.stream != nil {
				err := c.stream.Stop()
				if err != nil {
					logrus.Errorln("Error stopping stream:", err)
				}
				err = c.stream.Close()
				if err != nil {
					logrus.Errorln("Error closing stream:", err)
				}
			}
			err := portaudio.Terminate()
			c.paMutex.Unlock()
			if err != nil {
				logrus.Errorln("Error terminating portaudio:", err)
			}
			// c.audioPlayer.Close()
		})
	}

}

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	logrus.SetReportCaller(true)
	logrus.Infoln("Starting Gemini client")

	var err error
	APIKey, err = getAPIKey()
	if err != nil {
		logrus.Fatalln("Error retrieving API key:", err)
	}
	tools := []map[string]interface{}{
		// {"google_search": map[string]interface{}{}},
	}
	client := NewClient("models/gemini-2.0-flash-exp", tools)

	if err := client.Connect(APIKey); err != nil {
		logrus.Fatalln("Connection error:", err)
	}
	if err := client.Setup(); err != nil {
		logrus.Fatalln("Setup error:", err)
	}
	if err := client.StartAudio(); err != nil {
		logrus.Fatalln("Error starting audio:", err)
	}
	if err := client.StartRecording(); err != nil {
		logrus.Fatalln("Error starting microphone:", err)
	}
	client.wg.Add(1)
	go func() {
		defer func() {
			client.wg.Done()
			logrus.Debugln("done with ProcessAudioInput")
		}()
		if err := client.ProcessAudioInput(); err != nil {
			logrus.Errorln("Error processing audio input:", err)
		}
	}()

	client.wg.Add(1)
	go func() {
		defer func() {
			client.wg.Done()
			logrus.Debugln("done with ProcessAudio")
		}()
		if err := client.ProcessAudio(); err != nil {
			logrus.Errorln("Audio processing error:", err)
			client.Close()
		}
	}()

	client.wg.Add(1)
	go func() {
		defer func() {
			client.wg.Done()
			logrus.Debugln("done with ProcessAudioOutput")
		}()

		client.ProcessAudioOutput()
	}()

	client.wg.Add(1)
	go func() {
		defer func() {
			client.wg.Done()
			logrus.Debugln("done with ProcessVideoSend")
		}()
		client.ProcessVideoSend()
	}()

	client.wg.Add(1)
	go func() {
		defer func() {
			client.wg.Done()
			logrus.Debugln("done with Process Camera | Frames")
		}()
		if os.Getenv("MODE") == "" {
			logrus.Infoln("No mode specified")
		} else if os.Getenv("MODE") == "camera" {
			logrus.Infoln("camera is selected")
			if err := client.processCameraFrames(); err != nil {
				logrus.Errorln("Camera processing error:", err)
				client.Close()
			}
		} else if os.Getenv("MODE") == "screen" {
			logrus.Infoln("screen is selected")
			time.Sleep(1 * time.Second)
			if err := client.processScreenFrames(); err != nil {
				logrus.Errorln("Screen processing error:", err)
			}
		}
	}()

	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Printf("message>: ")
			input, err := reader.ReadString('\n')
			if err != nil {
				logrus.Errorln("Error reading input:", err)
				return
			}

			input = strings.TrimSpace(input)
			if err := client.SendTextInput(input); err != nil {
				logrus.Errorln("Error sending input:", err)
				return
			}
		}
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-interrupt
		fmt.Println("Intentent Exiting...")
		client.Close()
	}()
	client.wg.Wait()
	err = client.audioPlayer.Close()
	if err != nil {
		logrus.Errorln("Error closing audio player:", err)
	}

	time.Sleep(2 * time.Second)
}
