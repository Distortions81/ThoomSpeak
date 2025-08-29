package main

const (
	maxMessages = 1000
)

var consoleLog = messageLog{max: maxMessages}

func consoleMessage(msg string) {
	if msg == "" {
		return
	}

	consoleLog.Add(msg)
	appendConsoleLog(msg)

	updateConsoleWindow()

	runConsoleTriggers(msg)
}

func getConsoleMessages() []string {
	format := gs.TimestampFormat
	if format == "" {
		format = "3:04PM"
	}
	return consoleLog.Entries(format, gs.ConsoleTimestamps)
}
