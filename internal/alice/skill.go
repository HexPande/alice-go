package alice

import "strings"

// Handle is the entry point for the skill's dialogue scenarios.
func Handle(req Request) Response {
	reply := Reply{Text: "Пока я только учусь. Скажите «помощь», чтобы узнать, что я умею."}
	command := strings.ToLower(strings.TrimSpace(req.Request.Command))

	switch {
	case req.Session.New:
		reply.Text = "Привет! Это заготовка навыка Алисы на Go. Скажите «помощь» или «выход»."
	case command == "помощь" || command == "что ты умеешь":
		reply.Text = "Я умею здороваться и завершать разговор. Скажите «привет» или «выход»."
	case command == "привет":
		reply.Text = "Привет! Рада вас слышать."
	case command == "выход" || command == "пока" || command == "хватит":
		reply.Text = "До встречи!"
		reply.EndSession = true
	}

	return Response{Version: "1.0", Response: reply}
}
