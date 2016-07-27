package threading

const (
	// AlertAllMessages is the settings key used to determine if a notification should generate an alert for all messages or just the first unread
	AlertAllMessages = "alert_all_messages"
	// PreviewPatientMessageContentInNotification is the settings key used to determine whether or not to send the actual content of the message in the notification payload
	// for patient messages.
	PreviewPatientMessageContentInNotification = "preview_patient_message_content"
	// PreviewTeamMessageContentInNotification is the settings key used to determine whether or not to send the actual content of the message in the notification payload
	// for team messages
	PreviewTeamMessageContentInNotification = "preview_team_message_content"
)
