package model

import (
	"encoding/base64"
	"testing"
)

func Test_OpenAIToMediaMessage(t *testing.T) {
	data := []byte("this is not really an image but it will do")

	openEncoded := base64.StdEncoding.EncodeToString(data)

	d := D{
		"messages": DocumentArray(
			imageMessageOpenAI("what do you see in the picture?", openEncoded),
		),
	}

	d, err := openAIToMediaMessage(d)
	if err != nil {
		t.Fatalf("convering openai to media message: %s", err)
	}

	media := d["messages"].([]D)

	if len(media) != 2 {
		t.Fatalf("should have 2 documents in the media message, got %d", len(media))
	}

	mediaEncoded := base64.StdEncoding.EncodeToString(media[0]["content"].([]byte))

	if openEncoded != mediaEncoded {
		t.Fatalf("media mismatch from input to output\ngot:[%s]\nexp:[%s]", openEncoded, mediaEncoded)
	}
}

func imageMessageOpenAI(text string, image string) D {
	return D{
		"role": "user",
		"content": []D{
			{
				"type": "text",
				"text": text,
			},
			{
				"type": "image_url",
				"image_url": D{
					"url": image,
				},
			},
		},
	}
}

func audioMessageOpenAI(text string, audio string) D {
	return D{
		"role": "user",
		"content": []D{
			{
				"type": "text",
				"text": text,
			},
			{
				"type": "input_audio",
				"input_audio": D{
					"url": audio,
				},
			},
		},
	}
}
