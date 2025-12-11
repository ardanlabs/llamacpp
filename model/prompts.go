package model

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/hybridgroup/yzma/pkg/mtmd"
	"github.com/nikolalohinski/gonja/v2"
	"github.com/nikolalohinski/gonja/v2/builtins"
	"github.com/nikolalohinski/gonja/v2/exec"
	"github.com/nikolalohinski/gonja/v2/loaders"
)

func (m *Model) applyRequestJinjaTemplate(d D) (string, [][]byte, error) {
	dCopy := make(D, len(d))
	maps.Copy(dCopy, d)

	// We need to identify if there is media in the request. If there is
	// we want to replace the actual media with a media marker `<__media__>`.
	// We will move the media to it's own slice. The next call that will happen
	// is `processBitmap` which will process the prompt and media.

	var media [][]byte

	for _, doc := range dCopy["messages"].([]D) {
		if content, exists := doc["content"]; exists {
			switch value := content.(type) {
			case []byte:
				media = append(media, value)
				doc["content"] = fmt.Sprintf("%s\n", mtmd.DefaultMarker())
			}
		}
	}

	prompt, err := m.applyJinjaTemplate(dCopy)
	if err != nil {
		return "", nil, err
	}

	return prompt, media, nil
}

func (m *Model) applyJinjaTemplate(d D) (string, error) {
	if m.template == "" {
		return "", errors.New("apply-jinja-template:no template found")
	}

	gonja.DefaultLoader = &noFSLoader{}

	t, err := newTemplateWithFixedItems(m.template)
	if err != nil {
		return "", fmt.Errorf("apply-jinja-template:failed to parse template: %w", err)
	}

	data := exec.NewContext(d)

	s, err := t.ExecuteToString(data)
	if err != nil {
		return "", fmt.Errorf("apply-jinja-template:failed to execute template: %w", err)
	}

	return s, nil
}

// =============================================================================

type noFSLoader struct{}

func (nl *noFSLoader) Read(path string) (io.Reader, error) {
	return nil, errors.New("no-fs-loader:filesystem access disabled")
}

func (nl *noFSLoader) Resolve(path string) (string, error) {
	return "", errors.New("no-fs-loader:filesystem access disabled")
}

func (nl *noFSLoader) Inherit(from string) (loaders.Loader, error) {
	return nil, errors.New("no-fs-loader:filesystem access disabled")
}

// =============================================================================

// newTemplateWithFixedItems creates a gonja template with a fixed items() method
// that properly returns key-value pairs (the built-in one only returns values).
func newTemplateWithFixedItems(source string) (*exec.Template, error) {
	rootID := fmt.Sprintf("root-%s", string(sha256.New().Sum([]byte(source))))

	loader, err := loaders.NewFileSystemLoader("")
	if err != nil {
		return nil, err
	}

	shiftedLoader, err := loaders.NewShiftedLoader(rootID, bytes.NewReader([]byte(source)), loader)
	if err != nil {
		return nil, err
	}

	// Create custom environment with fixed items() method
	customContext := builtins.GlobalFunctions.Inherit()
	customContext.Set("add_generation_prompt", true)
	customContext.Set("strftime_now", func(format string) string {
		return time.Now().Format("2006-01-02")
	})
	customContext.Set("raise_exception", func(msg string) (string, error) {
		return "", errors.New(msg)
	})

	env := exec.Environment{
		Context:           customContext,
		Filters:           builtins.Filters,
		Tests:             builtins.Tests,
		ControlStructures: builtins.ControlStructures,
		Methods: exec.Methods{
			Dict: exec.NewMethodSet(map[string]exec.Method[map[string]any]{
				"keys": func(self map[string]any, selfValue *exec.Value, arguments *exec.VarArgs) (any, error) {
					if err := arguments.Take(); err != nil {
						return nil, err
					}
					keys := make([]string, 0, len(self))
					for key := range self {
						keys = append(keys, key)
					}
					sort.Strings(keys)
					return keys, nil
				},
				"items": func(self map[string]any, selfValue *exec.Value, arguments *exec.VarArgs) (any, error) {
					if err := arguments.Take(); err != nil {
						return nil, err
					}
					// Return [][]any where each inner slice is [key, value]
					// This allows gonja to unpack: for k, v in dict.items()
					items := make([][]any, 0, len(self))
					for key, value := range self {
						items = append(items, []any{key, value})
					}
					return items, nil
				},
			}),
			Str:   builtins.Methods.Str,
			List:  builtins.Methods.List,
			Bool:  builtins.Methods.Bool,
			Float: builtins.Methods.Float,
			Int:   builtins.Methods.Int,
		},
	}

	return exec.NewTemplate(rootID, gonja.DefaultConfig, shiftedLoader, &env)
}

