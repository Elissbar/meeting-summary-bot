package handler

import (
	"fmt"

	"github.com/Elissbar/meeting-summary-bot/internal/salutespeech"
	tg "gopkg.in/telebot.v3"
)

func OnVoice(b *tg.Bot) func(c tg.Context) error {
	return func(c tg.Context) error {
		voice := c.Message().Voice
		fmt.Println("Voice:", voice)

		rc, err := b.File(&voice.File)
		if err != nil {
			return c.Send("Could not get file info.")
		}
		defer rc.Close()

		salute := salutespeech.NewSaluteSpeechClient(
			"https://ngw.devices.sberbank.ru:9443/api/v2/oauth",
			"MDE5ZDExNTUtYmRhMC03YjY4LTgxZWMtN2MzYWEzYTczODNmOmRhM2EzYWZiLTg3MmItNGM1Zi1hMWM2LTE5NWIzMjA4YTMyMw==",
			"SALUTE_SPEECH_PERS",
		)
		err = salute.Authorization()
		if err != nil {
			return c.Send("Authorization SaluteSpeech API error")
		}

		err = salute.Send(rc)
		if err != nil {
			return c.Send("Upload file into Salute error")
		}

		return c.Send("Your voice message link: ")
	}
}
