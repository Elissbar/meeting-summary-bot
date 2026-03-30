package salutespeech

const task = `
{
  "options": {
    "model": "general",
    "audio_encoding": "%s",
    "sample_rate": 16000,
    "language": "ru-RU",
    "enable_profanity_filter": true,
    "hypotheses_count": 1,
    "channels_count": 1,
    "speaker_separation_options": {
      "enable": %t,
      "enable_only_main_speaker": false,
      "count": 10
    }
  },
  "request_file_id": "%s"
}`