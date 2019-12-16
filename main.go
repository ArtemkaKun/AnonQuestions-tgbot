package main

import (
	"database/sql"
	"fmt"
	"github.com/Syfaro/telegram-bot-api"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"strconv"
)

func main() {
	bot := BotStart()
	my_db := DBStart()
	defer my_db.Close()
	BotUpdateLoop(bot, my_db)
}

var numericKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Ask a question"),
		tgbotapi.NewKeyboardButton("Answer the question"),
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
			case "Ask a question":
				msg := tgbotapi.NewMessage(int64(update.Message.From.ID), "")
				if CheckReg(update.Message.From.ID, database) {
					if GetQuest(update.Message.From.ID, database) != "" {
						msg.Text = "Teraz podaj pytanie"
						StartQuest(update.Message.From.ID, database)
						SetAsk(update.Message.From.ID,1, database)
					} else {
						msg.Text = "Podaj tekst pytania"
					}
				} else {
					msg.Text = "Musisz się zarejestrować"
				}
				my_bot.Send(msg)
				continue

			case "Answer the question":
				msg := tgbotapi.NewMessage(int64(update.Message.From.ID), "")

				if CheckReg(update.Message.From.ID, database) {
					msg.Text = "Napisz ID pytania, na które chcesz odpowiedzieć"
					SetAnswer(update.Message.From.ID, 1, database)
				} else {
					msg.Text = "Musisz się zarejestrować"
				}

				my_bot.Send(msg)
				continue
			}

			switch CheckAnswr(update.Message.From.ID, database) {
			case 1:
				msg := tgbotapi.NewMessage(int64(update.Message.From.ID), "")
				user_mes, _ := strconv.Atoi(update.Message.Text)
				if user_mes < 0 && user_mes > GetMaxID(database) {
					msg.Text = "Wybrany zły ID"
					my_bot.Send(msg)
					continue
				}
				SetQuestID(update.Message.From.ID, user_mes, database)
				msg.Text = "Napisz odpowiedź"
				SetAnswer(update.Message.From.ID, 2, database)
				my_bot.Send(msg)
				continue
			case 2:
				msg := tgbotapi.NewMessage(int64(update.Message.From.ID), "")
				msg.Text = "Odpowiedź wysłana"
				SetAnswer(update.Message.From.ID, 0, database)
				my_bot.Send(msg)

				msg = tgbotapi.NewMessage(int64(GetUserID(GetWantID(update.Message.From.ID, database), database)), "")
				msg.Text = update.Message.Text
				my_bot.Send(msg)
				continue
			}

			if CheckAsk(update.Message.From.ID, database) {
				msg := tgbotapi.NewMessage(int64(update.Message.From.ID), "")

				msg.Text = update.Message.Text
				if msg.Text != "" {
					NewQuest(update.Message.From.ID, update.Message.Text, database)
					ShareQuest(update.Message.From.ID, database, my_bot)
					SetAsk(update.Message.From.ID, 0, database)
					msg.Text = "Pytanie wysłane"
					my_bot.Send(msg)
				} else {
					msg.Text = "Nie możesz wysyłać nic oprócz tekstu!"
					my_bot.Send(msg)
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
			if !CheckReg(update.Message.From.ID, database) {
				AddUser(update.Message.From.ID, database)
				msg.Text = "Cześć. Tutaj możesz anonimowo zapytać co chcesz lub odpowiedzieć na inne anonimowe pytania."
				msg.ReplyMarkup = numericKeyboard
			} else {
				msg.Text = "Już jesteś"
				msg.ReplyMarkup = numericKeyboard
			}
		case "new_question":
			if CheckReg(update.Message.From.ID, database) {
				if GetQuest(update.Message.From.ID, database) != "" {
					msg.Text = "Teraz podaj pytanie"
					StartQuest(update.Message.From.ID, database)
					SetAsk(update.Message.From.ID,1, database)
				} else {
					msg.Text = "Podaj tekst pytania"
				}
			} else {
				msg.Text = "Musisz się zarejestrować"
			}
		case "answer_question":
			if CheckReg(update.Message.From.ID, database) {
				msg.Text = "Napisz ID pytania, na które chcesz odpowiedzieć"
				SetAnswer(update.Message.From.ID, 1, database)
			} else {
				msg.Text = "Musisz się zarejestrować"
			}
		}

		my_bot.Send(msg)
	}
}

