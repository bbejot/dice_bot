package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"

	//"bbejot_claymctavish/dice_bot/utils"
	"github.com/bwmarrin/discordgo"
)

type Command struct {
	Name   string
	Action func(*discordgo.Session, *discordgo.MessageCreate, []string)
}

var (
	BotToken = flag.String("token", "", "Bot access token")
	Prefix   = "&"
	Commands = []*Command{}
)

func roll_action(session *discordgo.Session, mc *discordgo.MessageCreate, params []string) {
	fmt.Println("in roll_action")
	fmt.Println(strings.Join(params, ", "))
	regstr := "^(\\d*)d(\\d+)((kh)(\\d+))?((kl)(\\d+))?$"
	re, err := regexp.Compile(regstr)
	if err != nil {
		log.Fatalf("Regex failed to compile: %v, %s", err, regstr)
		return
	}
	matches := []string{}
	for _, m := range re.FindStringSubmatch(params[0]) {
		if m == "" {
			continue
		}
		matches = append(matches, m)
	}
	fmt.Println(matches)

	if len(matches) < 3 {
		session.ChannelMessageSend(mc.ChannelID, "malformed dice roll")
		return
	}

	num_dice := 1
	if matches[1] != "" {
		num_dice, err = strconv.Atoi(matches[1])
		if err != nil {
			log.Fatalf("Int parse error: %v, %s", err, regstr)
			return
		}
	}
	die_size, err := strconv.Atoi(matches[2])
	if err != nil {
		log.Fatalf("Int parse error: %v, %s", err, regstr)
		return
	}
	if die_size <= 0 || num_dice <= 0 || num_dice > 1000 {
		session.ChannelMessageSend(mc.ChannelID, "improper die format")
		return
	}

	keep_highest := 0
	keep_lowest := 0

	for idx := 3; idx < len(matches); {
		if idx+2 < len(matches) {
			// could be kh or kl
			switch matches[idx+1] {
			case "kh":
				keep_highest, err = strconv.Atoi(matches[idx+2])
				if err != nil {
					log.Fatalf("Int parse error: %v, %s", err, regstr)
					return
				}
			case "kl":
				keep_lowest, err = strconv.Atoi(matches[idx+2])
				if err != nil {
					log.Fatalf("Int parse error: %v, %s", err, regstr)
					return
				}
			default:
				session.ChannelMessageSend(mc.ChannelID, "malformed dice roll modifier")
				return
			}
			idx += 3
		} else {
			session.ChannelMessageSend(mc.ChannelID, "malformed dice roll parse")
			return
		}
	}

	rolls := []int{}
	for i := 0; i < num_dice; i++ {
		rolls = append(rolls, rand.Intn(die_size)+1)
	}

	sort.Ints(rolls)

	if keep_highest > 0 {
		if keep_highest > len(rolls) {
			session.ChannelMessageSend(mc.ChannelID, "malformed dice roll")
			return
		}
		rolls = rolls[len(rolls)-keep_highest:]
	}

	if keep_lowest > 0 {
		if keep_lowest > len(rolls) {
			session.ChannelMessageSend(mc.ChannelID, "malformed dice roll")
			return
		}
		rolls = rolls[:keep_lowest]
	}

	fmt.Println(len(rolls))

	sum := 0
	for _, v := range rolls {
		sum += v
	}

	fmt.Println(sum)
	session.ChannelMessageSend(mc.ChannelID, fmt.Sprintf("roll: %s\nresult: %d", matches[0], sum))
}

//func (something *Something) Write(p1 string) {
//  something.Name = p1
//}

func main() {
	flag.Parse()
	fmt.Printf("Initializing Bot...")

	discord, err := discordgo.New("Bot " + *BotToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
		return
	}

	discord.AddHandler(messageCreate)
	discord.Identify.Intents = discordgo.IntentsGuildMessages

	err = discord.Open()
	if err != nil {
		log.Fatalf("Error opening connection,", err)
		return
	}

	//GuildID := ""

	//cmd = *discordgo.ApplicationCommand{
	//        Name: "basic-command",
	//        Description: "Basic command",
	//}

	//need to add a command handler function thingy

	//_, err = discord.ApplicationCommandCreate(discord.State.User.ID, *GuildID, cmd)
	//if err != nil {
	//  log.Fatalf("Error adding command,", err)
	//  return
	//}

	cmd := Command{
		Name:   "r",
		Action: roll_action,
	}

	Commands = append(Commands, &cmd)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	discord.Close()
	fmt.Printf("Exiting...!\n")
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if !strings.HasPrefix(m.Content, Prefix) {
		return
	}

	params_str := m.Content[len(Prefix):]
	var params []string
	for _, s := range strings.Split(params_str, " ") {
		if s != "" {
			params = append(params, s)
		}
	}

	if len(params) == 0 {
		return
	}

	for _, cmd := range Commands {
		if params[0] != cmd.Name {
			continue
		}
		cmd.Action(s, m, params[1:])
	}
	//if m.Content == "&roll" {
	//  s.ChannelMessageSend(m.ChannelID, "test")
	//}
}
