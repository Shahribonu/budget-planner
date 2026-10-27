// Package telegram is a minimal Bot API client: only the methods and fields the bot uses.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

type User struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	Username     string `json:"username,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
}

type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type Message struct {
	MessageID int64  `json:"message_id"`
	From      *User  `json:"from,omitempty"`
	Chat      Chat   `json:"chat"`
	Date      int64  `json:"date"`
	Text      string `json:"text,omitempty"`
	Voice     *Voice `json:"voice,omitempty"`
}

type Voice struct {
	FileID   string `json:"file_id"`
	Duration int    `json:"duration"`
	MimeType string `json:"mime_type,omitempty"`
	FileSize int64  `json:"file_size,omitempty"`
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data,omitempty"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type InlineKeyboardButton struct {
	Text         string      `json:"text"`
	CallbackData string      `json:"callback_data,omitempty"`
	WebApp       *WebAppInfo `json:"web_app,omitempty"`
}

type WebAppInfo struct {
	URL string `json:"url"`
}

type SendMessage struct {
	ChatID      int64                 `json:"chat_id"`
	Text        string                `json:"text"`
	ParseMode   string                `json:"parse_mode,omitempty"`
	ReplyMarkup *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
	ReplyTo     *ReplyParameters      `json:"reply_parameters,omitempty"`
}

type ReplyParameters struct {
	MessageID int64 `json:"message_id"`
}

type EditMessageText struct {
	ChatID      int64                 `json:"chat_id"`
	MessageID   int64                 `json:"message_id"`
	Text        string                `json:"text"`
	ParseMode   string                `json:"parse_mode,omitempty"`
	ReplyMarkup *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

// Error is a Bot API error response.
type Error struct {
	Code        int    `json:"error_code"`
	Description string `json:"description"`
}

func (e *Error) Error() string { return fmt.Sprintf("telegram %d: %s", e.Code, e.Description) }

// IsBlocked reports whether the user blocked the bot or deleted their account.
func IsBlocked(err error) bool {
	var te *Error
	return errors.As(err, &te) && te.Code == 403
}

type Client struct {
	token string
	base  string
	http  *http.Client
}

func New(token string) *Client {
	return &Client{token: token, base: "https://api.telegram.org", http: &http.Client{Timeout: 70 * time.Second}}
}

// WithBaseURL points the client at a fake server in tests.
func (c *Client) WithBaseURL(u string) *Client { c.base = u; return c }

func (c *Client) call(ctx context.Context, method string, in, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/bot"+c.token+"/"+method, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("telegram %s: %w", method, errors.Unwrap(err)) // the URL holds the token; don't log it
	}
	defer res.Body.Close()
	var env struct {
		OK          bool            `json:"ok"`
		Result      json.RawMessage `json:"result"`
		ErrorCode   int             `json:"error_code"`
		Description string          `json:"description"`
	}
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		return fmt.Errorf("telegram %s: decode: %w", method, err)
	}
	if !env.OK {
		return &Error{Code: env.ErrorCode, Description: env.Description}
	}
	if out != nil {
		return json.Unmarshal(env.Result, out)
	}
	return nil
}

func (c *Client) SendMessage(ctx context.Context, m SendMessage) (Message, error) {
	var out Message
	err := c.call(ctx, "sendMessage", m, &out)
	return out, err
}

func (c *Client) EditMessageText(ctx context.Context, m EditMessageText) error {
	err := c.call(ctx, "editMessageText", m, nil)
	var te *Error
	if errors.As(err, &te) && te.Code == 400 && bytes.Contains([]byte(te.Description), []byte("not modified")) {
		return nil
	}
	return err
}

func (c *Client) AnswerCallbackQuery(ctx context.Context, id, text string) error {
	return c.call(ctx, "answerCallbackQuery", map[string]any{"callback_query_id": id, "text": text}, nil)
}

func (c *Client) GetUpdates(ctx context.Context, offset int64, timeoutSec int) ([]Update, error) {
	var out []Update
	err := c.call(ctx, "getUpdates", map[string]any{
		"offset": offset, "timeout": timeoutSec, "allowed_updates": []string{"message", "callback_query"},
	}, &out)
	return out, err
}

func (c *Client) SetWebhook(ctx context.Context, url, secret string) error {
	return c.call(ctx, "setWebhook", map[string]any{
		"url": url, "secret_token": secret, "allowed_updates": []string{"message", "callback_query"},
	}, nil)
}

func (c *Client) DeleteWebhook(ctx context.Context) error {
	return c.call(ctx, "deleteWebhook", map[string]any{}, nil)
}

type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

func (c *Client) SetMyCommands(ctx context.Context, cmds []BotCommand, languageCode string) error {
	in := map[string]any{"commands": cmds}
	if languageCode != "" {
		in["language_code"] = languageCode
	}
	return c.call(ctx, "setMyCommands", in, nil)
}

// DownloadFile fetches a file by file_id, refusing anything larger than maxBytes.
func (c *Client) DownloadFile(ctx context.Context, fileID string, maxBytes int64) ([]byte, error) {
	var f struct {
		FilePath string `json:"file_path"`
		FileSize int64  `json:"file_size"`
	}
	if err := c.call(ctx, "getFile", map[string]any{"file_id": fileID}, &f); err != nil {
		return nil, err
	}
	if f.FileSize > maxBytes {
		return nil, fmt.Errorf("file too large: %d bytes", f.FileSize)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/file/bot"+c.token+"/"+f.FilePath, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("telegram download: %w", errors.Unwrap(err))
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("telegram download: status %d", res.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > maxBytes {
		return nil, fmt.Errorf("file too large")
	}
	return b, nil
}
