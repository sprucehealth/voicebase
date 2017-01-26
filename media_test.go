package voicebase

import "testing"

func TestTranscriptionText(t *testing.T) {
	media := &Media{
		Transcripts: map[string]*transcript{
			"latest": &transcript{
				Words: []*word{
					{
						Word: "Hi",
					},
					{
						Word: ",",
						M:    "punc",
					},
					{
						Word: "I'm",
					},
					{
						Word: "trying",
					},

					{
						Word: "to",
					},

					{
						Word: "test",
					},

					{
						Word: "this",
					},

					{
						Word: ".",
						M:    "punc",
					},
				},
			},
		},
	}

	expected := "Hi, I'm trying to test this."
	if media.TranscriptionText() != expected {
		t.Fatalf("Expected %s got %s", expected, media.TranscriptionText())
	}

	media = &Media{
		Transcripts: map[string]*transcript{
			"latest": &transcript{
				Words: []*word{
					{
						Word: "Hi",
					},
					{
						Word: "I'm",
					},
					{
						Word: "trying",
					},

					{
						Word: "to",
					},

					{
						Word: "test",
					},

					{
						Word: "this",
					},
				},
			},
		},
	}

	expected = "Hi I'm trying to test this"
	if media.TranscriptionText() != expected {
		t.Fatalf("Expected %s got %s", expected, media.TranscriptionText())
	}
}
