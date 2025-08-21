package main

const (
	maxChatMessages = 1000
)

var chatLog = messageLog{max: maxChatMessages}

func chatMessage(msg string) {
	if msg == "" {
		return
	}

	chatLog.Add(msg)

	updateChatWindow()

	if gs.ChatTTS && !blockTTS {
		speakChatMessage(msg)
	}
}

func getChatMessages() []string {
	format := gs.TimestampFormat
	if format == "" {
		format = "3:04PM"
	}
	return chatLog.Entries(format, gs.ChatTimestamps)
}
