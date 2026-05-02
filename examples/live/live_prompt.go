// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build ignore_vet

package main

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"google.golang.org/genai"
)

var voiceSample = flag.String("voice-sample", "", "Path to voice sample file")
var voiceConsent = flag.String("voice-consent", "", "Path to voice consent file")
var voiceSignature = flag.String("voice-signature", "", "Voice consent signature")
var modelFlag = flag.String("model", "", "Model name")
var promptFlag = flag.String("prompt", "Hello Gemini, are you there?", "Text prompt for testing")

func main() {
	flag.Parse()
	log.SetFlags(0)

	if *promptFlag == "" {
		log.Fatal("--prompt must be specified")
	}

	var voiceSampleAudio []byte
	var consentAudio []byte

	if *voiceSample != "" {
		var err error
		voiceSampleAudio, err = os.ReadFile(*voiceSample)
		if err != nil {
			log.Fatal("read voice sample error: ", err)
		}
		if *voiceConsent != "" {
			consentAudio, err = os.ReadFile(*voiceConsent)
			if err != nil {
				log.Fatal("read voice consent error: ", err)
			}
		}
		if len(consentAudio) == 0 && *voiceSignature == "" {
			log.Fatal("Either --voice-consent or --voice-signature must be provided when --voice-sample is used.")
		}
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal("create client error: ", err)
	}

	var model string
	if *modelFlag != "" {
		model = *modelFlag
	} else if client.ClientConfig().Backend == genai.BackendVertexAI {
		model = "gemini-2.0-flash-live-preview-04-09"
	} else {
		model = "gemini-live-2.5-flash-preview"
	}

	config := &genai.LiveConnectConfig{}
	config.ResponseModalities = []genai.Modality{genai.ModalityAudio}

	if len(voiceSampleAudio) > 0 {
		replicatedConfig := &genai.ReplicatedVoiceConfig{
			MIMEType:         "audio/wav",
			VoiceSampleAudio: voiceSampleAudio,
		}
		if len(consentAudio) > 0 {
			replicatedConfig.ConsentAudio = consentAudio
		}
		if *voiceSignature != "" {
			replicatedConfig.VoiceConsentSignature = &genai.VoiceConsentSignature{
				Signature: *voiceSignature,
			}
		}
		config.SpeechConfig = &genai.SpeechConfig{
			VoiceConfig: &genai.VoiceConfig{
				ReplicatedVoiceConfig: replicatedConfig,
			},
		}
	}

	session, err := client.Live.Connect(ctx, model, config)
	if err != nil {
		log.Fatal("connect to model error: ", err)
	}
	defer session.Close()

	// Read SetupComplete
	setupMsg, err := session.Receive()
	if err != nil {
		log.Fatal("receive setup complete error: ", err)
	}
	if setupMsg.SetupComplete != nil && setupMsg.SetupComplete.VoiceConsentSignature != nil {
		log.Printf("\n=== Voice Consent Signature Received ===\n%s\n========================================\n", setupMsg.SetupComplete.VoiceConsentSignature.Signature)
	}

	fmt.Println("Sending prompt:", *promptFlag)
	err = session.SendRealtimeInput(genai.LiveRealtimeInput{
		Text: *promptFlag,
	})
	if err != nil {
		log.Fatal("send prompt error: ", err)
	}

	var audioData []byte
	for {
		msg, err := session.Receive()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("receive error: ", err)
		}

		if msg.ServerContent != nil {
			content := msg.ServerContent
			if content.TurnComplete {
				break
			}
			if content.ModelTurn != nil {
				for _, part := range content.ModelTurn.Parts {
					if part.InlineData != nil && part.InlineData.Data != nil {
						audioData = append(audioData, part.InlineData.Data...)
						fmt.Printf("Received audio chunk: %d bytes\n", len(part.InlineData.Data))
					}
				}
			}
		}
	}

	if len(audioData) > 0 {
		err = saveWav(audioData, "output.wav")
		if err != nil {
			log.Fatal("save wav error: ", err)
		}
	} else {
		fmt.Println("No audio data received.")
	}
}

func saveWav(data []byte, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// WAV header
	// Assume 24kHz, 16-bit, mono as per ReplicatedVoiceConfig spec.
	sampleRate := uint32(24000)
	bitsPerSample := uint16(16)
	channels := uint16(1)
	byteRate := sampleRate * uint32(channels) * uint32(bitsPerSample) / 8

	// RIFF header
	f.Write([]byte("RIFF"))
	binary.Write(f, binary.LittleEndian, uint32(36+len(data)))
	f.Write([]byte("WAVE"))

	// fmt chunk
	f.Write([]byte("fmt "))
	binary.Write(f, binary.LittleEndian, uint32(16))
	binary.Write(f, binary.LittleEndian, uint16(1)) // PCM
	binary.Write(f, binary.LittleEndian, channels)
	binary.Write(f, binary.LittleEndian, sampleRate)
	binary.Write(f, binary.LittleEndian, byteRate)
	binary.Write(f, binary.LittleEndian, uint16(channels*bitsPerSample/8))
	binary.Write(f, binary.LittleEndian, bitsPerSample)

	// data chunk
	f.Write([]byte("data"))
	binary.Write(f, binary.LittleEndian, uint32(len(data)))
	f.Write(data)

	fmt.Println("Saved audio response to", filename)
	return nil
}
