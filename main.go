package main

import (
	"database/sql"
	"fmt"
	"github.com/Syfaro/telegram-bot-api"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"strconv"
	"time"
)

const ADMIN int64 = 375806606

func main() {
	bot := BotStart()
	my_db := DBStart()
	defer my_db.Close()
	go QuestUpdater(my_db, bot)
	BotUpdateLoop(bot, my_db)
}

var numericKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Zapytać"),
		tgbotapi.NewKeyboardButton("Dać odpowiedź"),
	),
)

func BotStart() *tgbotapi.BotAPI {
	bot, err := tgbotapi.NewBotAPI("1024969183:AAHUoIHL_4WQnQ-G4c0bsH5OMdjQUAfy8Os") //1057128816:AAE3MrZxSXnMPV1UNYuLbOQobd-sxUIhGw4 - AnonStud 1024969183:AAHUoIHL_4WQnQ-G4c0bsH5OMdjQUAfy8Os - AnonQuest
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Autorised on account %s", bot.Self.UserName)

	return bot
}
func BotUpdateLoop(my_bot *tgbotapi.BotAPI, database *sql.DB) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := my_bot.GetUpdatesChan(u)
	if err != nil {
		log.Panic(err)
	}

	for update := range updates {

		if update.Message == nil {
			continue
		}

		if !update.Message.IsCommand() {
			switch update.Message.Text {
			case "Zapytać":
				msg := tgbotapi.NewMessage(int64(update.Message.From.ID), "")
				if CheckReg(update.Message.From.ID, database, my_bot) {
					if CheckAnswr(update.Message.From.ID, database, my_bot) == 0 {
						if GetQuest(update.Message.From.ID, database, my_bot) != "" {
							if GetQuestCount(update.Message.From.ID, database, my_bot) > 0 {
								msg.Text = "Teraz podaj pytanie"
								StartQuest(update.Message.From.ID, database, my_bot)
								SetAsk(update.Message.From.ID, 1, database, my_bot)
								MinusQuestCount(update.Message.From.ID, database, my_bot)
							} else {
								msg.Text = "Dzisiejszy limit pytań został przekroczony!"
							}
						} else {
							msg.Text = "Musisz dokończyć pytanie!"
						}
					} else {
						msg.Text = "Musisz dokończyć odpowiedź!"
					}
				} else {
					msg.Text = "Musisz się zarejestrować. Użyj komendy /start."
				}
				_, err := my_bot.Send(msg)
				if err != nil {
					ErrorCatch(err.Error(), my_bot)
				}
				continue

			case "Dać odpowiedź":
				msg := tgbotapi.NewMessage(int64(update.Message.From.ID), "")

				if CheckReg(update.Message.From.ID, database, my_bot) {
					if !CheckAsk(update.Message.From.ID, database, my_bot) {
						msg.Text = "Napisz ID pytania, na które chcesz odpowiedzieć"
						SetAnswer(update.Message.From.ID, 1, database, my_bot)
					} else {
						msg.Text = "Musisz dokończyć pytanie!"
					}
				} else {
					msg.Text = "Musisz się zarejestrować. Użyj komendy /start."
				}

				_, err := my_bot.Send(msg)
				if err != nil {
					ErrorCatch(err.Error(), my_bot)
				}
				continue
			}

			switch CheckAnswr(update.Message.From.ID, database, my_bot) {
			case 1:
				msg := tgbotapi.NewMessage(int64(update.Message.From.ID), "")
				user_mes, _ := strconv.Atoi(update.Message.Text)
				if user_mes < 0 || user_mes > GetMaxID(database, my_bot) {
					msg.Text = "Wybrany zły ID"
					_, err := my_bot.Send(msg)
					if err != nil {
						ErrorCatch(err.Error(), my_bot)
					}
					continue
				}
				SetQuestID(update.Message.From.ID, user_mes, database, my_bot)
				msg.Text = "Napisz odpowiedź"
				SetAnswer(update.Message.From.ID, 2, database, my_bot)
				_, err := my_bot.Send(msg)
				if err != nil {
					ErrorCatch(err.Error(), my_bot)
				}
				continue
			case 2:
				msg := tgbotapi.NewMessage(int64(update.Message.From.ID), "")
				msg.Text = "Odpowiedź wysłana"
				SetAnswer(update.Message.From.ID, 0, database, my_bot)
				_, err := my_bot.Send(msg)
				if err != nil {
					ErrorCatch(err.Error(), my_bot)
				}

				msg = tgbotapi.NewMessage(int64(GetUserID(GetWantID(update.Message.From.ID, database, my_bot), database, my_bot)), "")
				msg.Text = fmt.Sprintf("Odpowiedź na pytanie ID - %v \n", GetWantID(update.Message.From.ID, database, my_bot)) + update.Message.Text
				_, err = my_bot.Send(msg)
				if err != nil {
					ErrorCatch(err.Error(), my_bot)
				}
				continue
			}

			if CheckAsk(update.Message.From.ID, database, my_bot) {
				msg := tgbotapi.NewMessage(int64(update.Message.From.ID), "")

				msg.Text = update.Message.Text
				if msg.Text != "" {
					NewQuest(update.Message.From.ID, update.Message.Text, database, my_bot)
					ShareQuest(update.Message.From.ID, database, my_bot)
					SetAsk(update.Message.From.ID, 0, database, my_bot)
					msg.Text = "Pytanie wysłane"
					_, err := my_bot.Send(msg)
					if err != nil {
						ErrorCatch(err.Error(), my_bot)
					}
				} else {
					msg.Text = "Nie możesz wysyłać nic oprócz tekstu!"
					_, err := my_bot.Send(msg)
					if err != nil {
						ErrorCatch(err.Error(), my_bot)
					}
				}
				continue
			} else {
				continue
			}
		}

		chat_id := update.Message.Chat.ID
		msg := tgbotapi.NewMessage(chat_id, "")

		switch update.Message.Command() {
		case "start":
			if !CheckReg(update.Message.From.ID, database, my_bot) {
				AddUser(update.Message.From.ID, database, my_bot)
				msg.Text = "Cześć. Tutaj możesz postawić anonimowe pytanie lub odpowiedzieć na inne anonimowe pytania.\n\n" +
					"Żeby zadać pytanie - użyj przycisku \"Zapytać\" lub komendy /new_question, następnie podaj tekst pytania. Po wysłaniu pytania do bota, program przetworze twoje pytanie, przypisze do niego unikatowy ID" +
					"oraz wyśle go do innych anonimów. Dziennie możesz zapytać nie więcej niż 3 pytania.\n\n" + "Jeżeli chcesz odpowiedzieć na pytanie - " +
					"użyj przycisku \"Dać odpowiedź\" lub komendy /answer_question, po czym bot poprosi ci o podanie numeru ID tego pytanie, na które chcesz odpowiedzieć (numer ID każdego pytania jest pisany w końcu tego pytania).\n\n" +
					"Dalej podaj swoją odpowiedź, która zostanie wysłana tylko i wyłącznie do człowieka, który te pytanie zapytał.\n\n" +
					"Jeżeli masz pytania lub propozycje - pisz do mnie @YUART. Również możesz zajść na mojego Patreona (https://www.patreon.com/artemkakun) i kupić mi kebaba :)\n\n" +
					"Miłego użytkowania! :)\n"
				msg.ReplyMarkup = numericKeyboard
			} else {
				msg.Text = "Cześć. Tutaj możesz postawić anonimowe pytanie lub odpowiedzieć na inne anonimowe pytania.\n\n" +
					"Żeby zadać pytanie - użyj przycisku \"Zapytać\" lub komendy /new_question, następnie podaj tekst pytania. Po wysłaniu pytania do bota, program przetworze twoje pytanie, przypisze do niego unikatowy ID" +
					"oraz wyśle go do innych anonimów. Dziennie możesz zapytać nie więcej niż 3 pytania.\n\n" + "Jeżeli chcesz odpowiedzieć na pytanie - " +
					"użyj przycisku \"Dać odpowiedź\" lub komendy /answer_question, po czym bot poprosi ci o podanie numeru ID tego pytanie, na które chcesz odpowiedzieć (numer ID każdego pytania jest pisany w końcu tego pytania).\n\n" +
					"Dalej podaj swoją odpowiedź, która zostanie wysłana tylko i wyłącznie do człowieka, który te pytanie zapytał.\n\n" +
					"Jeżeli masz pytania lub propozycje - pisz do mnie @YUART. Również możesz zajść na mojego Patreona (https://www.patreon.com/artemkakun) i kupić mi kebaba :)\n\n" +
					"Miłego użytkowania! :)\n"
				msg.ReplyMarkup = numericKeyboard
			}
		case "new_question":
			if CheckReg(update.Message.From.ID, database, my_bot) {
				if CheckAnswr(update.Message.From.ID, database, my_bot) == 0 {
					if GetQuest(update.Message.From.ID, database, my_bot) != "" {
						if GetQuestCount(update.Message.From.ID, database, my_bot) > 0 {
							msg.Text = "Teraz podaj pytanie"
							StartQuest(update.Message.From.ID, database, my_bot)
							SetAsk(update.Message.From.ID, 1, database, my_bot)
							MinusQuestCount(update.Message.From.ID, database, my_bot)
						} else {
							msg.Text = "Dzisiejszy limit pytań został przekroczony!"
						}
					} else {
						msg.Text = "Musisz dokończyć pytanie!"
					}
				} else {
					msg.Text = "Musisz dokończyć odpowiedź!"
				}
			} else {
				msg.Text = "Musisz się zarejestrować. Użyj komendy /start."
			}
		case "answer_question":
			if CheckReg(update.Message.From.ID, database, my_bot) {
				if !CheckAsk(update.Message.From.ID, database, my_bot) {
					msg.Text = "Napisz ID pytania, na które chcesz odpowiedzieć"
					SetAnswer(update.Message.From.ID, 1, database, my_bot)
				} else {
					msg.Text = "Musisz dokończyć pytanie!"
				}
			} else {
				msg.Text = "Musisz się zarejestrować. Użyj komendy /start."
			}
		}

		_, err := my_bot.Send(msg)
		if err != nil {
			ErrorCatch(err.Error(), my_bot)
		}
	}
}

