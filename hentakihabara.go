package core

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
)

type hentakihabaraChat struct {
	baseURL    string
	authCookie string
}

type HentaikihabaraRoot struct {
	XMLName  xml.Name               `xml:"root"`
	Messages HentaikihabaraMessages `xml:"messages"`
	Users    HentakihabaraUsers     `xml:"users"`
}

type HentakihabaraMessage struct {
	XMLName   xml.Name `xml:"message"`
	Username  string   `xml:"username"`
	Text      string   `xml:"text"`
	Time      string   `xml:"dateTime,attr"`
	ID        string   `xml:"id,attr"`
	UserID    string   `xml:"userID,attr"`
	ChannelID string   `xml:"channelID,attr"`
	Role      int      `xml:"userRole,attr"`
}

type HentaikihabaraMessages struct {
	XMLName xml.Name               `xml:"messages"`
	Items   []HentakihabaraMessage `xml:"message"`
}

type HentakihabaraUsers struct {
	XMLName xml.Name            `xml:"users"`
	Items   []HentakihabaraUser `xml:"user"`
}

type HentakihabaraUser struct {
	XMLName xml.Name `xml:"user"`
	Name    string   `xml:",chardata"`
	ID      string   `xml:"userID,attr"`
}

const (
	hentakihabaraBaseURL       = "http://hentakihabara2.altervista.org"
	hentakihabaraCookiePath    = "akichat-cookie.txt"
	hentakihabaraChatBotRole   = 4
	hentakihabaraLogoutMessage = "<?xml version=\"1.0\" encoding=\"UTF-8\"?><root><infos><info type=\"logout\"><![CDATA[./?logout=true]]></info></infos></root>"
)

func (c *hentakihabaraChat) formRequest(endpoint string, data map[string]string) ([]byte, http.Header, error) {
	fullEndpoint := c.baseURL + endpoint
	form := url.Values{}

	for key, value := range data {
		form.Add(key, value)
	}

	response, err := http.PostForm(fullEndpoint, form)
	if err != nil {
		return nil, nil, err
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)

	headers := response.Header.Clone()

	return body, headers, err
}

func (c *hentakihabaraChat) request(method string, endpoint string, queryparams map[string]string, data []byte, headers map[string]string) ([]byte, http.Header, error) {
	reader := bytes.NewReader(data)

	fullEndpoint := c.baseURL + endpoint

	request, err := http.NewRequest(method, fullEndpoint, reader)
	if err != nil {
		return nil, nil, err
	}

	request.Header.Set("Cookie", c.authCookie)

	for key, value := range headers {
		request.Header.Set(key, value)
	}

	query := request.URL.Query()
	for key, value := range queryparams {
		query.Add(key, value)
	}
	request.URL.RawQuery = query.Encode()

	client := http.Client{}

	response, err := client.Do(request)
	if err != nil {
		return nil, nil, err
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)

	headersResponse := response.Header.Clone()

	return body, headersResponse, err
}

func (c *hentakihabaraChat) setInlineCookie(value string) error {
	c.authCookie = value
	return nil
}

func (c *hentakihabaraChat) setCookie() error {
	file, err := os.ReadFile(hentakihabaraCookiePath)
	if err != nil {
		return err
	}

	c.authCookie = string(file)
	return nil
}

func (c *hentakihabaraChat) saveCookie(userID string) error {
	_, err := os.Create(hentakihabaraCookiePath)
	if err != nil {
		return err
	}

	os.WriteFile(hentakihabaraCookiePath, []byte(c.authCookie+";userID="+userID), 600)

	return nil
}

func (c *hentakihabaraChat) isAuthenticated(data []byte) bool {
	if string(data) == hentakihabaraLogoutMessage {
		return false
	}

	return true
}

func (c *hentakihabaraChat) extractUsers(root HentaikihabaraRoot) []User {
	users := []User{}
	for _, user := range root.Users.Items {
		users = append(users, User{
			Name: user.Name,
			ID:   user.ID,
		})
	}

	return users
}

func (c *hentakihabaraChat) extractMessages(root HentaikihabaraRoot) []Message {
	messages := []Message{}

	privateMessageRgx := regexp.MustCompile(`^/(privmsg|privmsgto)\s(.*)`)
	privateToRgx := regexp.MustCompile(`^/privmsgto\s(.*?)\s`)

	for _, message := range root.Messages.Items {
		messageAuthor := User{
			ID:   message.UserID,
			Name: message.Username,
		}

		for _, user := range root.Users.Items {
			if user.ID == message.UserID {
				messageAuthor = User{
					ID:   user.ID,
					Name: user.Name,
				}
			}
		}

		messageType := MessageUserType
		if message.Role == hentakihabaraChatBotRole {
			messageType = MessageSystemType
		}

		messageTarget := ""
		if privateToRgx.MatchString(message.Text) {
			res := privateToRgx.FindStringSubmatch(message.Text)[1]
			for _, user := range root.Users.Items {
				if user.Name == res {
					messageTarget = user.ID
				}
			}
		}

		messageChannel := MessageChannelPublic
		messageContent := message.Text
		if privateMessageRgx.MatchString(message.Text) {
			messageChannel = message.ChannelID
			messageContent = privateMessageRgx.ReplaceAllString(messageContent, "$2")
		}

		messages = append(messages, Message{
			ID:      message.ID,
			Author:  messageAuthor,
			Content: messageContent,
			Time:    message.Time,
			Type:    messageType,
			Target:  messageTarget,
			Channel: messageChannel,
		})
	}

	return messages
}

