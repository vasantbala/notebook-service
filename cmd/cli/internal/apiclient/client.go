package apiclient

import (
"bufio"
"bytes"
"encoding/json"
"fmt"
"net/http"
"strings"
"time"

"github.com/vasantbala/notebook-service/internal/model"
)

// Client is a thin HTTP client for the notebook-service REST API.
type Client struct {
BaseURL    string
Token      string
httpClient *http.Client
}

// New creates a Client. Token is sent as a Bearer token on every request.
func New(baseURL, token string) *Client {
return &Client{
BaseURL:    strings.TrimRight(baseURL, "/"),
Token:      token,
httpClient: &http.Client{Timeout: 30 * time.Second},
}
}

func (c *Client) do(method, path string, body any) (*http.Response, error) {
var buf bytes.Buffer
if body != nil {
if err := json.NewEncoder(&buf).Encode(body); err != nil {
return nil, err
}
}
req, err := http.NewRequest(method, c.BaseURL+path, &buf)
if err != nil {
return nil, err
}
req.Header.Set("Authorization", "Bearer "+c.Token)
if body != nil {
req.Header.Set("Content-Type", "application/json")
}
return c.httpClient.Do(req)
}

// ListNotebooks returns all notebooks for the authenticated user.
func (c *Client) ListNotebooks() ([]model.Notebook, error) {
resp, err := c.do(http.MethodGet, "/notebooks/", nil)
if err != nil {
return nil, err
}
defer resp.Body.Close()
if resp.StatusCode != http.StatusOK {
return nil, fmt.Errorf("server returned %s", resp.Status)
}
var notebooks []model.Notebook
return notebooks, json.NewDecoder(resp.Body).Decode(&notebooks)
}

// ListConversations returns all conversations in the given notebook.
func (c *Client) ListConversations(notebookID string) ([]model.Conversation, error) {
resp, err := c.do(http.MethodGet, "/notebooks/"+notebookID+"/conversations/", nil)
if err != nil {
return nil, err
}
defer resp.Body.Close()
if resp.StatusCode != http.StatusOK {
return nil, fmt.Errorf("server returned %s", resp.Status)
}
var convs []model.Conversation
return convs, json.NewDecoder(resp.Body).Decode(&convs)
}

// CreateConversation creates a new conversation inside a notebook.
func (c *Client) CreateConversation(notebookID, title string) (*model.Conversation, error) {
body := map[string]string{"title": title}
resp, err := c.do(http.MethodPost, "/notebooks/"+notebookID+"/conversations/", body)
if err != nil {
return nil, err
}
defer resp.Body.Close()
if resp.StatusCode != http.StatusCreated {
return nil, fmt.Errorf("server returned %s", resp.Status)
}
var conv model.Conversation
return &conv, json.NewDecoder(resp.Body).Decode(&conv)
}

// ChatRequest is the body sent to the SSE chat endpoint.
type ChatRequest struct {
Query string `json:"query"`
TopK  int    `json:"top_k,omitempty"`
}

// ChatStream sends a chat query and calls onToken for every streamed token.
// Uses a client with no timeout because SSE streams are long-lived.
func (c *Client) ChatStream(notebookID, conversationID string, req ChatRequest, onToken func(string)) error {
var buf bytes.Buffer
if err := json.NewEncoder(&buf).Encode(req); err != nil {
return err
}

httpReq, err := http.NewRequest(
http.MethodPost,
fmt.Sprintf("%s/notebooks/%s/conversations/%s/chat", c.BaseURL, notebookID, conversationID),
&buf,
)
if err != nil {
return err
}
httpReq.Header.Set("Authorization", "Bearer "+c.Token)
httpReq.Header.Set("Content-Type", "application/json")
httpReq.Header.Set("Accept", "text/event-stream")

// No timeout: SSE streams are long-lived.
sseClient := &http.Client{}
resp, err := sseClient.Do(httpReq)
if err != nil {
return err
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
return fmt.Errorf("server returned %s", resp.Status)
}

scanner := bufio.NewScanner(resp.Body)
for scanner.Scan() {
line := scanner.Text()
if !strings.HasPrefix(line, "data:") {
continue
}
data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
if data == "[DONE]" {
break
}
var evt struct {
Token string `json:"token"`
}
if err := json.Unmarshal([]byte(data), &evt); err != nil {
continue
}
if evt.Token != "" {
onToken(evt.Token)
}
}
return scanner.Err()
}
