package core

// The files Misoku's "/" menu attaches to a note, by kind. The attach
// dialogs offer them, and sync setup counts them, with notes, as what a
// vault is made of.
var (
	// ImageExts are the image files a note can show.
	ImageExts = []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".bmp", ".heic", ".avif"}
	// VideoExts are the video files a note can link.
	VideoExts = []string{".mp4", ".mov", ".webm", ".mkv", ".avi", ".m4v"}
	// AudioExts are the audio files a note can link.
	AudioExts = []string{".mp3", ".wav", ".m4a", ".ogg", ".flac", ".aac"}
)
