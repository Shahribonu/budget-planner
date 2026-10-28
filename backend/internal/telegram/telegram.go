// Package telegram is a minimal Bot API client: only the methods and fields the bot uses.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"
)

type Update struct {
	UpdateID         int64             `json:"update_id"`
	Message          *Message          `json:"message,omitempty"`
	CallbackQuery    *CallbackQuery    `json:"callback_query,omitempty"`
	PreCheckoutQuery *PreCheckoutQuery `json:"pre_checkout_query,omitempty"`
}

// PreCheckoutQuery must be answered within 10 seconds or the payment fails.
type PreCheckoutQuery struct {
	ID             string `json:"id"`
	From           User   `json:"from"`
	Currency       string `json:"currency"`
	TotalAmount    int    `json:"total_amount"`
	InvoicePayload string `json:"invoice_payload"`
}

type SuccessfulPayment struct {
	Currency                   string `json:"currency"`
	TotalAmount                int    `json:"total_amount"`
	InvoicePayload             string `json:"invoice_payload"`
	SubscriptionExpirationDate int64  `json:"subscription_expiration_date,omitempty"`
	IsRecurring                bool   `json:"is_recurring,omitempty"`
	TelegramPaymentChargeID    string `json:"telegram_payment_charge_id"`
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

	SuccessfulPayment *SuccessfulPayment `json:"successful_payment,omitempty"`
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
	URL          string      `json:"url,omitempty"`
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
	return c.do(req, method, out)
}

// do sends a prepared Bot API request and decodes the response envelope.
func (c *Client) do(req *http.Request, method string, out any) error {
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
		"offset": offset, "timeout": timeoutSec, "allowed_updates": allowedUpdates,
	}, &out)
	return out, err
}

func (c *Client) SetWebhook(ctx context.Context, url, secret string) error {
	return c.call(ctx, "setWebhook", map[string]any{
		"url": url, "secret_token": secret, "allowed_updates": allowedUpdates,
	}, nil)
}

var allowedUpdates = []string{"message", "callback_query", "pre_checkout_query"}

type LabeledPrice struct {
	Label  string `json:"label"`
	Amount int    `json:"amount"`
}

func (c *Client) AnswerPreCheckoutQuery(ctx context.Context, id string, ok bool, errMsg string) error {
	in := map[string]any{"pre_checkout_query_id": id, "ok": ok}
	if !ok {
		in["error_message"] = errMsg
	}
	return c.call(ctx, "answerPreCheckoutQuery", in, nil)
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

type PhotoSize struct {
	FileID   string `json:"file_id"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	FileSize int64  `json:"file_size,omitempty"`
}

// ProfilePhoto returns the file_id of the user's current profile photo in the smallest
// size at least minSide pixels wide, or "" when there is none (or privacy hides it).
func (c *Client) ProfilePhoto(ctx context.Context, userID int64, minSide int) (string, error) {
	var out struct {
		Photos [][]PhotoSize `json:"photos"`
	}
	if err := c.call(ctx, "getUserProfilePhotos", map[string]any{"user_id": userID, "limit": 1}, &out); err != nil {
		return "", err
	}
	if len(out.Photos) == 0 || len(out.Photos[0]) == 0 {
		return "", nil
	}
	sizes := out.Photos[0]
	best := sizes[len(sizes)-1]
	for _, s := range sizes {
		if s.Width >= minSide && s.Width < best.Width {
			best = s
		}
	}
	return best.FileID, nil
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

// SendDocument uploads a file into a chat.
func (c *Client) SendDocument(ctx context.Context, chatID int64, filename string, data []byte, caption string) error {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("chat_id", strconv.FormatInt(chatID, 10))
	if caption != "" {
		_ = mw.WriteField("caption", caption)
	}
	fw, err := mw.CreateFormFile("document", filename)
	if err != nil {
		return err
	}
	if _, err := fw.Write(data); err != nil {
		return err
	}
	if err := mw.Close(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/bot"+c.token+"/sendDocument", &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return c.do(req, "sendDocument", nil)
}

// SetMyDescription sets the bot's description (shown before /start) and short
// description (profile and shares) for one language, or the default when languageCode is "".
func (c *Client) SetMyDescription(ctx context.Context, description, short, languageCode string) error {
	in := map[string]any{"description": description}
	sh := map[string]any{"short_description": short}
	if languageCode != "" {
		in["language_code"], sh["language_code"] = languageCode, languageCode
	}
	if err := c.call(ctx, "setMyDescription", in, nil); err != nil {
		return err
	}
	return c.call(ctx, "setMyShortDescription", sh, nil)
}
