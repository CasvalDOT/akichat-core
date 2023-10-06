package core

type Chat interface {
	Login(string, string) (string, error)
	Logout() error
	ReadMessages(string) ([]Message, error)
	Snapshot(string) ([]Message, []User, error)
	WriteMessage(string) error
	WritePrivateMessage(string, string) error
	GetUsers() ([]User, error)
	ChangeUsername(string) error
	IsAuthenticated() bool
}

const (
	MessageSystemType     = "sys"
	MessageUserType       = "usr"
	MessageChannelPublic  = "public"
	ChatTypeHentakihabara = "hentakihabara"
)

type Message struct {
	ID      string `json:"id"`
	Author  User   `json:"author"`
	Content string `json:"content"`
	Time    string `json:"time"`
	Type    string `json:"type"`
	Target  string `json:"target"`
	Channel string `json:"channel"`
}

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func NewChat(provider string) Chat {
	switch provider {
	case ChatTypeHentakihabara:
		return NewHeintakihabaraChat()
	}

	return nil
}
