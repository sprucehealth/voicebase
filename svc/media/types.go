package media

var (
	// SupportedImageMIMETypes returns a list of image related mimetypes that the media service can serve.
	SupportedImageMIMETypes = []string{
		"image/png",
		"image/jpeg",
	}

	// SupportedVideoMIMETypes returns a list of video related mimetypes that the media service can serve.
	SupportedVideoMIMETypes = []string{
		"video/3gpp",
		"video/mp4",
	}

	// SupportedAudioMIMETypes returns a list of audio related mimetypes that the media service can serve.
	SupportedAudioMIMETypes = []string{
		"audio/mpeg",
		"audio/wav",
	}

	// SupportedDocumentMIMETypes returns a list of document related mimetypes that the media service can serve.
	SupportedDocumentMIMETypes = []string{
		"application/pdf",
	}

	// SupportedMIMETypes returns all the mime types the media service can serve.
	SupportedMIMETypes = append(append(append(SupportedImageMIMETypes, SupportedVideoMIMETypes...), SupportedAudioMIMETypes...), SupportedDocumentMIMETypes...)
)
