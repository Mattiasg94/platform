package session

type Conversation struct {
	messages []Message
}

func NewConversation() *Conversation {
	return &Conversation{
		messages: make([]Message, 0),
	}
}

func (c *Conversation) AddMessage(msg Message) {
	c.messages = append(c.messages, msg)
}

func (c *Conversation) History() []Message {
	return c.messages
}

func (c *Conversation) Clear() {
	c.messages = make([]Message, 0)
}

func (c *Conversation) Len() int {
	return len(c.messages)
}
