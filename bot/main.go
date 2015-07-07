package main

import (
	"log"
	"strconv"
	"strings"

	"gopkg.in/redis.v3"

	"github.com/Syfaro/telegram-bot-api"
	"github.com/kylelemons/go-gypsy/yaml"
)

func main() {
	rc := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rc.Close()

	conf, err := yaml.ReadFile("botconf.yaml")
	if err != nil {
		log.Panic(err)
	}

	botapi, _ := conf.Get("botapi")

	bot, err := tgbotapi.NewBotAPI(botapi)
	if err != nil {
		log.Panic(err)
	}

	botname := bot.Self.UserName

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.UpdatesChan(u)

	for update := range updates {

		log.Printf("[%d](%s) -- [%s] -- %s",
			update.Message.Chat.ID, update.Message.Chat.Title,
			update.Message.From.UserName, update.Message.Text,
		)

		u := Updater{
			redis:  rc,
			bot:    bot,
			update: update,
			conf:   conf,
		}

		switch update.Message.Text {

		case "/help", "/start", "/help@" + botname, "/start@" + botname:
			go u.Start()

		case "/rules", "/rules@" + botname:
			go u.Rule()

		case "/about", "/about@" + botname:
			go u.BotReply(YamlList2String(conf, "about"))

		case "/linux", "/linux@" + botname:
			go u.BotReply(YamlList2String(conf, "Linux"))

		case "/programming", "/programming@" + botname:
			go u.BotReply(YamlList2String(conf, "Programming"))

		case "/software", "/software@" + botname:
			go u.BotReply(YamlList2String(conf, "Software"))

		case "/media", "/media@" + botname:
			go u.BotReply(YamlList2String(conf, "影音"))

		case "/sci_fi", "/sci_fi@" + botname:
			go u.BotReply(YamlList2String(conf, "科幻"))

		case "/acg", "/acg@" + botname:
			go u.BotReply(YamlList2String(conf, "ACG"))

		case "/it", "/it@" + botname:
			go u.BotReply(YamlList2String(conf, "IT"))

		case "/free_chat", "/free_chat@" + botname:
			go u.BotReply(YamlList2String(conf, "闲聊"))

		case "/resources", "/resources@" + botname:
			go u.BotReply(YamlList2String(conf, "资源"))

		case "/same_city", "/same_city@" + botname:
			go u.BotReply(YamlList2String(conf, "同城"))

		case "/others", "/others@" + botname:
			go u.BotReply(YamlList2String(conf, "Others"))

		case "/other_resources", "/other_resources@" + botname:
			go u.BotReply(YamlList2String(conf, "其他资源"))

		case "/subscribe", "/subscribe@" + botname:
			go u.Subscribe()

		case "/unsubscribe", "/unsubscribe@" + botname:
			go u.UnSubscribe()

		case "/list":
			go u.ListGroups()

		default:
			s := strings.Split(update.Message.Text, " ")
			if len(s) >= 2 && s[0] == "/broadcast" {
				msg := strings.Join(s[1:], " ")
				go u.Broadcast(msg)
			} else if len(s) >= 3 && s[0] == "/addmaster" {
				go u.AddMaster(s[1], s[2])
			} else if len(s) >= 3 && s[0] == "/rmmaster" {
				go u.RmMaster(s[1], s[2])
			} else if len(s) >= 3 && s[0] == "/setrule" {
				rule := strings.Join(s[2:], " ")
				go u.SetRule(s[1], rule)
			}
		}
	}
}

func YamlList2String(config *yaml.File, text string) string {
	count, err := config.Count(text)
	if err != nil {
		log.Println(err)
		return ""
	}

	var resultGroup []string
	for i := 0; i < count; i++ {
		v, err := config.Get(text + "[" + strconv.Itoa(i) + "]")
		if err != nil {
			log.Println(err)
			return ""
		}
		resultGroup = append(resultGroup, v)
	}

	result := strings.Join(resultGroup, "\n")
	result = strings.Replace(result, "\\n", "", -1)

	return result
}
