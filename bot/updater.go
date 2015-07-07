package main

import (
	"log"
	"strconv"
	"strings"
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

func (u *Updater) Start() {
	if u.update.Message.Chat.ID < 0 && !u.redis.HExists("tgGroups",
		strconv.Itoa(u.update.Message.Chat.ID)).Val() {
		u.redis.HSet("tgGroups",
			strconv.Itoa(u.update.Message.Chat.ID), u.update.Message.Chat.Title)
		log.Printf("%d --- %s join", u.update.Message.Chat.ID, u.update.Message.Chat.Title)
	}
	u.BotReply(YamlList2String(u.conf, "help"))
}

func (u *Updater) ListGroups() {
	if u.update.Message.Chat.ID > 0 {
		groups := u.redis.HGetAllMap("tgGroups").Val()
		var messages []string
		for groupID, groupTitle := range groups {
			masters := u.redis.SMembers("tgGroupMasters:" + groupID).Val()
			mastersStr := strings.Join(masters, ",")
			messages = append(messages, groupID+" --- "+groupTitle+" [ "+mastersStr+" ] ")
		}
		message := strings.Join(messages, "\n")
		msg := tgbotapi.NewMessage(u.update.Message.Chat.ID, message)
		u.bot.SendMessage(msg)
	}
}

func (u *Updater) AddMaster(chatid, master string) {
	superMaster, _ := u.conf.Get("master")
	if u.update.Message.Chat.UserName == superMaster {
		log.Println("add master @" + master + " to " + chatid)
		u.redis.SAdd("tgGroupMasters:"+chatid, master)
	}
	masters := u.redis.SMembers("tgGroupMasters:" + chatid).Val()
	mastersStr := strings.Join(masters, ",")
	msg := tgbotapi.NewMessage(u.update.Message.Chat.ID, "Masters Update:[ "+mastersStr+" ]")
	u.bot.SendMessage(msg)
}

func (u *Updater) RmMaster(chatid, master string) {
	superMaster, _ := u.conf.Get("master")
	if u.update.Message.Chat.UserName == superMaster {
		log.Println("remove master @" + master + " from " + chatid)
		u.redis.SRem("tgGroupMasters:"+chatid, master)
	}
	masters := u.redis.SMembers("tgGroupMasters:" + chatid).Val()
	mastersStr := strings.Join(masters, ",")
	msg := tgbotapi.NewMessage(u.update.Message.Chat.ID, "Masters Update:[ "+mastersStr+" ]")
	u.bot.SendMessage(msg)
}

func (u *Updater) SetRule(chatid, rule string) {
	superMaster, _ := u.conf.Get("master")
	if u.redis.SIsMember("tgGroupMasters:"+
		chatid, u.update.Message.Chat.UserName).Val() ||
		u.update.Message.Chat.UserName == superMaster {
		log.Println("setting rule " + rule + " to " + chatid)
		u.redis.Set("tgGroupRule:"+chatid, rule, -1)
		msg := tgbotapi.NewMessage(u.update.Message.Chat.ID, "Rule Update!")
		u.bot.SendMessage(msg)
	}
}

func (u *Updater) Rule() {
	if u.redis.Exists("tgGroupRule:" +
		strconv.Itoa(u.update.Message.Chat.ID)).Val() {
		msg := tgbotapi.NewMessage(u.update.Message.Chat.ID,
			u.redis.Get("tgGroupRule:"+strconv.Itoa(u.update.Message.Chat.ID)).Val())
		u.bot.SendMessage(msg)
	} else {
		u.BotReply(YamlList2String(u.conf, "rules"))
	}
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
	isSubscribe, _ := strconv.ParseBool(u.redis.HGet("tgSubscribe", chatIDStr).Val())
	if isSubscribe {
		msg := tgbotapi.NewMessage(u.update.Message.Chat.ID,
			"已经订阅过，就不要重复订阅啦😘")
		u.bot.SendMessage(msg)
	} else {
		u.redis.HSet("tgSubscribe", chatIDStr, strconv.FormatBool(true))
		u.redis.HIncrBy("tgSubscribeTimes", chatIDStr, 1)
		msg := tgbotapi.NewMessage(u.update.Message.Chat.ID,
			"订阅成功\n以后奴家知道新的群组的话，会第一时间告诉你哟😊\n(订阅仅对当前会话有效)")
		u.bot.SendMessage(msg)
	}
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
