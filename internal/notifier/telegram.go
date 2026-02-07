package notifier

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/amirhdev/crypto-trader/internal/config"

	"github.com/rs/zerolog/log"
)

type Telegram struct {
	token   string
	chatIDs []string
	client  *http.Client
}

func New(cfg *config.Config) *Telegram {
	return &Telegram{
		token:   cfg.Telegram.BotToken,
		chatIDs: cfg.Telegram.ChatIDs,
		client:  &http.Client{Timeout: 8 * time.Second},
	}
}

func (t *Telegram) Send(msg string) {
	if t.token == "" {
		return
	}
	msg = strings.ReplaceAll(msg, "", "\\")
	for _, chatID := range t.chatIDs {
		url := fmt.Sprintf(
			"https://api.telegram.org/bot%s/sendMessage?chat_id=%s&text=%s&parse_mode=Markdown",
			t.token, chatID, msg,
		)

		resp, err := t.client.Get(url)
		if err != nil || (resp != nil && resp.StatusCode != 200) {
			log.Warn().Err(err).Str("chat", chatID).Msg("telegram send failed")
		}
		if resp != nil {
			resp.Body.Close()
		}
	}
}