// =============================================================================

func readJinjaTemplate(fileName string) (string, error) {
	data, err := os.ReadFile(fileName)
	if err != nil {
		return "", fmt.Errorf("read-jinja-template:failed to read file: %w", err)
	}

	return string(data), nil
}

func isOpenAIMediaRequest(req D) bool {
	messages, ok := req["messages"].([]D)
	if !ok {
		return false
	}

	for _, doc := range messages {
		contentField, exists := doc["content"]
		if !exists {
			continue
		}

		contentDocs, ok := contentField.([]D)
		if !ok {
			continue
		}

		for _, contentDoc := range contentDocs {
			typ, _ := contentDoc["type"].(string)
			if typ == "image_url" || typ == "image" || typ == "video_url" || typ == "audio_url" {
				return true
			}
		}
	}

	return false
}

func openAIToMediaMessage(req D) (D, error) {
	type tm struct {
		text string
		data []byte
	}

	var textMedia []tm

	// OpenAI spec.
	// Understand the template for a model can be coded differently
	// but I've noticed they all support the content field having the
	// raw media. So this code will conform to that standard.
	// Look at model.ImageMessage for the format.

	messages, ok := req["messages"].([]D)
	if !ok {
		return nil, errors.New("missing messages field")
	}

	// Iterate over the message documents looking for media content.
	for _, doc := range messages {
		// We expect to find a content field.
		contentField, exists := doc["content"]
		if !exists {
			return nil, errors.New("expecting content field")
		}

		// If the content field is a string, we only have text and no
		// media to go this is message.
		contentText, ok := contentField.(string)
		if ok {
			textMedia = append(textMedia, tm{
				text: contentText,
			})
			continue
		}

		// Check if the content is the binary data or OpenAI spec.
		contentDocs, ok := contentField.([]D)
		if !ok {
			return nil, fmt.Errorf("expecting the content field to be an array of docs, %T", contentField)
		}

		// WE NOW HAVE MEDIA

		// The text and media will be in 2 separate documents.
		// We expect to have text with the media.
		if len(contentDocs) != 2 {
			return nil, errors.New("expecting 2 documents inside the content field")
		}

		var found int
		var mediaText string
		var mediaData []byte

		for _, contentDoc := range contentDocs {
			// We should have a text field of type string. I don't need
			// to check the type because I'll just get the zero value
			// string if the caller made a mistake.
			typ, _ := contentDoc["type"].(string)

			// If the type is "text" we need to capture this content.
			if typ == "text" {
				textContent, _ := contentDoc["text"].(string)

				// We found text so let's mark that.
				found++

				mediaText = textContent

				// If we found both the text and data, save it.
				if found == 2 {
					textMedia = append(textMedia, tm{
						text: mediaText,
						data: mediaData,
					})

					found = 0
					mediaText = ""
					mediaData = nil
				}

				// Continue on to the next document in the slice.
				continue
			}

			// We expect to have a field that matched the type value.
			mediaContent, exists := contentDoc[typ]
			if !exists {
				return nil, fmt.Errorf("missing %q field under content", typ)
			}

			// We expect this field to be a document.
			mediaField, ok := mediaContent.(D)
			if !ok {
				return nil, fmt.Errorf("%q field is not a document, %T", typ, mediaField)
			}

			// We expect this document to have a url field.
			base64Data, exists := mediaField["url"]
			if !exists {
				base64Data, exists = mediaField["data"]
				if !exists {
					return nil, errors.New("expecting url or data field")
				}
			}

			data, ok := base64Data.(string)
			if !ok {
				return nil, errors.New("expecting media to be a base64 string")
			}

			// Remove the data URI marker if present (e.g., "data:image/png;base64,")
			if idx := strings.Index(data, ";base64,"); idx != -1 && strings.HasPrefix(data, "data:") {
				data = data[idx+8:]
			}

			// Decode the base64 data back to binary data.
			decoded, err := base64.StdEncoding.DecodeString(data)
			if err != nil {
				return nil, fmt.Errorf("unable to decode base64 data: %w", err)
			}

			// We found media so let's mark that.
			found++
			mediaData = decoded

			// If we found both the text and data, save it.
			if found == 2 {
				textMedia = append(textMedia, tm{
					text: mediaText,
					data: mediaData,
				})

				found = 0
				mediaText = ""
				mediaData = nil
			}
		}
	}

	docs := make([]D, 0, len(textMedia))
	for _, tm := range textMedia {
		if len(tm.data) > 0 {
			msgs := MediaMessage(tm.text, tm.data)
			docs = append(docs, msgs...)
			continue
		}

		docs = append(docs, TextMessage("user", tm.text))
	}

	req["messages"] = docs

	return req, nil
}