func (c *hentakihabaraChat) IsAuthenticated() bool {
	c.setCookie()
	queryParams := map[string]string{
		"ajax":      "true",
		"lastID":    "0",
		"channelID": "0",
	}

	data, _, err := c.request("GET", "/", queryParams, nil, map[string]string{})
	if err != nil {
		return false
	}

	if !c.isAuthenticated(data) {
		return false
	}

	return true
}

func (c *hentakihabaraChat) Snapshot(fromMessageID string) ([]Message, []User, error) {
	c.setCookie()
	queryParams := map[string]string{
		"ajax":      "true",
		"lastID":    fromMessageID,
		"channelID": "0",
	}

	data, _, err := c.request("GET", "/", queryParams, nil, map[string]string{})

	if !c.isAuthenticated(data) {
		return nil, nil, errors.New("unathorized")
	}

	var root HentaikihabaraRoot
	err = xml.Unmarshal(data, &root)
	if err != nil {
		return nil, nil, err
	}

	users := c.extractUsers(root)
	messages := c.extractMessages(root)

	return messages, users, err
}

func (c *hentakihabaraChat) ReadMessages(fromMessageID string) ([]Message, error) {
	c.setCookie()
	queryParams := map[string]string{
		"ajax":      "true",
		"lastID":    fromMessageID,
		"channelID": "0",
	}

	data, _, err := c.request("GET", "/", queryParams, nil, map[string]string{})

	if !c.isAuthenticated(data) {
		return nil, errors.New("unathorized")
	}

	var root HentaikihabaraRoot
	err = xml.Unmarshal(data, &root)
	if err != nil {
		return nil, err
	}

	messages := c.extractMessages(root)

	return messages, err
}

func (c *hentakihabaraChat) WriteMessage(message string) error {
	c.setCookie()

	form := url.Values{}
	form.Add("text", message)
	body := form.Encode()

	queryParams := map[string]string{
		"ajax": "true",
	}

	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}

	data, _, err := c.request("POST", "/", queryParams, []byte(body), headers)
	if err != nil {
		return err
	}

	if !c.isAuthenticated(data) {
		return errors.New("unathorized")
	}

	return nil
}

func (c *hentakihabaraChat) WritePrivateMessage(user string, message string) error {
	message = fmt.Sprintf("/msg %s %s", user, message)
	return c.WriteMessage(message)
}

func (c *hentakihabaraChat) ChangeUsername(username string) error {
	message := fmt.Sprintf("/nick %s", username)
	return c.WriteMessage(message)
}

func (c *hentakihabaraChat) GetUsers() ([]User, error) {
	c.setCookie()
	queryParams := map[string]string{
		"ajax":      "true",
		"lastID":    "0",
		"channelID": "0",
	}

	data, _, err := c.request("GET", "/", queryParams, nil, map[string]string{})
	if err != nil {
		return nil, err
	}

	if !c.isAuthenticated(data) {
		return nil, errors.New("unathorized")
	}

	var root HentaikihabaraRoot
	err = xml.Unmarshal(data, &root)
	if err != nil {
		return nil, err
	}

	users := c.extractUsers(root)

	return users, nil
}

func (c *hentakihabaraChat) Login(username string, password string) (string, error) {
	form := map[string]string{
		"userName":    username,
		"password":    password,
		"channelName": "Public",
		"lang":        "en",
		"submit":      "Login",
		"login":       "login",
		"redirect":    "",
	}

	_, headers, err := c.formRequest("/", form)
	if err != nil {
		return "", err
	}

	c.authCookie = headers.Get("Set-Cookie")

	users, err := c.GetUsers()
	if err != nil {
		return "", err
	}

	userID := ""
	for _, user := range users {
		if user.Name == fmt.Sprintf("(%s)", username) {
			userID = user.ID
		}
	}

	c.saveCookie(userID)

	return userID, err
}

func (c *hentakihabaraChat) Logout() error {
	c.WriteMessage("/quit")
	return os.Remove(hentakihabaraCookiePath)
}

func NewHeintakihabaraChat() Chat {
	return &hentakihabaraChat{
		baseURL:    hentakihabaraBaseURL,
		authCookie: "",
	}
}