func DBStart() *sql.DB {
	my_db, err := sql.Open("mysql", "root:11hahozeGood!@/anonyquestions")
	if err != nil {
		log.Panic(err)
	} else {
		err = my_db.Ping()
		if err != nil {
			log.Panic(err)
		}
	}
	return my_db
}
func AddUser(user_id int, my_db *sql.DB, my_bot *tgbotapi.BotAPI) {
	stmtIns, err := my_db.Prepare("INSERT INTO users VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	_, err = stmtIns.Exec(user_id, 0, 0, 0, 3)
	if err != nil {ErrorCatch(err.Error(), my_bot)

		panic(err.Error())
	}

	err = stmtIns.Close()
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}
}
func CheckReg(user_id int, my_db *sql.DB, my_bot *tgbotapi.BotAPI) bool {
	stmtOut, err := my_db.Prepare("SELECT user_id FROM users WHERE user_id = ?")
	if err != nil {
		panic(err.Error())
	}

	var is_reg int
	err = stmtOut.QueryRow(user_id).Scan(&is_reg)
	if err != nil {
		err = stmtOut.Close()
		if err != nil {
			ErrorCatch(err.Error(), my_bot)
			panic(err.Error())
		}
		return false
	}

	err = stmtOut.Close()
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	if is_reg != 0 {
		return true
	} else {
		return false
	}
}
func StartQuest(user_id int, my_db *sql.DB, my_bot *tgbotapi.BotAPI) {
	stmtIns, err := my_db.Prepare("INSERT INTO question_info (user_id, question, is_shared) VALUES (?, ?, ?)")
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	_, err = stmtIns.Exec(user_id, "", 0)
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	err = stmtIns.Close()
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}
}
func NewQuest(user_id int, text string, my_db *sql.DB, my_bot *tgbotapi.BotAPI) {
	stmtIns, err := my_db.Prepare("UPDATE question_info SET question = ? WHERE user_id = ? AND is_shared = 0")
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	_, err = stmtIns.Exec(text, user_id)
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	err = stmtIns.Close()
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}
}
func ShareQuest(user_id int, my_db *sql.DB, my_bot *tgbotapi.BotAPI) {
	stmtOut, err := my_db.Query("SELECT user_id FROM users")
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	var one_user int
	next_quest_id := GetQuestID(user_id, my_db, my_bot)
	for stmtOut.Next() {
		err = stmtOut.Scan(&one_user)
		if err != nil {
			err = stmtOut.Close()
			if err != nil {
				ErrorCatch(err.Error(), my_bot)
				panic(err.Error())
			}
		}
		msg := tgbotapi.NewMessage(int64(one_user), "")
		full_quest := GetQuest(user_id, my_db, my_bot) + fmt.Sprintf("\n ID pytania - %v", next_quest_id)
		msg.Text = full_quest
		_, err := my_bot.Send(msg)
		if err != nil {
			ErrorCatch(err.Error(), my_bot)
		}
	}

	SetShared(next_quest_id, my_db, my_bot)

	err = stmtOut.Close()
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}
}
func GetQuest(user_id int, my_db *sql.DB, my_bot *tgbotapi.BotAPI) string {
	stmtOut, err := my_db.Prepare("SELECT question FROM question_info WHERE user_id = ? AND is_shared = 0")
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	var question string
	err = stmtOut.QueryRow(user_id).Scan(&question)
	if err != nil {
		err = stmtOut.Close()
		if err != nil {
			ErrorCatch(err.Error(), my_bot)
			panic(err.Error())
		}
		return "ok"
	}

	err = stmtOut.Close()
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	return question
}
func GetQuestID(user_id int, my_db *sql.DB, my_bot *tgbotapi.BotAPI) int {
	stmtOut, err := my_db.Prepare("SELECT quest_id FROM question_info WHERE user_id = ? AND is_shared = 0")
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	var id int
	err = stmtOut.QueryRow(user_id).Scan(&id)
	if err != nil {
		err = stmtOut.Close()
		if err != nil {
			ErrorCatch(err.Error(), my_bot)
			panic(err.Error())
		}
		return -1
	}

	err = stmtOut.Close()
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	return id
}
func SetShared(quest_id int, my_db *sql.DB, my_bot *tgbotapi.BotAPI) {
	stmtIns, err := my_db.Prepare("UPDATE question_info SET is_shared = 1 WHERE quest_id = ?")
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	_, err = stmtIns.Exec(quest_id)
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	err = stmtIns.Close()
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}
}
func SetAnswer(user_id int, answer_type int, my_db *sql.DB, my_bot *tgbotapi.BotAPI) {
	stmtIns, err := my_db.Prepare("UPDATE users SET is_answering = ? WHERE user_id = ?")
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	_, err = stmtIns.Exec(answer_type, user_id)
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	err = stmtIns.Close()
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}
}
func SetAsk(user_id int, answer_type int, my_db *sql.DB, my_bot *tgbotapi.BotAPI) {
	stmtIns, err := my_db.Prepare("UPDATE users SET is_asking = ? WHERE user_id = ?")
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	_, err = stmtIns.Exec(answer_type, user_id)
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	err = stmtIns.Close()
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}
}
func CheckAsk(user_id int, my_db *sql.DB, my_bot *tgbotapi.BotAPI) bool {
	stmtOut, err := my_db.Prepare("SELECT is_asking FROM users WHERE user_id = ?")
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	var is_reg int
	err = stmtOut.QueryRow(user_id).Scan(&is_reg)
	if err != nil {
		err = stmtOut.Close()
		if err != nil {
			ErrorCatch(err.Error(), my_bot)
			panic(err.Error())
		}
		return false
	}

	err = stmtOut.Close()
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	if is_reg == 1 {
		return true
	} else {
		return false
	}

}
func CheckAnswr(user_id int, my_db *sql.DB, my_bot *tgbotapi.BotAPI) int {
	stmtOut, err := my_db.Prepare("SELECT is_answering FROM users WHERE user_id = ?")
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	var is_reg int
	err = stmtOut.QueryRow(user_id).Scan(&is_reg)
	if err != nil {
		err = stmtOut.Close()
		if err != nil {
			ErrorCatch(err.Error(), my_bot)
			panic(err.Error())
		}
		return 0
	}

	err = stmtOut.Close()
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	return is_reg
}
func SetQuestID(user_id int, answer_type int, my_db *sql.DB, my_bot *tgbotapi.BotAPI) {
	stmtIns, err := my_db.Prepare("UPDATE users SET want_id = ? WHERE user_id = ?")
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	_, err = stmtIns.Exec(answer_type, user_id)
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	err = stmtIns.Close()
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}
}
func GetMaxID(my_db *sql.DB, my_bot *tgbotapi.BotAPI) int {
	stmtOut, err := my_db.Prepare("SELECT MAX(quest_id) FROM question_info")
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	var id int
	err = stmtOut.QueryRow().Scan(&id)
	if err != nil {
		err = stmtOut.Close()
		if err != nil {
			ErrorCatch(err.Error(), my_bot)
			panic(err.Error())
		}
		return -1
	}

	err = stmtOut.Close()
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	return id
}
func GetUserID(quest_id int, my_db *sql.DB, my_bot *tgbotapi.BotAPI) int {
	stmtOut, err := my_db.Prepare("SELECT user_id FROM question_info WHERE quest_id = ?")
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	var id int
	err = stmtOut.QueryRow(quest_id).Scan(&id)
	if err != nil {
		err = stmtOut.Close()
		if err != nil {
			ErrorCatch(err.Error(), my_bot)
			panic(err.Error())
		}
		return -1
	}

	err = stmtOut.Close()
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	return id
}
func GetWantID(user_id int, my_db *sql.DB, my_bot *tgbotapi.BotAPI) int {
	stmtOut, err := my_db.Prepare("SELECT want_id FROM users WHERE user_id = ?")
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	var id int
	err = stmtOut.QueryRow(user_id).Scan(&id)
	if err != nil {
		err = stmtOut.Close()
		if err != nil {
			ErrorCatch(err.Error(), my_bot)
			panic(err.Error())
		}
		return -1
	}

	err = stmtOut.Close()
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	return id
}

