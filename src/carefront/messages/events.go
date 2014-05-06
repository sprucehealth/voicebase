package messages

type ConversationStartedEvent struct {
	ConversationId int64
	TopicId        int64
	FromId         int64
	ToId           int64
}

type ConversationReplyEvent struct {
	ConversationId int64
	MessageId      int64
	FromId         int64
}

type ConversationReadEvent struct {
	ConversationId int64
	FromId         int64
}
