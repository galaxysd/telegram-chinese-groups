package main

import (
	"log"
	"strconv"
	"time"

	"github.com/Syfaro/telegram-bot-api"
	"github.com/kylelemons/go-gypsy/yaml"
	"gopkg.in/redis.v3"
)

type Updater struct {
	redis  *redis.Client
	bot    *tgbotapi.BotAPI
	update tgbotapi.Update
	conf   *yaml.File
}

func (u *Updater) BotReply(msgText string) {
	chatIDStr := strconv.Itoa(u.update.Message.Chat.ID)
	enableGroupLimit, _ := u.conf.GetBool("enableGroupLimit")
	limitInterval, _ := u.conf.Get("limitInterval")
	limitTimes, _ := u.conf.GetInt("limitTimes")

	if enableGroupLimit && u.update.Message.Chat.ID < 0 {
		if u.redis.Exists(chatIDStr).Val() {
			u.redis.Incr(chatIDStr)
			counter, _ := u.redis.Get(chatIDStr).Int64()
			if counter >= limitTimes {
				log.Println("--- " + u.update.Message.Chat.Title + " --- " + "防刷屏 ---")
				msg := tgbotapi.NewMessage(u.update.Message.Chat.ID,
					"刷屏是坏孩纸~！\n聪明宝宝是会跟奴家私聊的哟😊\n@"+u.bot.Self.UserName)
				msg.ReplyToMessageID = u.update.Message.MessageID
				u.bot.SendMessage(msg)
				return
			}
		} else {
			expire, _ := time.ParseDuration(limitInterval)
			u.redis.Set(chatIDStr, "0", expire)
		}
	}

	msg := tgbotapi.NewMessage(u.update.Message.Chat.ID, msgText)
	u.bot.SendMessage(msg)
	return
}

func (u *Updater) Subscribe() {
	chatIDStr := strconv.Itoa(u.update.Message.Chat.ID)
	u.redis.HSet("tgSubscribe", chatIDStr, strconv.FormatBool(true))
	u.redis.HIncrBy("tgSubscribeTimes", chatIDStr, 1)
	msg := tgbotapi.NewMessage(u.update.Message.Chat.ID,
		"订阅成功\n以后奴家知道新的群组的话，会第一时间告诉你哟😊\n(订阅仅对当前会话有效)")
	u.bot.SendMessage(msg)
}

func (u *Updater) UnSubscribe() {
	chatIDStr := strconv.Itoa(u.update.Message.Chat.ID)
	var msg tgbotapi.MessageConfig
	if u.redis.HExists("tgSubscribe", chatIDStr).Val() {
		u.redis.HDel("tgSubscribe", chatIDStr)
		times, _ := u.redis.HIncrBy("tgSubscribeTimes", chatIDStr, 1).Result()
		if times > 5 {
			msg = tgbotapi.NewMessage(u.update.Message.Chat.ID,
				"订了退，退了订，你烦不烦嘛！！！⊂彡☆))∀`)`")
			u.redis.HDel("tgSubscribeTimes", chatIDStr)
		} else {
			msg = tgbotapi.NewMessage(u.update.Message.Chat.ID,
				"好伤心，退订了就不能愉快的玩耍了呢😭")
		}
	} else {
		msg = tgbotapi.NewMessage(u.update.Message.Chat.ID,
			"你都还没订阅，让人家怎么退订嘛！o(≧口≦)o")
	}
	u.bot.SendMessage(msg)
}

func (u *Updater) Broadcast(msgText string) {
	master, _ := u.conf.Get("master")
	if u.update.Message.Chat.UserName == master &&
		u.redis.Exists("tgSubscribe").Val() {

		subStates := u.redis.HGetAllMap("tgSubscribe").Val()

		chs := make(map[string]chan bool)
		for k, v := range subStates {
			chatid, _ := strconv.Atoi(k)
			subState, _ := strconv.ParseBool(v)
			chs[k] = make(chan bool)

			if subState {
				log.Printf("sending boardcast to %d ...", chatid)
				msg := tgbotapi.NewMessage(chatid, msgText)
				go func(ch chan bool) {
					u.bot.SendMessage(msg)
					ch <- true
				}(chs[k])
			}
		}

		for k, v := range chs {
			if <-v {
				log.Println(k + " --- done")
			}
		}
	}
}