func GetQuestCount(user_id int, my_db *sql.DB, my_bot *tgbotapi.BotAPI) int {
	stmtOut, err := my_db.Prepare("SELECT quest_count FROM users WHERE user_id = ?")
	if err != nil {
		panic(err.Error())
	}

	var count int
	err = stmtOut.QueryRow(user_id).Scan(&count)
	if err != nil {
		err = stmtOut.Close()
		if err != nil {
			ErrorCatch(err.Error(), my_bot)
			panic(err.Error())
		}
		return -1
	}

	err = stmtOut.Close()
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	return count
}

func MinusQuestCount(user_id int, my_db *sql.DB, my_bot *tgbotapi.BotAPI) {
	actual_quest_count := GetQuestCount(user_id, my_db, my_bot)

	stmtIns, err := my_db.Prepare("UPDATE users SET quest_count = ? WHERE user_id = ?")
	if err != nil {
		panic(err.Error())
	}

	_, err = stmtIns.Exec(actual_quest_count - 1, user_id)
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	err = stmtIns.Close()
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}
}

func UpdateQuestCount(my_db *sql.DB, my_bot *tgbotapi.BotAPI) {
	stmtIns, err := my_db.Prepare("UPDATE users SET quest_count = ?")
	if err != nil {
		panic(err.Error())
	}

	_, err = stmtIns.Exec(3)
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}

	err = stmtIns.Close()
	if err != nil {
		ErrorCatch(err.Error(), my_bot)
		panic(err.Error())
	}
}

func QuestUpdater(my_db *sql.DB, my_bot *tgbotapi.BotAPI) {
	for true {
		if time.Now().Hour() == 0 && time.Now().Minute() == 0 && time.Now().Second() == 0 {
			UpdateQuestCount(my_db, my_bot)
		}
		amt := time.Duration(1000)
		time.Sleep(time.Millisecond * amt)
	}
}

func ErrorCatch(err string, my_bot *tgbotapi.BotAPI) {
	msg := tgbotapi.NewMessage(ADMIN, err)
	my_bot.Send(msg)
}