func DBStart() *sql.DB {
	my_db, err := sql.Open("mysql", "root:1337@/anonyquestions")
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
func AddUser(user_id int, my_db *sql.DB) {
	stmtIns, err := my_db.Prepare("INSERT INTO users VALUES (?, ?, ?, ?)")
	if err != nil {
		panic(err.Error())
	}

	_, err = stmtIns.Exec(user_id, 0, 0, 0)
	if err != nil {
		panic(err.Error())
	}

	err = stmtIns.Close()
	if err != nil {
		panic(err.Error())
	}
}
func CheckReg(user_id int, my_db *sql.DB) bool {
	stmtOut, err := my_db.Prepare("SELECT user_id FROM users WHERE user_id = ?")
	if err != nil {
		panic(err.Error())
	}

	var is_reg int
	err = stmtOut.QueryRow(user_id).Scan(&is_reg)
	if err != nil {
		err = stmtOut.Close()
		if err != nil {
			panic(err.Error())
		}
		return false
	}

	err = stmtOut.Close()
	if err != nil {
		panic(err.Error())
	}

	if is_reg != 0 {
		return true
	} else {
		return false
	}
}
func StartQuest(user_id int, my_db *sql.DB) {
	stmtIns, err := my_db.Prepare("INSERT INTO question_info (user_id, question, is_shared) VALUES (?, ?, ?)")
	if err != nil {
		panic(err.Error())
	}

	_, err = stmtIns.Exec(user_id, "", 0)
	if err != nil {
		panic(err.Error())
	}

	err = stmtIns.Close()
	if err != nil {
		panic(err.Error())
	}
}
func NewQuest(user_id int, text string, my_db *sql.DB) {
	stmtIns, err := my_db.Prepare("UPDATE question_info SET question = ? WHERE user_id = ?")
	if err != nil {
		panic(err.Error())
	}

	_, err = stmtIns.Exec(text, user_id)
	if err != nil {
		panic(err.Error())
	}

	err = stmtIns.Close()
	if err != nil {
		panic(err.Error())
	}
}
func ShareQuest(user_id int, my_db *sql.DB, my_bot *tgbotapi.BotAPI) {
	stmtOut, err := my_db.Query("SELECT user_id FROM users")
	if err != nil {
		panic(err.Error())
	}

	var one_user int
	next_quest_id := GetQuestID(user_id, my_db)
	for stmtOut.Next() {
		err = stmtOut.Scan(&one_user)
		if err != nil {
			err = stmtOut.Close()
			if err != nil {
				panic(err.Error())
			}
		}
		msg := tgbotapi.NewMessage(int64(one_user), "")
		full_quest := GetQuest(user_id, my_db) + fmt.Sprintf("\n Question ID - %v", next_quest_id)
		msg.Text = full_quest
		my_bot.Send(msg)
	}

	SetShared(next_quest_id, my_db)

	err = stmtOut.Close()
	if err != nil {
		panic(err.Error())
	}
}
func GetQuest(user_id int, my_db *sql.DB) string {
	stmtOut, err := my_db.Prepare("SELECT question FROM question_info WHERE user_id = ? AND is_shared = 0")
	if err != nil {
		panic(err.Error())
	}

	var question string
	err = stmtOut.QueryRow(user_id).Scan(&question)
	if err != nil {
		err = stmtOut.Close()
		if err != nil {
			panic(err.Error())
		}
		return "ok"
	}

	err = stmtOut.Close()
	if err != nil {
		panic(err.Error())
	}

	return question
}
func GetQuestID(user_id int, my_db *sql.DB) int {
	stmtOut, err := my_db.Prepare("SELECT quest_id FROM question_info WHERE user_id = ?")
	if err != nil {
		panic(err.Error())
	}

	var id int
	err = stmtOut.QueryRow(user_id).Scan(&id)
	if err != nil {
		err = stmtOut.Close()
		if err != nil {
			panic(err.Error())
		}
		return -1
	}

	err = stmtOut.Close()
	if err != nil {
		panic(err.Error())
	}

	return id
}
func SetShared(quest_id int, my_db *sql.DB) {
	stmtIns, err := my_db.Prepare("UPDATE question_info SET is_shared = 1 WHERE quest_id = ?")
	if err != nil {
		panic(err.Error())
	}

	_, err = stmtIns.Exec(quest_id)
	if err != nil {
		panic(err.Error())
	}

	err = stmtIns.Close()
	if err != nil {
		panic(err.Error())
	}
}
func SetAnswer(user_id int, answer_type int, my_db *sql.DB) {
	stmtIns, err := my_db.Prepare("UPDATE users SET is_answering = ? WHERE user_id = ?")
	if err != nil {
		panic(err.Error())
	}

	_, err = stmtIns.Exec(answer_type, user_id)
	if err != nil {
		panic(err.Error())
	}

	err = stmtIns.Close()
	if err != nil {
		panic(err.Error())
	}
}
func SetAsk(user_id int, answer_type int, my_db *sql.DB) {
	stmtIns, err := my_db.Prepare("UPDATE users SET is_asking = ? WHERE user_id = ?")
	if err != nil {
		panic(err.Error())
	}

	_, err = stmtIns.Exec(answer_type, user_id)
	if err != nil {
		panic(err.Error())
	}

	err = stmtIns.Close()
	if err != nil {
		panic(err.Error())
	}
}
func CheckAsk(user_id int, my_db *sql.DB) bool {
	stmtOut, err := my_db.Prepare("SELECT is_asking FROM users WHERE user_id = ?")
	if err != nil {
		panic(err.Error())
	}

	var is_reg int
	err = stmtOut.QueryRow(user_id).Scan(&is_reg)
	if err != nil {
		err = stmtOut.Close()
		if err != nil {
			panic(err.Error())
		}
		return false
	}

	err = stmtOut.Close()
	if err != nil {
		panic(err.Error())
	}

	if is_reg == 1 {
		return true
	} else {
		return false
	}

}
func CheckAnswr(user_id int, my_db *sql.DB) int {
	stmtOut, err := my_db.Prepare("SELECT is_answering FROM users WHERE user_id = ?")
	if err != nil {
		panic(err.Error())
	}

	var is_reg int
	err = stmtOut.QueryRow(user_id).Scan(&is_reg)
	if err != nil {
		err = stmtOut.Close()
		if err != nil {
			panic(err.Error())
		}
		return 0
	}

	err = stmtOut.Close()
	if err != nil {
		panic(err.Error())
	}

	return is_reg
}
func SetQuestID(user_id int, answer_type int, my_db *sql.DB) {
	stmtIns, err := my_db.Prepare("UPDATE users SET want_id = ? WHERE user_id = ?")
	if err != nil {
		panic(err.Error())
	}

	_, err = stmtIns.Exec(answer_type, user_id)
	if err != nil {
		panic(err.Error())
	}

	err = stmtIns.Close()
	if err != nil {
		panic(err.Error())
	}
}
func GetMaxID(my_db *sql.DB) int {
	stmtOut, err := my_db.Prepare("SELECT MAX(quest_id) FROM question_info")
	if err != nil {
		panic(err.Error())
	}

	var id int
	err = stmtOut.QueryRow().Scan(&id)
	if err != nil {
		err = stmtOut.Close()
		if err != nil {
			panic(err.Error())
		}
		return -1
	}

	err = stmtOut.Close()
	if err != nil {
		panic(err.Error())
	}

	return id
}
func GetUserID(quest_id int, my_db *sql.DB) int {
	stmtOut, err := my_db.Prepare("SELECT user_id FROM question_info WHERE quest_id = ?")
	if err != nil {
		panic(err.Error())
	}

	var id int
	err = stmtOut.QueryRow(quest_id).Scan(&id)
	if err != nil {
		err = stmtOut.Close()
		if err != nil {
			panic(err.Error())
		}
		return -1
	}

	err = stmtOut.Close()
	if err != nil {
		panic(err.Error())
	}

	return id
}
func GetWantID(user_id int, my_db *sql.DB) int {
	stmtOut, err := my_db.Prepare("SELECT want_id FROM users WHERE user_id = ?")
	if err != nil {
		panic(err.Error())
	}

	var id int
	err = stmtOut.QueryRow(user_id).Scan(&id)
	if err != nil {
		err = stmtOut.Close()
		if err != nil {
			panic(err.Error())
		}
		return -1
	}

	err = stmtOut.Close()
	if err != nil {
		panic(err.Error())
	}

	return id
}